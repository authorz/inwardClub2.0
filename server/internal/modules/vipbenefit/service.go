// Package vipbenefit applies configured VIP benefits at their real business
// triggers. It is database-only so both WeChat and coin settlement can share the
// same transaction-safe, idempotent grant implementation.
package vipbenefit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

const (
	grantReason     = "VIP等级福利"
	grantSourceType = "vip_benefit"
)

type pointBenefit struct {
	Amount  int64  `json:"amount"`
	Period  string `json:"period"`
	Trigger string `json:"trigger"`
}

type couponBenefit struct {
	CategoryID   int64  `json:"categoryId"`
	CouponType   string `json:"couponType"`
	Quantity     int    `json:"quantity"`
	Period       string `json:"period"`
	Trigger      string `json:"trigger"`
	ValidityDays int    `json:"validityDays"`
}

type benefitConfig struct {
	Points  []pointBenefit  `json:"points"`
	Coupons []couponBenefit `json:"coupons"`
}

type tierBenefits struct {
	TierID int64
	Config benefitConfig
}

// FoodPayment describes one successfully paid member food order.
type FoodPayment struct {
	PaymentOrderID  int64
	BusinessOrderID int64
	MemberID        int64
	StoreID         int64
	PaidAt          time.Time
	LowSpend        bool
}

type growthTier struct {
	ID    int64
	Level int
}

// UpgradeForGrowthBalance advances a member to the highest active tier covered
// by their current growth balance, then grants that tier's reached and scheduled
// benefits. The caller supplies the surrounding transaction so a growth credit,
// tier change and all rewards either commit together or roll back together.
func UpgradeForGrowthBalance(ctx context.Context, tx *sql.Tx, memberID, growthBalance int64, now time.Time) error {
	var target growthTier
	err := tx.QueryRowContext(ctx, `SELECT id, level FROM membership_tiers
		WHERE status = 'active' AND threshold <= ?
		ORDER BY threshold DESC, level DESC, id ASC LIMIT 1`, growthBalance).
		Scan(&target.ID, &target.Level)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperr.Internal(err)
	}

	var currentTierID sql.NullInt64
	switch err := tx.QueryRowContext(ctx, `SELECT current_tier_id FROM members WHERE id = ? FOR UPDATE`, memberID).Scan(&currentTierID); {
	case errors.Is(err, sql.ErrNoRows):
		return apperr.NotFound("member not found")
	case err != nil:
		return apperr.Internal(err)
	}

	var currentLevel *int
	if currentTierID.Valid {
		var level int
		switch err := tx.QueryRowContext(ctx, `SELECT level FROM membership_tiers WHERE id = ?`, currentTierID.Int64).Scan(&level); {
		case errors.Is(err, sql.ErrNoRows):
			// A dangling tier reference is treated as unranked and repaired below.
		case err != nil:
			return apperr.Internal(err)
		default:
			currentLevel = &level
		}
	}
	if !tierUpgradeNeeded(currentLevel, target.Level) {
		return nil
	}

	if _, err := tx.ExecContext(ctx, `UPDATE members SET current_tier_id = ?, updated_at = ? WHERE id = ?`, target.ID, now, memberID); err != nil {
		return apperr.Internal(err)
	}
	return GrantTierReached(ctx, tx, memberID, target.ID, now)
}

func tierUpgradeNeeded(currentLevel *int, targetLevel int) bool {
	return currentLevel == nil || *currentLevel < targetLevel
}

// GrantTierReached applies one-time reached-tier benefits and the current
// natural period's scheduled benefits. It is safe to call on retries.
func GrantTierReached(ctx context.Context, tx *sql.Tx, memberID, tierID int64, now time.Time) error {
	tier, err := loadTierByID(ctx, tx, tierID)
	if err != nil {
		return err
	}
	if _, err := grantMatching(ctx, tx, memberID, tier, now, func(trigger string) bool {
		return trigger == "tier_achieved"
	}); err != nil {
		return err
	}
	_, err = grantScheduled(ctx, tx, memberID, tier, now)
	return err
}

// GrantFoodPayment applies benefits driven by a paid food order: first paid
// order of the local day, qualified low spend, and the first in-hours visit.
func GrantFoodPayment(ctx context.Context, tx *sql.Tx, in FoodPayment) (int64, error) {
	tier, ok, err := loadCurrentTier(ctx, tx, in.MemberID)
	if err != nil || !ok {
		return 0, err
	}
	firstOrder, err := isFirstPaidFoodOrder(ctx, tx, in.MemberID, in.PaymentOrderID, in.PaidAt)
	if err != nil {
		return 0, err
	}
	visit, err := isInHoursVisit(ctx, tx, in.StoreID, in.PaidAt)
	if err != nil {
		return 0, err
	}
	return grantMatching(ctx, tx, in.MemberID, tier, in.PaidAt, func(trigger string) bool {
		switch trigger {
		case "low_spend":
			return in.LowSpend
		case "first_order":
			return firstOrder
		case "visit":
			return visit
		default:
			return false
		}
	})
}

