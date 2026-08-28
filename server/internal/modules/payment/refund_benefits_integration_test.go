package payment

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/modules/vipbenefit"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func TestRefundBenefitsIntegration(t *testing.T) {
	dsn := os.Getenv("REFUND_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set REFUND_TEST_MYSQL_DSN to an isolated MySQL database")
	}
	ctx := context.Background()
	database, err := platdb.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open isolated database: %v", err)
	}
	defer database.Close()
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	fixture := createRefundBenefitFixture(t, ctx, tx)
	legacyLedgerID := insertLegacyVIPGrant(t, ctx, tx, fixture)
	legacyErr := prepareRefundBenefits(
		ctx, tx, fixture.refundID, fixture.paymentOrderID, fixture.businessOrderID,
		1000, 1000, orderTypeRecharge, fixture.now,
	)
	if apperr.From(legacyErr).Code != apperr.CodeConflict {
		t.Fatalf("legacy VIP refund error = %v, want conflict", legacyErr)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM wallet_ledger_entries WHERE id = ?`, legacyLedgerID); err != nil {
		t.Fatalf("remove legacy VIP fixture: %v", err)
	}

	partialErr := prepareRefundBenefits(
		ctx, tx, fixture.refundID, fixture.paymentOrderID, fixture.businessOrderID,
		500, 1000, orderTypeRecharge, fixture.now,
	)
	if apperr.From(partialErr).Code != apperr.CodeConflict {
		t.Fatalf("partial recharge refund error = %v, want conflict", partialErr)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE coupon_entitlements SET status = 'used'
		WHERE id = ?`, fixture.couponEntitlementIDs[0]); err != nil {
		t.Fatalf("mark coupon used: %v", err)
	}
	usedCouponErr := prepareRefundBenefits(
		ctx, tx, fixture.refundID, fixture.paymentOrderID, fixture.businessOrderID,
		1000, 1000, orderTypeRecharge, fixture.now,
	)
	if apperr.From(usedCouponErr).Code != apperr.CodeConflict {
		t.Fatalf("used coupon refund error = %v, want conflict", usedCouponErr)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE coupon_entitlements SET status = 'active'
		WHERE id = ?`, fixture.couponEntitlementIDs[0]); err != nil {
		t.Fatalf("restore coupon: %v", err)
	}

	if err := prepareRefundBenefits(
		ctx, tx, fixture.refundID, fixture.paymentOrderID, fixture.businessOrderID,
		1000, 1000, orderTypeRecharge, fixture.now,
	); err != nil {
		t.Fatalf("prepare benefits: %v", err)
	}
	assertRefundBenefitState(t, ctx, tx, fixture, map[string]int64{
		"coins": -7, "points": -3, "growth_value": 0,
	}, refundPendingCouponStatus)
	assertRefundClawbackAmount(t, ctx, tx, fixture.refundID, "points", 10)

	memberID := sql.NullInt64{Int64: fixture.memberID, Valid: true}
	if err := rollbackRefundBenefits(
		ctx, tx, fixture.refundID, fixture.paymentOrderID, fixture.businessOrderID,
		fixture.now.Add(time.Minute),
	); err != nil {
		t.Fatalf("rollback benefits: %v", err)
	}
	assertRefundBenefitState(t, ctx, tx, fixture, map[string]int64{
		"coins": 3, "points": 7, "growth_value": 2,
	}, "active")

	secondRefundID := insertRefundFixture(t, ctx, tx, fixture, "RF-BENEFIT-COMPLETE")
	if err := prepareRefundBenefits(
		ctx, tx, secondRefundID, fixture.paymentOrderID, fixture.businessOrderID,
		1000, 1000, orderTypeRecharge, fixture.now.Add(2*time.Minute),
	); err != nil {
		t.Fatalf("prepare benefits for completion: %v", err)
	}
	if err := completeRefundBenefits(
		ctx, tx, secondRefundID, fixture.paymentOrderID, fixture.businessOrderID,
		memberID, fixture.now.Add(3*time.Minute),
	); err != nil {
		t.Fatalf("complete benefits: %v", err)
	}
	assertRefundBenefitState(t, ctx, tx, fixture, map[string]int64{
		"coins": -7, "points": -3, "growth_value": 0,
	}, "void")
	assertRefundClawbackAmount(t, ctx, tx, secondRefundID, "points", 10)
	var currentTierID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT current_tier_id FROM members WHERE id = ?`, fixture.memberID).
		Scan(&currentTierID); err != nil {
		t.Fatalf("read current tier: %v", err)
	}
	if !currentTierID.Valid || currentTierID.Int64 != fixture.baseTierID {
		t.Fatalf("current tier = %+v, want base tier %d", currentTierID, fixture.baseTierID)
	}
}

