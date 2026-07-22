package wallet

import (
	"context"
	"database/sql"
	"errors"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// sqlPointsRepository is the MySQL points write repository. Point savings and
// withdrawals are member-initiated requests reviewed by the store console: each
// create inserts a 'pending' row into point_savings / point_withdrawals and
// claims the idempotency key via the row's unique key, so a retried create
// returns the existing request instead of duplicating it.
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
			res = SignInResult{Date: dateStr, PointsEarned: prevPoints, StreakDays: prevStreak}
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
	return r.createRequest(ctx, "point_savings", memberID, storeID, amount, idemKey)
}

func (r *sqlPointsRepository) WithdrawPoints(ctx context.Context, memberID, storeID, amount int64, idemKey string) (PointsTxnResult, error) {
	return r.createRequest(ctx, "point_withdrawals", memberID, storeID, amount, idemKey)
}

// createRequest inserts a pending points request. table is a trusted internal
// literal ("point_savings" / "point_withdrawals"), never client input. On a
// duplicate idempotency key it returns the already-created request.
func (r *sqlPointsRepository) createRequest(ctx context.Context, table string, memberID, storeID, amount int64, idemKey string) (PointsTxnResult, error) {
	now := time.Now().UTC()
	var storeArg any
	if storeID > 0 {
		storeArg = storeID
	}
	q := `INSERT INTO ` + table + ` (store_id, member_id, points, status, idem_key, created_at, updated_at)
		VALUES (?, ?, ?, 'pending', ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q, storeArg, memberID, amount, idemKey, now, now)
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
	return PointsTxnResult{AssetType: AssetPoints, Amount: amount, RequestID: id, Status: "pending"}, nil
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