// Service runs the daily scheduled sweep. Weekly/monthly grants are selected by
// their period key, so a repeated daily sweep grants at most once per period.
type Service struct {
	db  *platdb.DB
	now func() time.Time
}

func NewService(db *platdb.DB) *Service {
	return &Service{db: db, now: time.Now}
}

func (s *Service) SweepScheduled(ctx context.Context) (int64, error) {
	now := s.now().UTC()
	rows, err := s.db.QueryContext(ctx, `SELECT m.id, mt.id, COALESCE(mt.benefit_config, JSON_OBJECT())
		FROM members m JOIN membership_tiers mt ON mt.id = m.current_tier_id
		WHERE m.status = 'active' AND mt.status = 'active' ORDER BY m.id`)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	var entries []struct {
		memberID int64
		tier     tierBenefits
	}
	for rows.Next() {
		var raw []byte
		var entry struct {
			memberID int64
			tier     tierBenefits
		}
		if err := rows.Scan(&entry.memberID, &entry.tier.TierID, &raw); err != nil {
			rows.Close()
			return 0, apperr.Internal(err)
		}
		if err := json.Unmarshal(raw, &entry.tier.Config); err != nil {
			rows.Close()
			return 0, apperr.Internal(err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, apperr.Internal(err)
	}
	rows.Close()

	var total int64
	for _, entry := range entries {
		var granted int64
		if err := s.db.WithinTx(ctx, func(tx *sql.Tx) error {
			var err error
			granted, err = grantScheduled(ctx, tx, entry.memberID, entry.tier, now)
			return err
		}); err != nil {
			return total, err
		}
		total += granted
	}
	return total, nil
}

func grantScheduled(ctx context.Context, tx *sql.Tx, memberID int64, tier tierBenefits, now time.Time) (int64, error) {
	return grantMatching(ctx, tx, memberID, tier, now, func(trigger string) bool {
		return scheduledTrigger(trigger) && scheduledTriggerActive(trigger, now)
	})
}

func grantMatching(
	ctx context.Context,
	tx *sql.Tx,
	memberID int64,
	tier tierBenefits,
	now time.Time,
	match func(string) bool,
) (int64, error) {
	var granted int64
	for _, benefit := range tier.Config.Points {
		if benefit.Amount <= 0 || !match(benefit.Trigger) {
			continue
		}
		key, ok := periodKey(benefit.Period, now)
		if !ok {
			continue
		}
		applied, err := grantPoints(ctx, tx, memberID, tier.TierID, benefit, key, now)
		if err != nil {
			return granted, err
		}
		if applied {
			granted++
		}
	}
	for _, benefit := range tier.Config.Coupons {
		if benefit.Quantity <= 0 || benefit.Quantity > 99 || !match(benefit.Trigger) {
			continue
		}
		key, ok := periodKey(benefit.Period, now)
		if !ok {
			continue
		}
		count, err := grantCoupons(ctx, tx, memberID, tier.TierID, benefit, key, now)
		if err != nil {
			return granted, err
		}
		granted += count
	}
	return granted, nil
}

func loadTierByID(ctx context.Context, tx *sql.Tx, tierID int64) (tierBenefits, error) {
	var raw []byte
	tier := tierBenefits{TierID: tierID}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(benefit_config, JSON_OBJECT())
		FROM membership_tiers WHERE id = ?`, tierID).Scan(&raw); err != nil {
		return tier, apperr.Internal(err)
	}
	if err := json.Unmarshal(raw, &tier.Config); err != nil {
		return tier, apperr.Internal(err)
	}
	return tier, nil
}

func loadCurrentTier(ctx context.Context, tx *sql.Tx, memberID int64) (tierBenefits, bool, error) {
	var tierID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT current_tier_id FROM members WHERE id = ?`, memberID).Scan(&tierID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tierBenefits{}, false, apperr.NotFound("member not found")
		}
		return tierBenefits{}, false, apperr.Internal(err)
	}
	if !tierID.Valid {
		return tierBenefits{}, false, nil
	}
	tier, err := loadTierByID(ctx, tx, tierID.Int64)
	return tier, err == nil, err
}

func isFirstPaidFoodOrder(ctx context.Context, tx *sql.Tx, memberID, paymentOrderID int64, now time.Time) (bool, error) {
	start, end := businessDayBounds(now)
	var firstPaymentOrderID int64
	err := tx.QueryRowContext(ctx, `SELECT po.id
		FROM payment_orders po JOIN business_orders bo ON bo.id = po.business_order_id
		WHERE bo.member_id = ? AND bo.order_type = 'food' AND po.paid_at >= ? AND po.paid_at < ?
		ORDER BY po.paid_at ASC, po.id ASC LIMIT 1`, memberID, start, end).Scan(&firstPaymentOrderID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, apperr.Internal(err)
	}
	return firstPaymentOrderID == paymentOrderID, nil
}

