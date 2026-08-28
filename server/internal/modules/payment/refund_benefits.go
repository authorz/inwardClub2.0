package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

const (
	refundBenefitReason         = "refund_benefit_clawback"
	refundBenefitRollbackReason = "refund_benefit_rollback"
	refundBenefitSource         = "refund_order"
	refundPendingCouponStatus   = "refund_pending"
)

type refundWalletGrant struct {
	accountID int64
	memberID  int64
	assetType string
	amount    int64
}

type refundAccountClawback struct {
	accountID int64
	memberID  int64
	assetType string
	amount    int64
}

func prepareRefundBenefits(
	ctx context.Context,
	tx *sql.Tx,
	refundID, paymentOrderID, businessOrderID, refundCent, paidCent int64,
	orderType string,
	now time.Time,
) error {
	grants, err := refundWalletGrants(ctx, tx, paymentOrderID, businessOrderID)
	if err != nil {
		return err
	}
	couponIDs, hasCouponBenefit, err := refundableCouponIDs(ctx, tx, paymentOrderID, businessOrderID)
	if err != nil {
		return err
	}
	hasBenefit := len(grants) > 0 || hasCouponBenefit
	if refundCent < paidCent && (orderType == orderTypeRecharge || hasBenefit) {
		return apperr.Conflict("该订单已发放权益，仅支持全额退款")
	}
	clawbacks := aggregateRefundGrants(grants)
	if err := subtractPreparedFoodCancellation(ctx, tx, paymentOrderID, clawbacks); err != nil {
		return err
	}
	for _, clawback := range orderedClawbacks(clawbacks) {
		if clawback.amount <= 0 {
			continue
		}
		var available int64
		if err := tx.QueryRowContext(ctx, `SELECT available_amount FROM wallet_accounts
			WHERE id = ? AND member_id = ? AND asset_type = ? FOR UPDATE`,
			clawback.accountID, clawback.memberID, clawback.assetType,
		).Scan(&available); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return apperr.Conflict("会员权益账户不存在，无法完成退款")
			}
			return apperr.Internal(err)
		}
		newBalance := available - clawback.amount
		if _, err := tx.ExecContext(ctx, `UPDATE wallet_accounts
			SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`,
			newBalance, now, clawback.accountID,
		); err != nil {
			return apperr.Internal(err)
		}
		idemKey := fmt.Sprintf("refund_benefit_clawback:%d:%d", refundID, clawback.accountID)
		if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries
			(account_id, member_id, asset_type, direction, amount, balance_after, reason,
			 source_type, source_id, idem_key, created_at)
			VALUES (?, ?, ?, 'debit', ?, ?, ?, ?, ?, ?, ?)`,
			clawback.accountID, clawback.memberID, clawback.assetType, clawback.amount,
			newBalance, refundBenefitReason, refundBenefitSource, refundID, idemKey, now,
		); err != nil {
			return apperr.Internal(err)
		}
	}
	if len(couponIDs) > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE coupon_entitlements
			SET status = ?, updated_at = ? WHERE id IN (`+placeholders(len(couponIDs))+`)`,
			append([]any{refundPendingCouponStatus, now}, int64Args(couponIDs)...)...,
		); err != nil {
			return apperr.Internal(err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE benefit_grants
		SET status = 'refund_pending'
		WHERE source_type = 'offline_collection' AND source_id = ?
			AND rule_key = 'low_spend_reward' AND status = 'granted'`, paymentOrderID); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func refundWalletGrants(ctx context.Context, tx *sql.Tx, paymentOrderID, businessOrderID int64) ([]refundWalletGrant, error) {
	const q = `SELECT account_id, member_id, asset_type, amount
		FROM wallet_ledger_entries
		WHERE direction = 'credit' AND amount > 0 AND (
			(source_id = ? AND source_type IN
			 ('recharge_order', 'first_recharge_reward', 'food_order', 'low_spend_reward',
			  'wechat_payment_growth', 'vip_benefit_order'))
			OR (source_id = ? AND source_type = 'offline_collection' AND reason = 'low_spend_reward')
		)
		ORDER BY account_id, id FOR UPDATE`
	rows, err := tx.QueryContext(ctx, q, businessOrderID, paymentOrderID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	var grants []refundWalletGrant
	for rows.Next() {
		var grant refundWalletGrant
		if err := rows.Scan(&grant.accountID, &grant.memberID, &grant.assetType, &grant.amount); err != nil {
			return nil, apperr.Internal(err)
		}
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return grants, nil
}

func aggregateRefundGrants(grants []refundWalletGrant) map[int64]*refundAccountClawback {
	out := make(map[int64]*refundAccountClawback)
	for _, grant := range grants {
		entry := out[grant.accountID]
		if entry == nil {
			entry = &refundAccountClawback{
				accountID: grant.accountID,
				memberID:  grant.memberID,
				assetType: grant.assetType,
			}
			out[grant.accountID] = entry
		}
		entry.amount += grant.amount
	}
	return out
}

func orderedClawbacks(clawbacks map[int64]*refundAccountClawback) []*refundAccountClawback {
	accountIDs := make([]int64, 0, len(clawbacks))
	for accountID := range clawbacks {
		accountIDs = append(accountIDs, accountID)
	}
	sort.Slice(accountIDs, func(i, j int) bool { return accountIDs[i] < accountIDs[j] })
	out := make([]*refundAccountClawback, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		out = append(out, clawbacks[accountID])
	}
	return out
}

func subtractPreparedFoodCancellation(
	ctx context.Context,
	tx *sql.Tx,
	paymentOrderID int64,
	clawbacks map[int64]*refundAccountClawback,
) error {
	var recovered int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(points_recovered), 0)
		FROM food_order_cancellations WHERE payment_order_id = ? AND status = 'processing'`,
		paymentOrderID,
	).Scan(&recovered); err != nil {
		return apperr.Internal(err)
	}
	if recovered <= 0 {
		return nil
	}
	for _, clawback := range clawbacks {
		if clawback.assetType != "points" {
			continue
		}
		if recovered >= clawback.amount {
			recovered -= clawback.amount
			clawback.amount = 0
			continue
		}
		clawback.amount -= recovered
		break
	}
	return nil
}

