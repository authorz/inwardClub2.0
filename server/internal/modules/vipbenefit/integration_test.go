package vipbenefit

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
)

func TestFoodAndScheduledBenefitsIntegration(t *testing.T) {
	dsn := os.Getenv("VIP_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("VIP_TEST_MYSQL_DSN is not set")
	}
	ctx := context.Background()
	database, err := platdb.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	var tier4ID, tier5ID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM membership_tiers WHERE level=4 AND status='active' ORDER BY id LIMIT 1`).Scan(&tier4ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM membership_tiers WHERE level=5 AND status='active' ORDER BY id LIMIT 1`).Scan(&tier5ID); err != nil {
		t.Fatal(err)
	}
	now := mustTime(t, "2026-08-24T12:00:00+08:00")
	res, err := tx.ExecContext(ctx, `INSERT INTO members
		(nickname, profile_completed, status, current_tier_id, created_at, updated_at)
		VALUES ('vip-benefit-integration', 1, 'active', ?, ?, ?)`, tier4ID, now, now)
	if err != nil {
		t.Fatal(err)
	}
	memberID, _ := res.LastInsertId()
	orderNo := fmt.Sprintf("VBIT-%d", time.Now().UnixNano())
	res, err = tx.ExecContext(ctx, `INSERT INTO business_orders
		(business_order_no, order_type, store_id, member_id, total_amount_cent,
		 order_status, payment_status, created_at, updated_at)
		VALUES (?, 'food', 1, ?, 8800, 'completed', 'paid', ?, ?)`, orderNo, memberID, now, now)
	if err != nil {
		t.Fatal(err)
	}
	businessOrderID, _ := res.LastInsertId()
	res, err = tx.ExecContext(ctx, `INSERT INTO payment_orders
		(payment_order_no, business_order_id, store_id, member_id, amount_cent,
		 pay_method, status, created_at, updated_at, paid_at)
		VALUES (?, ?, 1, ?, 8800, 'wechat', 'paid', ?, ?, ?)`, orderNo, businessOrderID, memberID, now, now, now)
	if err != nil {
		t.Fatal(err)
	}
	paymentOrderID, _ := res.LastInsertId()

	input := FoodPayment{
		PaymentOrderID: paymentOrderID, BusinessOrderID: businessOrderID,
		MemberID: memberID, StoreID: 1, PaidAt: now, LowSpend: true,
	}
	granted, err := GrantFoodPayment(ctx, tx, input)
	if err != nil {
		t.Fatal(err)
	}
	if granted != 3 { // 3000 points + alcohol coupon + snack coupon.
		t.Fatalf("food grants = %d, want 3", granted)
	}
	if replayed, err := GrantFoodPayment(ctx, tx, input); err != nil || replayed != 0 {
		t.Fatalf("food replay = %d, %v; want 0, nil", replayed, err)
	}
	var points int64
	if err := tx.QueryRowContext(ctx, `SELECT available_amount FROM wallet_accounts
		WHERE member_id=? AND asset_type='points'`, memberID).Scan(&points); err != nil {
		t.Fatal(err)
	}
	if points != 3000 {
		t.Fatalf("points = %d, want 3000", points)
	}
	var foodCoupons int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM coupon_entitlements e
		JOIN coupon_templates ct ON ct.id=e.coupon_template_id
		WHERE e.member_id=? AND ct.coupon_type IN ('alcohol','snack')`, memberID).Scan(&foodCoupons); err != nil {
		t.Fatal(err)
	}
	if foodCoupons != 2 {
		t.Fatalf("food coupons = %d, want 2", foodCoupons)
	}

	tier5, err := loadTierByID(ctx, tx, tier5ID)
	if err != nil {
		t.Fatal(err)
	}
	scheduled, err := grantScheduled(ctx, tx, memberID, tier5, now)
	if err != nil {
		t.Fatal(err)
	}
	if scheduled != 2 {
		t.Fatalf("scheduled grants = %d, want 2", scheduled)
	}
	if replayed, err := grantScheduled(ctx, tx, memberID, tier5, now); err != nil || replayed != 0 {
		t.Fatalf("scheduled replay = %d, %v; want 0, nil", replayed, err)
	}
	assertEventExpiry(t, tx, memberID, "weekly", "weekday_event", "2026-08-29 00:00:00")
	assertEventExpiry(t, tx, memberID, "monthly", "weekly_event", "2026-08-31 00:00:00")
}

func assertEventExpiry(t *testing.T, tx *sql.Tx, memberID int64, period, trigger, wantLocal string) {
	t.Helper()
	pattern := fmt.Sprintf("vipb:c:%d:%%:%s:%s:event_ticket:%%", memberID, period, trigger)
	var expiresAt time.Time
	if err := tx.QueryRow(`SELECT expires_at FROM coupon_entitlements
		WHERE member_id=? AND idem_key LIKE ? ORDER BY id DESC LIMIT 1`, memberID, pattern).Scan(&expiresAt); err != nil {
		t.Fatal(err)
	}
	if got := expiresAt.In(businessLocation).Format("2006-01-02 15:04:05"); got != wantLocal {
		t.Fatalf("%s/%s expiry = %s, want %s", period, trigger, got, wantLocal)
	}
}