func isInHoursVisit(ctx context.Context, tx *sql.Tx, storeID int64, now time.Time) (bool, error) {
	var hours string
	if err := tx.QueryRowContext(ctx, `SELECT business_hours FROM stores WHERE id = ? AND status = 'active'`, storeID).Scan(&hours); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, apperr.Internal(err)
	}
	return withinBusinessHours(hours, now), nil
}

func grantPoints(ctx context.Context, tx *sql.Tx, memberID, tierID int64, benefit pointBenefit, key string, now time.Time) (bool, error) {
	var accountID, available int64
	const selectAccount = `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = 'points' FOR UPDATE`
	switch err := tx.QueryRowContext(ctx, selectAccount, memberID).Scan(&accountID, &available); {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_accounts
			(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
			VALUES (?, 'points', 0, 0, 0, ?, ?) ON DUPLICATE KEY UPDATE id = id`, memberID, now, now); err != nil {
			return false, apperr.Internal(err)
		}
		if err := tx.QueryRowContext(ctx, selectAccount, memberID).Scan(&accountID, &available); err != nil {
			return false, apperr.Internal(err)
		}
	case err != nil:
		return false, apperr.Internal(err)
	}
	idemKey := benefitKey("p", memberID, tierID, benefit.Period, benefit.Trigger, key, 0)
	newBalance := available + benefit.Amount
	_, err := tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after, reason,
		 source_type, source_id, idem_key, created_at)
		VALUES (?, ?, 'points', 'credit', ?, ?, ?, ?, ?, ?, ?)`,
		accountID, memberID, benefit.Amount, newBalance, grantReason, grantSourceType, tierID, idemKey, now)
	if err != nil {
		if platdb.IsDuplicate(err) {
			return false, nil
		}
		return false, apperr.Internal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE wallet_accounts
		SET available_amount = ?, version = version + 1, updated_at = ? WHERE id = ?`,
		newBalance, now, accountID); err != nil {
		return false, apperr.Internal(err)
	}
	return true, nil
}

func grantCoupons(ctx context.Context, tx *sql.Tx, memberID, tierID int64, benefit couponBenefit, key string, now time.Time) (int64, error) {
	var (
		templateID     int64
		admissionCount int
	)
	query := `SELECT t.id, t.admission_count
		FROM coupon_templates t
		JOIN coupon_categories c ON c.canonical_template_id = t.id
		WHERE c.status = 'active' AND t.scope_type = 'global' AND t.status = 'published'`
	args := make([]any, 0, 1)
	benefitIdentity := benefit.CouponType
	if benefit.CategoryID > 0 {
		query += ` AND c.id = ?`
		args = append(args, benefit.CategoryID)
		benefitIdentity = fmt.Sprintf("category:%d", benefit.CategoryID)
	} else {
		query += ` AND c.business_type = ?`
		args = append(args, benefit.CouponType)
	}
	query += ` ORDER BY c.id ASC LIMIT 1`
	err := tx.QueryRowContext(ctx, query, args...).Scan(&templateID, &admissionCount)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, apperr.Internal(err)
	}
	expiresAt := couponExpiry(benefit.Trigger, now)
	if benefit.ValidityDays > 0 {
		expiresAt = now.UTC().AddDate(0, 0, benefit.ValidityDays)
	}
	var granted int64
	for sequence := 0; sequence < benefit.Quantity; sequence++ {
		idemKey := benefitKey("c", memberID, tierID, benefit.Period, benefit.Trigger+":"+benefitIdentity, key, sequence)
		entitlementNo := compactEntitlementNo(memberID, idemKey)
		_, err := tx.ExecContext(ctx, `INSERT INTO coupon_entitlements
			(entitlement_no, coupon_template_id, admission_count, member_id, store_id, status, rule_version,
			 granted_reason, granted_by_type, expires_at, idem_key, created_at, updated_at)
			VALUES (?, ?, ?, ?, NULL, 'active', 1, ?, 'system', ?, ?, ?, ?)`,
			entitlementNo, templateID, admissionCount, memberID, grantReason, expiresAt, idemKey, now, now)
		if err != nil {
			if platdb.IsDuplicate(err) {
				continue
			}
			return granted, apperr.Internal(err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE coupon_templates
			SET issued_quantity = issued_quantity + 1, updated_at = ? WHERE id = ?`, now, templateID); err != nil {
			return granted, apperr.Internal(err)
		}
		granted++
	}
	return granted, nil
}

func compactEntitlementNo(memberID int64, idemKey string) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(idemKey))
	return fmt.Sprintf("VB%d-%016x", memberID, hash.Sum64())
}
