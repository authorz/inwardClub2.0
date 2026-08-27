package wallet

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/inwardclub/server/internal/modules/printer"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// sqlPointsRepository is the MySQL points write repository. Point savings wait
// for store review. Point withdrawals immediately debit the points wallet,
// append the ledger, record the approved withdrawal and queue the store receipt
// in one transaction. Both paths claim the idempotency key on their request row.
//
// The daily sign-in flow (RecordSignIn) awards points immediately: it writes a
// sign_in_records row and the matching wallet ledger credit in one transaction.
type sqlPointsRepository struct {
	db    *platdb.DB
	clock Clock
}

// NewPointsRepository builds the MySQL points repository. clock supplies the
// business-day "now" used to bucket the sign-in calendar day.
func NewPointsRepository(db *platdb.DB, clock Clock) PointsRepository {
	return &sqlPointsRepository{db: db, clock: clock}
}

// GetSignInStatus returns the member's effective streak and today's configured
// reward without writing a sign-in record.
func (r *sqlPointsRepository) GetSignInStatus(ctx context.Context, memberID int64) (SignInStatus, error) {
	ladder := r.signInLadder(ctx)
	today := r.clock.Now()
	status := SignInStatus{
		Date:         today.Format("2006-01-02"),
		RewardPoints: pointsForStreak(ladder, 1),
		DailyRewards: append([]int64(nil), ladder...),
	}

	var prevDate time.Time
	var prevStreak int
	var prevPoints int64
	const q = `SELECT sign_date, streak_days, points_awarded FROM sign_in_records
		WHERE member_id = ? ORDER BY sign_date DESC LIMIT 1`
	switch err := r.db.QueryRowContext(ctx, q, memberID).Scan(&prevDate, &prevStreak, &prevPoints); {
	case errors.Is(err, sql.ErrNoRows):
		status.NextRewardPoints = pointsForStreak(ladder, 2)
		return status, nil
	case err != nil:
		return SignInStatus{}, apperr.Internal(err)
	}

	if sameDate(prevDate, today) {
		status.SignedToday = true
		status.StreakDays = prevStreak
		status.RewardPoints = prevPoints
		status.NextRewardPoints = pointsForStreak(ladder, prevStreak+1)
		return status, nil
	}

	if daysBetween(prevDate, today) == 1 {
		status.StreakDays = prevStreak
		status.RewardPoints = pointsForStreak(ladder, prevStreak+1)
		status.NextRewardPoints = pointsForStreak(ladder, prevStreak+2)
		return status, nil
	}

	status.NextRewardPoints = pointsForStreak(ladder, 2)
	return status, nil
}

// RecordSignIn records the member's sign-in for today, computes the streak-based
// reward and credits it to the points wallet in the same transaction. A second
// sign-in on the same calendar day awards nothing and returns the existing
// result (the unique key on (member_id, sign_date) is the final guard).
func (r *sqlPointsRepository) RecordSignIn(ctx context.Context, memberID int64, idemKey string) (SignInResult, error) {
	ladder := r.signInLadder(ctx)
	today := r.clock.Now()
	dateStr := today.Format("2006-01-02")

	var res SignInResult
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		// Most recent prior sign-in for this member.
		var (
			prevDate   time.Time
			prevStreak int
			prevPoints int64
			hasPrev    = true
		)
		const selPrev = `SELECT sign_date, streak_days, points_awarded FROM sign_in_records
			WHERE member_id = ? ORDER BY sign_date DESC LIMIT 1`
		switch err := tx.QueryRowContext(ctx, selPrev, memberID).Scan(&prevDate, &prevStreak, &prevPoints); {
		case errors.Is(err, sql.ErrNoRows):
			hasPrev = false
		case err != nil:
			return apperr.Internal(err)
		}

		// Already signed in today: idempotent no-op, return the existing award.
		if hasPrev && sameDate(prevDate, today) {
			res = SignInResult{Date: dateStr, PointsEarned: prevPoints, StreakDays: prevStreak, AlreadySigned: true}
			return nil
		}

		streak := nextStreak(hasPrev, prevDate, today, prevStreak)
		points := pointsForStreak(ladder, streak)
		now := time.Now().UTC()

		const insRecord = `INSERT INTO sign_in_records
			(member_id, sign_date, streak_days, points_awarded, idem_key, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`
		insRes, err := tx.ExecContext(ctx, insRecord, memberID, dateStr, streak, points, idemKey, now)
		if err != nil {
			if platdb.IsDuplicate(err) {
				// Concurrent same-day sign-in won the race; treat as no-op.
				return apperr.Conflict("already signed in today")
			}
			return apperr.Internal(err)
		}
		recordID, err := insRes.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}

		if err := creditPoints(ctx, tx, memberID, points, "sign_in", "sign_in", recordID, idemKey, now); err != nil {
			return err
		}

		res = SignInResult{Date: dateStr, PointsEarned: points, StreakDays: streak}
		return nil
	})
	if err != nil {
		return SignInResult{}, err
	}
	return res, nil
}