type refundBenefitFixture struct {
	now                      time.Time
	memberID, baseTierID     int64
	businessOrderID          int64
	paymentOrderID, refundID int64
	highTierID               int64
	couponEntitlementIDs     []int64
}

func createRefundBenefitFixture(t *testing.T, ctx context.Context, tx *sql.Tx) refundBenefitFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	var baseTierID, highTierID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM membership_tiers
		WHERE status = 'active' ORDER BY threshold ASC, level ASC LIMIT 1`).Scan(&baseTierID); err != nil {
		t.Fatalf("select base tier: %v", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM membership_tiers
		WHERE status = 'active' ORDER BY threshold DESC, level DESC LIMIT 1`).Scan(&highTierID); err != nil {
		t.Fatalf("select high tier: %v", err)
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO members
		(nickname, current_tier_id, status, created_at, updated_at)
		VALUES ('退款权益集成测试', ?, 'active', ?, ?)`, highTierID, now, now)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}
	memberID, _ := result.LastInsertId()
	orderNo := fmt.Sprintf("BO-REFUND-BENEFIT-%d", now.UnixNano())
	result, err = tx.ExecContext(ctx, `INSERT INTO business_orders
		(business_order_no, order_type, member_id, total_amount_cent,
		 order_status, payment_status, created_at, updated_at)
		VALUES (?, 'recharge', ?, 1000, 'completed', 'paid', ?, ?)`, orderNo, memberID, now, now)
	if err != nil {
		t.Fatalf("insert business order: %v", err)
	}
	businessOrderID, _ := result.LastInsertId()
	result, err = tx.ExecContext(ctx, `INSERT INTO payment_orders
		(payment_order_no, business_order_id, member_id, amount_cent, pay_method,
		 status, created_at, updated_at, paid_at)
		VALUES (?, ?, ?, 1000, 'wechat', 'paid', ?, ?, ?)`,
		"PO-"+orderNo, businessOrderID, memberID, now, now, now)
	if err != nil {
		t.Fatalf("insert payment order: %v", err)
	}
	paymentOrderID, _ := result.LastInsertId()
	fixture := refundBenefitFixture{
		now: now, memberID: memberID, baseTierID: baseTierID,
		businessOrderID: businessOrderID, paymentOrderID: paymentOrderID, highTierID: highTierID,
	}
	fixture.refundID = insertRefundFixture(t, ctx, tx, fixture, "RF-BENEFIT-ROLLBACK")

	grants := []struct {
		asset, source    string
		available, grant int64
	}{
		{asset: "coins", source: "recharge_order", available: 3, grant: 10},
		{asset: "points", source: "first_recharge_reward", available: 1, grant: 4},
		{asset: "growth_value", source: "wechat_payment_growth", available: 2, grant: 2},
	}
	for index, grant := range grants {
		result, err := tx.ExecContext(ctx, `INSERT INTO wallet_accounts
			(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
			VALUES (?, ?, ?, 0, 0, ?, ?)`, memberID, grant.asset, grant.available, now, now)
		if err != nil {
			t.Fatalf("insert %s account: %v", grant.asset, err)
		}
		accountID, _ := result.LastInsertId()
		if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries
			(account_id, member_id, asset_type, direction, amount, balance_after, reason,
			 source_type, source_id, idem_key, created_at)
			VALUES (?, ?, ?, 'credit', ?, ?, 'refund_test_grant', ?, ?, ?, ?)`,
			accountID, memberID, grant.asset, grant.grant, grant.available,
			grant.source, businessOrderID, fmt.Sprintf("refund-test-grant:%d:%d", businessOrderID, index), now,
		); err != nil {
			t.Fatalf("insert %s grant: %v", grant.asset, err)
		}
	}
	result, err = tx.ExecContext(ctx, `INSERT INTO coupon_entitlements
		(entitlement_no, coupon_template_id, member_id, status, granted_reason,
		 granted_by_type, idem_key, created_at, updated_at)
		VALUES (?, 1, ?, 'active', '充值赠券', 'recharge', ?, ?, ?)`,
		fmt.Sprintf("REFUND-TEST-%d", paymentOrderID), memberID,
		fmt.Sprintf("recharge_coupon:%d", paymentOrderID), now, now)
	if err != nil {
		t.Fatalf("insert coupon entitlement: %v", err)
	}
	rechargeCouponID, _ := result.LastInsertId()
	fixture.couponEntitlementIDs = append(fixture.couponEntitlementIDs, rechargeCouponID)
	grantVIPUpgradeBenefits(t, ctx, tx, &fixture)
	return fixture
}

