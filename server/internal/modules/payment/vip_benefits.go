package payment

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

const (
	vipBenefitReason     = "VIP等级福利"
	vipBenefitSourceType = "vip_tier_benefit"
)

type vipPointBenefit struct {
	Amount int64 `json:"amount"`
}

type vipCouponBenefit struct {
	CouponType string `json:"couponType"`
	Quantity   int    `json:"quantity"`
}

type vipBenefitConfig struct {
	Points  []vipPointBenefit  `json:"points"`
	Coupons []vipCouponBenefit `json:"coupons"`
}

// grantVIPTierBenefits grants the newly reached tier's current benefit batch.
// Ledger and entitlement idempotency keys make replays harmless. Coupon grants
// always expire after 30 days and never depend on a face value.
func grantVIPTierBenefits(ctx context.Context, tx *sql.Tx, memberID, tierID int64, now time.Time) error {
	var raw []byte
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(benefit_config, JSON_OBJECT()) FROM membership_tiers WHERE id = ?`, tierID,
	).Scan(&raw); err != nil {
		return apperr.Internal(err)
	}
	var config vipBenefitConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return apperr.Internal(err)
	}
	var totalPoints int64
	for _, benefit := range config.Points {
		if benefit.Amount > 0 {
			totalPoints += benefit.Amount
		}
	}
	if totalPoints > 0 {
		if err := grantVIPPoints(ctx, tx, memberID, tierID, totalPoints, now); err != nil {
			return err
		}
	}
	for benefitIndex, benefit := range config.Coupons {
		if benefit.Quantity <= 0 || benefit.Quantity > 99 {
			continue
		}
		var templateID int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM coupon_templates
			WHERE scope_type = 'global' AND coupon_type = ? AND status = 'published'
			ORDER BY id ASC LIMIT 1`, benefit.CouponType).Scan(&templateID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return apperr.Internal(err)
		}
		for sequence := 0; sequence < benefit.Quantity; sequence++ {
			if err := grantVIPCoupon(ctx, tx, memberID, tierID, templateID, benefitIndex, sequence, now); err != nil {
				return err
			}
		}
	}
	return nil
}

func grantVIPPoints(ctx context.Context, tx *sql.Tx, memberID, tierID, amount int64, now time.Time) error {
	var accountID, available int64
	const selectAccount = `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = 'points' FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, selectAccount, memberID).Scan(&accountID, &available); {
	case errors.Is(err, sql.ErrNoRows):
		res, err := tx.ExecContext(ctx, `INSERT INTO wallet_accounts
			(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
			VALUES (?, 'points', 0, 0, 0, ?, ?)`, memberID, now, now)
		if err != nil {
			return apperr.Internal(err)
		}
		accountID, err = res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
	case err != nil:
		return apperr.Internal(err)
	}
	idemKey := fmt.Sprintf("vip-tier:%d:%d:points", memberID, tierID)
	newBalance := available + amount
	_, err := tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after, reason,
		 source_type, source_id, idem_key, created_at)
		VALUES (?, ?, 'points', 'credit', ?, ?, ?, ?, ?, ?, ?)`,
		accountID, memberID, amount, newBalance, vipBenefitReason, vipBenefitSourceType, tierID, idemKey, now)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return nil
		}
		return apperr.Internal(err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE wallet_accounts
		SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`,
		newBalance, now, accountID)
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func grantVIPCoupon(
	ctx context.Context,
	tx *sql.Tx,
	memberID, tierID, templateID int64,
	benefitIndex, sequence int,
	now time.Time,
) error {
	idemKey := fmt.Sprintf("vip-tier:%d:%d:coupon:%d:%d", memberID, tierID, benefitIndex, sequence)
	entitlementNo := fmt.Sprintf("VIP%d-%d-%d-%d", memberID, tierID, benefitIndex, sequence)
	expiresAt := now.AddDate(0, 0, 30)
	_, err := tx.ExecContext(ctx, `INSERT INTO coupon_entitlements
		(entitlement_no, coupon_template_id, member_id, store_id, status, rule_version,
		 granted_reason, granted_by_type, expires_at, idem_key, created_at, updated_at)
		VALUES (?, ?, ?, NULL, 'active', 1, ?, 'system', ?, ?, ?, ?)`,
		entitlementNo, templateID, memberID, vipBenefitReason, expiresAt, idemKey, now, now)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return nil
		}
		return apperr.Internal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE coupon_templates
		SET issued_quantity = issued_quantity + 1, updated_at = ? WHERE id = ?`, now, templateID); err != nil {
		return apperr.Internal(err)
	}
	return nil
}
