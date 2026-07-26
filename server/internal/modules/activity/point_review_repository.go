package activity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func (r *storeSQLRepository) ReviewPointSaving(
	ctx context.Context,
	storeID, requestID int64,
	decision, remark string,
	byID int64,
	now time.Time,
) (PointSaving, error) {
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		var saving PointSaving
		const lockSaving = `SELECT id, store_id, member_id, points, status
			FROM point_savings WHERE id = ? AND store_id = ? FOR UPDATE`
		err := tx.QueryRowContext(ctx, lockSaving, requestID, storeID).Scan(
			&saving.ID, &saving.StoreID, &saving.MemberID, &saving.Points, &saving.Status,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("point-saving request not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if saving.Status != PointSavingPending {
			return apperr.Conflict("point-saving request is not pending review")
		}

		if decision == ReviewReject {
			const reject = `UPDATE point_savings
				SET status = ?, remark = ?, reviewed_by = ?, reviewed_at = ?, updated_at = ?
				WHERE id = ? AND store_id = ? AND status = ?`
			res, err := tx.ExecContext(
				ctx, reject, PointSavingRejected, remark, byID, now, now,
				requestID, storeID, PointSavingPending,
			)
			if err != nil {
				return apperr.Internal(err)
			}
			return requireSingleReview(res)
		}

		rule, err := pointReviewRule(ctx, tx)
		if err != nil {
			return err
		}
		window := businessWindow(now)
		var (
			basePoints   int64
			calcStart    *time.Time
			lastSavingID *int64
		)
		if window.InBusiness {
			start := window.Start.UTC()
			calcStart = &start
			var lastAt time.Time
			const lastApproved = `SELECT id, reviewed_at FROM point_savings
				WHERE member_id = ? AND id <> ? AND status = ?
				  AND reviewed_at >= ? AND reviewed_at < ?
				ORDER BY reviewed_at DESC, id DESC LIMIT 1`
			var lastID int64
			err := tx.QueryRowContext(
				ctx, lastApproved, saving.MemberID, requestID, PointSavingApproved,
				window.Start.UTC(), now,
			).Scan(&lastID, &lastAt)
			switch {
			case errors.Is(err, sql.ErrNoRows):
			case err != nil:
				return apperr.Internal(err)
			default:
				lastSavingID = &lastID
				lastAt = lastAt.UTC()
				calcStart = &lastAt
			}

			const base = `SELECT COALESCE(SUM(points), 0) FROM point_withdrawals
				WHERE member_id = ? AND status = 'approved'
				  AND created_at >= ? AND created_at < ?`
			if err := tx.QueryRowContext(ctx, base, saving.MemberID, *calcStart, now).Scan(&basePoints); err != nil {
				return apperr.Internal(err)
			}
		}

		calc := calculatePointReview(now, saving.Points, basePoints, rule)
		if err := creditPointReviewAsset(
			ctx, tx, saving.MemberID, "points", calc.AwardedPoints,
			"point_saving_reward", requestID,
			fmt.Sprintf("point-saving:%d:points", requestID), now,
		); err != nil {
			return err
		}
		if calc.AwardedCoins > 0 {
			if err := creditPointReviewAsset(
				ctx, tx, saving.MemberID, "coins", calc.AwardedCoins,
				"point_saving_coin_reward", requestID,
				fmt.Sprintf("point-saving:%d:coins", requestID), now,
			); err != nil {
				return err
			}
		}

		var businessDate any
		var businessStart, businessEnd any
		if window.InBusiness {
			businessDate = window.Date
			businessStart = window.Start.UTC()
			businessEnd = window.End.UTC()
		}
		const approve = `UPDATE point_savings SET
			status = ?, remark = ?, reviewed_by = ?, reviewed_at = ?, updated_at = ?,
			base_points = ?, excess_points = ?, awarded_points = ?, coin_base_points = ?,
			awarded_coins = ?, rule_version = ?, points_divisor = ?, coin_points_divisor = ?,
			business_date = ?, business_start_at = ?, business_end_at = ?,
			calculation_start_at = ?, calculation_end_at = ?, last_approved_saving_id = ?,
			calculation_description = ?
			WHERE id = ? AND store_id = ? AND status = ?`
		res, err := tx.ExecContext(
			ctx, approve,
			PointSavingApproved, remark, byID, now, now,
			calc.BasePoints, calc.ExcessPoints, calc.AwardedPoints, calc.CoinBasePoints,
			calc.AwardedCoins, rule.Version, rule.PointsDivisor, rule.CoinPointsDivisor,
			businessDate, businessStart, businessEnd, calcStart, now, lastSavingID,
			calc.Description, requestID, storeID, PointSavingPending,
		)
		if err != nil {
			return apperr.Internal(err)
		}
		return requireSingleReview(res)
	})
	if err != nil {
		return PointSaving{}, err
	}
	return r.GetPointSaving(ctx, storeID, requestID)
}

func requireSingleReview(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return apperr.Internal(err)
	}
	if affected != 1 {
		return apperr.Conflict("point-saving request is not pending review")
	}
	return nil
}

func pointReviewRule(ctx context.Context, tx *sql.Tx) (PointReviewRule, error) {
	var rule PointReviewRule
	const q = `SELECT points_divisor, coin_points_divisor, version
		FROM point_review_settings WHERE id = 1`
	err := tx.QueryRowContext(ctx, q).Scan(
		&rule.PointsDivisor, &rule.CoinPointsDivisor, &rule.Version,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return PointReviewRule{
			PointsDivisor: defaultPointsDivisor, CoinPointsDivisor: defaultCoinPointsDivisor, Version: 1,
		}, nil
	}
	if err != nil {
		return PointReviewRule{}, apperr.Internal(err)
	}
	if rule.PointsDivisor <= 0 || rule.CoinPointsDivisor <= 0 {
		return PointReviewRule{}, apperr.Internal(errors.New("invalid point review settings"))
	}
	return rule, nil
}

func creditPointReviewAsset(
	ctx context.Context,
	tx *sql.Tx,
	memberID int64,
	assetType string,
	amount int64,
	reason string,
	requestID int64,
	idemKey string,
	now time.Time,
) error {
	var accountID, available int64
	const lock = `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = ? FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, lock, memberID, assetType).Scan(&accountID, &available); {
	case errors.Is(err, sql.ErrNoRows):
		const create = `INSERT INTO wallet_accounts
			(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
			VALUES (?, ?, 0, 0, 0, ?, ?)`
		result, err := tx.ExecContext(ctx, create, memberID, assetType, now, now)
		if err != nil {
			return apperr.Internal(err)
		}
		accountID, err = result.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
	case err != nil:
		return apperr.Internal(err)
	}

	newBalance := available + amount
	const update = `UPDATE wallet_accounts
		SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`
	if _, err := tx.ExecContext(ctx, update, newBalance, now, accountID); err != nil {
		return apperr.Internal(err)
	}
	const ledger = `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after,
		 reason, source_type, source_id, idem_key, created_at)
		VALUES (?, ?, ?, 'credit', ?, ?, ?, 'point_saving', ?, ?, ?)`
	if _, err := tx.ExecContext(
		ctx, ledger, accountID, memberID, assetType, amount, newBalance,
		reason, requestID, idemKey, now,
	); err != nil {
		if platdb.IsDuplicate(err) {
			return apperr.Conflict("point-saving request is already credited")
		}
		return apperr.Internal(err)
	}
	return nil
}