func grantVIPUpgradeBenefits(t *testing.T, ctx context.Context, tx *sql.Tx, fixture *refundBenefitFixture) {
	t.Helper()
	var categoryID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM coupon_categories
		WHERE status = 'active' AND canonical_template_id IS NOT NULL ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("select VIP coupon category: %v", err)
	}
	config := fmt.Sprintf(`{
		"points":[{"amount":6,"period":"once","trigger":"tier_achieved"}],
		"coupons":[{"categoryId":%d,"quantity":1,"period":"daily","trigger":"period_start"}],
		"descriptions":[]
	}`, categoryID)
	if _, err := tx.ExecContext(ctx, `UPDATE membership_tiers SET benefit_config = ?, updated_at = ? WHERE id = ?`,
		config, fixture.now, fixture.highTierID); err != nil {
		t.Fatalf("configure VIP upgrade benefits: %v", err)
	}
	if err := vipbenefit.GrantTierReachedForOrder(
		ctx, tx, fixture.memberID, fixture.highTierID, fixture.businessOrderID, fixture.now,
	); err != nil {
		t.Fatalf("grant VIP upgrade benefits: %v", err)
	}
	var couponID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM coupon_entitlements
		WHERE member_id = ? AND granted_by_id = ? AND granted_reason = 'VIP等级福利'`,
		fixture.memberID, fixture.businessOrderID,
	).Scan(&couponID); err != nil {
		t.Fatalf("read VIP upgrade coupon: %v", err)
	}
	fixture.couponEntitlementIDs = append(fixture.couponEntitlementIDs, couponID)
}

func insertLegacyVIPGrant(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	fixture refundBenefitFixture,
) int64 {
	t.Helper()
	var accountID, balance int64
	if err := tx.QueryRowContext(ctx, `SELECT id, available_amount FROM wallet_accounts
		WHERE member_id = ? AND asset_type = 'points'`, fixture.memberID).Scan(&accountID, &balance); err != nil {
		t.Fatalf("read points account: %v", err)
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries
		(account_id, member_id, asset_type, direction, amount, balance_after, reason,
		 source_type, source_id, idem_key, created_at)
		VALUES (?, ?, 'points', 'credit', 1, ?, 'VIP等级福利', 'vip_benefit', ?, ?, ?)`,
		accountID, fixture.memberID, balance, fixture.highTierID,
		fmt.Sprintf("legacy-vip-refund-test:%d", fixture.businessOrderID), fixture.now)
	if err != nil {
		t.Fatalf("insert legacy VIP grant: %v", err)
	}
	ledgerID, _ := result.LastInsertId()
	return ledgerID
}

func insertRefundFixture(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	fixture refundBenefitFixture,
	prefix string,
) int64 {
	t.Helper()
	result, err := tx.ExecContext(ctx, `INSERT INTO refund_orders
		(refund_order_no, payment_order_id, business_order_id, amount_cent, channel,
		 status, reason, requested_by_type, created_at, updated_at)
		VALUES (?, ?, ?, 1000, 'wechat', 'processing', '集成测试', 'test', ?, ?)`,
		fmt.Sprintf("%s-%d", prefix, fixture.now.UnixNano()), fixture.paymentOrderID,
		fixture.businessOrderID, fixture.now, fixture.now)
	if err != nil {
		t.Fatalf("insert refund: %v", err)
	}
	refundID, _ := result.LastInsertId()
	return refundID
}

func assertRefundBenefitState(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	fixture refundBenefitFixture,
	wantBalances map[string]int64,
	wantCouponStatus string,
) {
	t.Helper()
	for asset, want := range wantBalances {
		var got int64
		if err := tx.QueryRowContext(ctx, `SELECT available_amount FROM wallet_accounts
			WHERE member_id = ? AND asset_type = ?`, fixture.memberID, asset).Scan(&got); err != nil {
			t.Fatalf("read %s balance: %v", asset, err)
		}
		if got != want {
			t.Fatalf("%s balance = %d, want %d", asset, got, want)
		}
	}
	for _, couponID := range fixture.couponEntitlementIDs {
		var couponStatus string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM coupon_entitlements WHERE id = ?`,
			couponID,
		).Scan(&couponStatus); err != nil {
			t.Fatalf("read coupon %d status: %v", couponID, err)
		}
		if couponStatus != wantCouponStatus {
			t.Fatalf("coupon %d status = %q, want %q", couponID, couponStatus, wantCouponStatus)
		}
	}
}

func assertRefundClawbackAmount(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	refundID int64,
	assetType string,
	want int64,
) {
	t.Helper()
	var amount int64
	if err := tx.QueryRowContext(ctx, `SELECT amount FROM wallet_ledger_entries
		WHERE source_type = ? AND source_id = ? AND reason = ? AND asset_type = ?`,
		refundBenefitSource, refundID, refundBenefitReason, assetType,
	).Scan(&amount); err != nil {
		t.Fatalf("read %s clawback: %v", assetType, err)
	}
	if amount != want {
		t.Fatalf("%s clawback = %d, want %d", assetType, amount, want)
	}
}