func refundableCouponIDs(
	ctx context.Context,
	tx *sql.Tx,
	paymentOrderID, businessOrderID int64,
) ([]int64, bool, error) {
	rechargeKey := fmt.Sprintf("recharge_coupon:%d", paymentOrderID)
	foodPrefix := fmt.Sprintf("food_coupon:%d:%%", paymentOrderID)
	const q = `SELECT id, status FROM coupon_entitlements
		WHERE idem_key = ? OR idem_key LIKE ?
			OR (granted_by_id = ? AND granted_by_type = 'system' AND granted_reason = 'VIP等级福利')
		ORDER BY id FOR UPDATE`
	rows, err := tx.QueryContext(ctx, q, rechargeKey, foodPrefix, businessOrderID)
	if err != nil {
		return nil, false, apperr.Internal(err)
	}
	defer rows.Close()
	var ids []int64
	hasBenefit := false
	for rows.Next() {
		var id int64
		var status string
		if err := rows.Scan(&id, &status); err != nil {
			return nil, false, apperr.Internal(err)
		}
		hasBenefit = true
		switch status {
		case "active":
			ids = append(ids, id)
		case "used":
			return nil, false, apperr.Conflict("订单发放的券已使用，无法退款")
		}
	}
	if err := rows.Err(); err != nil {
		return nil, false, apperr.Internal(err)
	}
	return ids, hasBenefit, nil
}

func completeRefundBenefits(
	ctx context.Context,
	tx *sql.Tx,
	refundID, paymentOrderID, businessOrderID int64,
	memberID sql.NullInt64,
	now time.Time,
) error {
	if err := updateRefundCoupons(ctx, tx, paymentOrderID, businessOrderID, refundPendingCouponStatus, "void", now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE benefit_grants SET status = 'revoked'
		WHERE source_type = 'offline_collection' AND source_id = ?
			AND rule_key = 'low_spend_reward' AND status = 'refund_pending'`, paymentOrderID); err != nil {
		return apperr.Internal(err)
	}
	if !memberID.Valid {
		return nil
	}
	var growthClawedBack bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_ledger_entries
		WHERE source_type = ? AND source_id = ? AND reason = ? AND asset_type = 'growth_value')`,
		refundBenefitSource, refundID, refundBenefitReason,
	).Scan(&growthClawedBack); err != nil {
		return apperr.Internal(err)
	}
	if growthClawedBack {
		return reconcileRefundedMemberTier(ctx, tx, memberID.Int64, now)
	}
	return nil
}

