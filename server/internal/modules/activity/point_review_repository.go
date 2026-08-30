package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func (r *storeSQLRepository) ReviewPointSaving(
	ctx context.Context,
	storeID, requestID int64,
	decision, remark, reviewerType string,
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
		reviewerSnapshot, err := loadPointSavingReviewerSnapshot(ctx, tx, storeID, reviewerType, byID)
		if err != nil {
			return err
		}

		if decision == ReviewReject {
			const reject = `UPDATE point_savings
				SET status = ?, remark = ?, reviewed_by = ?, reviewed_by_type = ?,
					reviewer_snapshot_json = ?, reviewed_at = ?, updated_at = ?
				WHERE id = ? AND store_id = ? AND status = ?`
			res, err := tx.ExecContext(
				ctx, reject, PointSavingRejected, remark, byID, reviewerType,
				reviewerSnapshot, now, now,
				requestID, storeID, PointSavingPending,
			)
			if err != nil {
				return apperr.Internal(err)
			}
			return requireSingleReview(res)
		}

		evaluation, err := evaluatePointReview(ctx, tx, saving, now)
		if err != nil {
			return err
		}
		calc := evaluation.Calculation
		rule := evaluation.Rule
		window := calc.Window
		calcStart := evaluation.CalculationStartAt
		lastSavingID := evaluation.LastApprovedSavingID
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
			status = ?, remark = ?, reviewed_by = ?, reviewed_by_type = ?,
			reviewer_snapshot_json = ?, reviewed_at = ?, updated_at = ?,
			base_points = ?, excess_points = ?, awarded_points = ?, coin_base_points = ?,
			awarded_coins = ?, rule_version = ?, points_divisor = ?, below_base_points_divisor = ?, coin_points_divisor = ?,
			business_date = ?, business_start_at = ?, business_end_at = ?,
			calculation_start_at = ?, calculation_end_at = ?, last_approved_saving_id = ?,
			calculation_description = ?
			WHERE id = ? AND store_id = ? AND status = ?`
		res, err := tx.ExecContext(
			ctx, approve,
			PointSavingApproved, remark, byID, reviewerType, reviewerSnapshot, now, now,
			calc.BasePoints, calc.ExcessPoints, calc.AwardedPoints, calc.CoinBasePoints,
			calc.AwardedCoins, rule.Version, rule.PointsDivisor, rule.BelowBasePointsDivisor, rule.CoinPointsDivisor,
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

type pointReviewQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type pointReviewEvaluation struct {
	Calculation          PointReviewCalculation
	Rule                 PointReviewRule
	CalculationStartAt   *time.Time
	LastApprovedSavingID *int64
}

func evaluatePointReview(
	ctx context.Context,
	queryer pointReviewQueryer,
	saving PointSaving,
	now time.Time,
) (pointReviewEvaluation, error) {
	rule, err := pointReviewRule(ctx, queryer)
	if err != nil {
		return pointReviewEvaluation{}, err
	}
	var (
		basePoints   int64
		calcStart    *time.Time
		lastSavingID *int64
	)
	start := pointReviewBaseWindowStart(now).UTC()
	calcStart = &start
	var lastAt time.Time
	const lastApproved = `SELECT id, reviewed_at FROM point_savings
		WHERE member_id = ? AND id <> ? AND status = ?
		  AND reviewed_at >= ? AND reviewed_at < ?
		ORDER BY reviewed_at DESC, id DESC LIMIT 1`
	var lastID int64
	err = queryer.QueryRowContext(
		ctx, lastApproved, saving.MemberID, saving.ID, PointSavingApproved,
		start, now,
	).Scan(&lastID, &lastAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return pointReviewEvaluation{}, apperr.Internal(err)
	default:
		lastSavingID = &lastID
		lastAt = lastAt.UTC()
		calcStart = &lastAt
	}

	const base = `SELECT COALESCE(SUM(points), 0) FROM point_withdrawals
		WHERE member_id = ? AND status = 'approved'
		  AND created_at >= ? AND created_at < ?`
	if err := queryer.QueryRowContext(ctx, base, saving.MemberID, *calcStart, now).Scan(&basePoints); err != nil {
		return pointReviewEvaluation{}, apperr.Internal(err)
	}

	return pointReviewEvaluation{
		Calculation:          calculatePointReview(now, saving.Points, basePoints, rule),
		Rule:                 rule,
		CalculationStartAt:   calcStart,
		LastApprovedSavingID: lastSavingID,
	}, nil
}

func (r *storeSQLRepository) PreviewPointSaving(
	ctx context.Context,
	saving PointSaving,
	now time.Time,
) (PointSaving, error) {
	if saving.Status != PointSavingPending {
		return saving, nil
	}
	evaluation, err := evaluatePointReview(ctx, r.db, saving, now)
	if err != nil {
		return PointSaving{}, err
	}
	calc := evaluation.Calculation
	saving.BasePoints = calc.BasePoints
	saving.ExcessPoints = calc.ExcessPoints
	saving.AwardedPoints = calc.AwardedPoints
	saving.CoinBasePoints = calc.CoinBasePoints
	saving.AwardedCoins = calc.AwardedCoins
	saving.RuleVersion = evaluation.Rule.Version
	saving.PointsDivisor = evaluation.Rule.PointsDivisor
	saving.BelowBasePointsDivisor = evaluation.Rule.BelowBasePointsDivisor
	saving.CoinPointsDivisor = evaluation.Rule.CoinPointsDivisor
	saving.CalculationStartAt = evaluation.CalculationStartAt
	saving.LastApprovedSavingID = evaluation.LastApprovedSavingID
	saving.CalculationDescription = calc.Description
	return saving, nil
}

type pointSavingReviewerSnapshot struct {
	Type        string `json:"type"`
	ID          int64  `json:"id"`
	Role        string `json:"role,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	StaffName   string `json:"staffName,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	Phone       string `json:"phone,omitempty"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

func loadPointSavingReviewerSnapshot(
	ctx context.Context,
	tx *sql.Tx,
	storeID int64,
	reviewerType string,
	byID int64,
) (string, error) {
	snapshot := pointSavingReviewerSnapshot{Type: reviewerType, ID: byID}
	switch reviewerType {
	case "staff":
		err := tx.QueryRowContext(ctx, `SELECT sa.name, m.nickname, COALESCE(m.phone,''), COALESCE(m.avatar_url,'')
			FROM staff_accounts sa JOIN members m ON m.id = sa.member_id
			WHERE sa.member_id = ? AND sa.store_id = ?`, byID, storeID,
		).Scan(&snapshot.StaffName, &snapshot.Nickname, &snapshot.Phone, &snapshot.AvatarURL)
		if errors.Is(err, sql.ErrNoRows) {
			return "", apperr.Forbidden("reviewing staff account not found")
		}
		if err != nil {
			return "", apperr.Internal(err)
		}
	case "store_admin", "cashier":
		err := tx.QueryRowContext(ctx, `SELECT role, username, display_name
			FROM admin_accounts WHERE id = ? AND store_id = ?`, byID, storeID,
		).Scan(&snapshot.Role, &snapshot.Username, &snapshot.DisplayName)
		if errors.Is(err, sql.ErrNoRows) {
			return "", apperr.Forbidden("reviewing admin account not found")
		}
		if err != nil {
			return "", apperr.Internal(err)
		}
	default:
		return "", apperr.Forbidden("unsupported point-saving reviewer")
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return "", apperr.Internal(err)
	}
	return string(raw), nil
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

func pointReviewRule(ctx context.Context, queryer pointReviewQueryer) (PointReviewRule, error) {
	var rule PointReviewRule
	const q = `SELECT points_divisor, below_base_points_divisor, coin_points_divisor, version
		FROM point_review_settings WHERE id = 1`
	err := queryer.QueryRowContext(ctx, q).Scan(
		&rule.PointsDivisor, &rule.BelowBasePointsDivisor, &rule.CoinPointsDivisor, &rule.Version,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return PointReviewRule{
			PointsDivisor: defaultPointsDivisor, BelowBasePointsDivisor: defaultBelowBasePointsDivisor,
			CoinPointsDivisor: defaultCoinPointsDivisor, Version: 1,
		}, nil
	}
	if err != nil {
		return PointReviewRule{}, apperr.Internal(err)
	}
	if rule.PointsDivisor <= 0 || rule.BelowBasePointsDivisor <= 0 || rule.CoinPointsDivisor <= 0 {
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