// signInLadder returns the sign-in points ladder. It prefers an enabled,
// currently-effective rule_definitions row (rule_key=sign_in); when none is
// found or the config is malformed it falls back to signInDailyDefault (the same
// 100..700 numbers, resolved on the server).
func (r *sqlPointsRepository) signInLadder(ctx context.Context) []int64 {
	now := time.Now().UTC()
	const q = `SELECT config_json FROM rule_definitions
		WHERE rule_key = 'sign_in' AND enabled = 1 AND status = 'published'
		  AND (effective_from IS NULL OR effective_from <= ?)
		  AND (effective_to IS NULL OR effective_to > ?)
		ORDER BY version DESC LIMIT 1`
	var raw []byte
	if err := r.db.QueryRowContext(ctx, q, now, now).Scan(&raw); err != nil {
		return signInDailyDefault
	}
	if ladder, ok := parseSignInLadder(raw); ok {
		return ladder
	}
	return signInDailyDefault
}

// creditPoints credits amount to the member's points account and appends the
// matching ledger row, inside the caller's transaction. The account is created
// on first credit. The unique wallet_ledger_entries.idem_key is the final guard
// against a double credit for the same logical operation.
func creditPoints(ctx context.Context, tx *sql.Tx, memberID, amount int64, reason, sourceType string, sourceID int64, idemKey string, now time.Time) error {
	var accountID, available int64
	const selAcct = `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = ? FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, selAcct, memberID, AssetPoints).Scan(&accountID, &available); {
	case errors.Is(err, sql.ErrNoRows):
		const insAcct = `INSERT INTO wallet_accounts
			(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
			VALUES (?, ?, 0, 0, 0, ?, ?)`
		res, err := tx.ExecContext(ctx, insAcct, memberID, AssetPoints, now, now)
		if err != nil {
			return apperr.Internal(err)
		}
		if accountID, err = res.LastInsertId(); err != nil {
			return apperr.Internal(err)
		}
		available = 0
	case err != nil:
		return apperr.Internal(err)
	}

	newBalance := available + amount
	const credit = `UPDATE wallet_accounts SET available_amount = ?, version = version + 1, updated_at = ?
		WHERE id = ?`
	if _, err := tx.ExecContext(ctx, credit, newBalance, now, accountID); err != nil {
		return apperr.Internal(err)
	}

	const insLedger = `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after, reason, source_type, source_id, idem_key, created_at)
		VALUES (?, ?, ?, 'credit', ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insLedger, accountID, memberID, AssetPoints, amount, newBalance, reason, sourceType, sourceID, idemKey, now); err != nil {
		if platdb.IsDuplicate(err) {
			return apperr.Conflict("sign-in already recorded")
		}
		return apperr.Internal(err)
	}
	return nil
}

func (r *sqlPointsRepository) SavePoints(ctx context.Context, memberID, storeID, amount int64, idemKey string) (PointsTxnResult, error) {
	return r.createRequest(ctx, "point_savings", memberID, storeID, amount, "pending", idemKey)
}