func rollbackRefundBenefits(
	ctx context.Context,
	tx *sql.Tx,
	refundID, paymentOrderID, businessOrderID int64,
	now time.Time,
) error {
	rows, err := tx.QueryContext(ctx, `SELECT account_id, member_id, asset_type, amount
		FROM wallet_ledger_entries WHERE source_type = ? AND source_id = ?
		AND direction = 'debit' AND reason = ? ORDER BY account_id FOR UPDATE`,
		refundBenefitSource, refundID, refundBenefitReason,
	)
	if err != nil {
		return apperr.Internal(err)
	}
	var clawbacks []refundAccountClawback
	for rows.Next() {
		var clawback refundAccountClawback
		if err := rows.Scan(&clawback.accountID, &clawback.memberID, &clawback.assetType, &clawback.amount); err != nil {
			rows.Close()
			return apperr.Internal(err)
		}
		clawbacks = append(clawbacks, clawback)
	}
	if err := rows.Close(); err != nil {
		return apperr.Internal(err)
	}
	if err := rows.Err(); err != nil {
		return apperr.Internal(err)
	}
	for _, clawback := range clawbacks {
		var available int64
		if err := tx.QueryRowContext(ctx, `SELECT available_amount FROM wallet_accounts WHERE id = ? FOR UPDATE`,
			clawback.accountID,
		).Scan(&available); err != nil {
			return apperr.Internal(err)
		}
		newBalance := available + clawback.amount
		if _, err := tx.ExecContext(ctx, `UPDATE wallet_accounts
			SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`,
			newBalance, now, clawback.accountID,
		); err != nil {
			return apperr.Internal(err)
		}
		idemKey := fmt.Sprintf("refund_benefit_rollback:%d:%d", refundID, clawback.accountID)
		if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries
			(account_id, member_id, asset_type, direction, amount, balance_after, reason,
			 source_type, source_id, idem_key, created_at)
			VALUES (?, ?, ?, 'credit', ?, ?, ?, ?, ?, ?, ?)`,
			clawback.accountID, clawback.memberID, clawback.assetType, clawback.amount,
			newBalance, refundBenefitRollbackReason, refundBenefitSource, refundID, idemKey, now,
		); err != nil {
			return apperr.Internal(err)
		}
	}
	if err := updateRefundCoupons(ctx, tx, paymentOrderID, businessOrderID, refundPendingCouponStatus, "active", now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE benefit_grants SET status = 'granted'
		WHERE source_type = 'offline_collection' AND source_id = ?
			AND rule_key = 'low_spend_reward' AND status = 'refund_pending'`, paymentOrderID); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func updateRefundCoupons(
	ctx context.Context,
	tx *sql.Tx,
	paymentOrderID, businessOrderID int64,
	fromStatus, toStatus string,
	now time.Time,
) error {
	rechargeKey := fmt.Sprintf("recharge_coupon:%d", paymentOrderID)
	foodPrefix := fmt.Sprintf("food_coupon:%d:%%", paymentOrderID)
	if _, err := tx.ExecContext(ctx, `UPDATE coupon_entitlements SET status = ?, updated_at = ?
		WHERE status = ? AND (idem_key = ? OR idem_key LIKE ?
			OR (granted_by_id = ? AND granted_by_type = 'system' AND granted_reason = 'VIP等级福利'))`,
		toStatus, now, fromStatus, rechargeKey, foodPrefix, businessOrderID,
	); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func reconcileRefundedMemberTier(ctx context.Context, tx *sql.Tx, memberID int64, now time.Time) error {
	var growthBalance int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = 'growth_value'), 0)`, memberID).Scan(&growthBalance); err != nil {
		return apperr.Internal(err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, level, threshold FROM membership_tiers WHERE status = 'active'`)
	if err != nil {
		return apperr.Internal(err)
	}
	var tiers []tierRow
	for rows.Next() {
		var tier tierRow
		if err := rows.Scan(&tier.id, &tier.level, &tier.threshold); err != nil {
			rows.Close()
			return apperr.Internal(err)
		}
		tiers = append(tiers, tier)
	}
	if err := rows.Close(); err != nil {
		return apperr.Internal(err)
	}
	if err := rows.Err(); err != nil {
		return apperr.Internal(err)
	}
	var tierArg any
	if tier, ok := resolveTier(tiers, growthBalance); ok {
		tierArg = tier.id
	}
	if _, err := tx.ExecContext(ctx, `UPDATE members SET current_tier_id = ?, updated_at = ? WHERE id = ?`,
		tierArg, now, memberID,
	); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func placeholders(count int) string {
	out := "?"
	for i := 1; i < count; i++ {
		out += ",?"
	}
	return out
}

func int64Args(values []int64) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}