func (r *sqlPointsRepository) WithdrawPoints(ctx context.Context, memberID, storeID, amount int64, idemKey string) (PointsTxnResult, error) {
	now := time.Now().UTC()
	var out PointsTxnResult
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var storeName string
		if err := tx.QueryRowContext(ctx, `SELECT name FROM stores WHERE id = ? AND status = 'active'`, storeID).
			Scan(&storeName); errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("store not found")
		} else if err != nil {
			return apperr.Internal(err)
		}

		const insertWithdrawal = `INSERT INTO point_withdrawals
			(store_id, member_id, points, status, remark, idem_key, reviewed_at, created_at, updated_at)
			VALUES (?, ?, ?, 'approved', '用户提取积分', ?, ?, ?, ?)`
		res, err := tx.ExecContext(ctx, insertWithdrawal, storeID, memberID, amount, idemKey, now, now, now)
		if err != nil {
			if platdb.IsDuplicate(err) {
				existing, existingErr := existingPointWithdrawal(ctx, tx, idemKey)
				if existingErr != nil {
					return existingErr
				}
				out = existing
				return nil
			}
			return apperr.Internal(err)
		}
		withdrawalID, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}

		var phone, nickname string
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(phone, ''), COALESCE(nickname, '')
			FROM members WHERE id = ?`, memberID).Scan(&phone, &nickname); errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("member not found")
		} else if err != nil {
			return apperr.Internal(err)
		}

		var accountID, available int64
		const lockAccount = `SELECT id, available_amount FROM wallet_accounts
			WHERE member_id = ? AND asset_type = ? FOR UPDATE`
		if err := tx.QueryRowContext(ctx, lockAccount, memberID, AssetPoints).
			Scan(&accountID, &available); errors.Is(err, sql.ErrNoRows) {
			return apperr.New(apperr.CodeInsufficientBalance, "积分余额不足")
		} else if err != nil {
			return apperr.Internal(err)
		}
		if available < amount {
			return apperr.New(apperr.CodeInsufficientBalance, "积分余额不足")
		}
		balanceAfter := available - amount
		if _, err := tx.ExecContext(ctx, `UPDATE wallet_accounts
			SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`,
			balanceAfter, now, accountID,
		); err != nil {
			return apperr.Internal(err)
		}

		const insertLedger = `INSERT INTO wallet_ledger_entries
			(account_id, member_id, asset_type, direction, amount, balance_after,
			 reason, source_type, source_id, idem_key, created_at)
			VALUES (?, ?, ?, 'debit', ?, ?, 'point_withdrawal', 'point_withdrawal', ?, ?, ?)`
		if _, err := tx.ExecContext(
			ctx, insertLedger, accountID, memberID, AssetPoints, amount, balanceAfter,
			withdrawalID, idemKey, now,
		); err != nil {
			if platdb.IsDuplicate(err) {
				return apperr.Conflict("points withdrawal already recorded")
			}
			return apperr.Internal(err)
		}

		if err := printer.WritePointWithdrawalReceipt(ctx, tx, printer.PointWithdrawalReceipt{
			StoreID: storeID, WithdrawalID: withdrawalID, StoreName: storeName,
			Member: printer.MaskedMember(phone, nickname), Points: amount,
			BalanceAfter: balanceAfter, WithdrawnAt: now,
		}); err != nil {
			return err
		}
		out = PointsTxnResult{
			AssetType: AssetPoints, Amount: amount, BalanceAfter: balanceAfter,
			RequestID: withdrawalID, Status: "approved",
		}
		return nil
	})
	if err != nil {
		return PointsTxnResult{}, err
	}
	return out, nil
}

func existingPointWithdrawal(ctx context.Context, tx *sql.Tx, idemKey string) (PointsTxnResult, error) {
	var out PointsTxnResult
	const q = `SELECT pw.id, pw.points, pw.status, COALESCE(wle.balance_after, 0)
		FROM point_withdrawals pw
		LEFT JOIN wallet_ledger_entries wle
			ON wle.source_type = 'point_withdrawal' AND wle.source_id = pw.id
		WHERE pw.idem_key = ? FOR UPDATE`
	if err := tx.QueryRowContext(ctx, q, idemKey).Scan(
		&out.RequestID, &out.Amount, &out.Status, &out.BalanceAfter,
	); err != nil {
		return PointsTxnResult{}, apperr.Internal(err)
	}
	out.AssetType = AssetPoints
	return out, nil
}

// createRequest inserts a points request. table and status are trusted internal
// literal ("point_savings" / "point_withdrawals"), never client input. On a
// duplicate idempotency key it returns the already-created request.
func (r *sqlPointsRepository) createRequest(ctx context.Context, table string, memberID, storeID, amount int64, status, idemKey string) (PointsTxnResult, error) {
	now := time.Now().UTC()
	var storeExists int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stores WHERE id = ? AND status = 'active'`, storeID).Scan(&storeExists); err != nil {
		return PointsTxnResult{}, apperr.Internal(err)
	}
	if storeExists == 0 {
		return PointsTxnResult{}, apperr.NotFound("store not found")
	}
	q := `INSERT INTO ` + table + ` (store_id, member_id, points, status, idem_key, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q, storeID, memberID, amount, status, idemKey, now, now)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return r.existingRequest(ctx, table, idemKey)
		}
		return PointsTxnResult{}, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return PointsTxnResult{}, apperr.Internal(err)
	}
	return PointsTxnResult{AssetType: AssetPoints, Amount: amount, RequestID: id, Status: status}, nil
}

// existingRequest returns the request previously created under idemKey, so a
// retried create is idempotent.
func (r *sqlPointsRepository) existingRequest(ctx context.Context, table, idemKey string) (PointsTxnResult, error) {
	var id, points int64
	var status string
	q := `SELECT id, points, status FROM ` + table + ` WHERE idem_key = ?`
	if err := r.db.QueryRowContext(ctx, q, idemKey).Scan(&id, &points, &status); err != nil {
		return PointsTxnResult{}, apperr.Internal(err)
	}
	return PointsTxnResult{AssetType: AssetPoints, Amount: points, RequestID: id, Status: status}, nil
}
