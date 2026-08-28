package coupon

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	"github.com/joho/godotenv"
)

func TestGrantPurchasedCouponsIntegration(t *testing.T) {
	if os.Getenv("RUN_MYSQL_INTEGRATION") != "1" {
		t.Skip("set RUN_MYSQL_INTEGRATION=1 to run")
	}
	ctx := context.Background()
	_, sourceFile, _, _ := runtime.Caller(0)
	env, err := godotenv.Read(filepath.Join(filepath.Dir(sourceFile), "../../../.env"))
	if err != nil {
		t.Fatalf("load .env: %v", err)
	}
	db, err := platdb.Open(ctx, env["MYSQL_DSN"])
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	var memberID, storeID, templateID int64
	var templateAdmissionCount int
	if err := tx.QueryRowContext(ctx, `SELECT id FROM members ORDER BY id LIMIT 1`).Scan(&memberID); err != nil {
		t.Fatalf("select member: %v", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM stores ORDER BY id LIMIT 1`).Scan(&storeID); err != nil {
		t.Fatalf("select store: %v", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT id, admission_count FROM coupon_templates
		WHERE status = 'published' ORDER BY id LIMIT 1`).Scan(&templateID, &templateAdmissionCount); err != nil {
		t.Fatalf("select coupon template: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	orderNo := fmt.Sprintf("VERIFY-COUPON-%d", now.UnixNano())
	res, err := tx.ExecContext(ctx, `INSERT INTO business_orders
		(business_order_no, order_type, store_id, member_id, total_amount_cent,
		 order_status, payment_status, created_at, updated_at)
		VALUES (?, 'food', ?, ?, 100, 'completed', 'paid', ?, ?)`,
		orderNo, storeID, memberID, now, now)
	if err != nil {
		t.Fatalf("insert business order: %v", err)
	}
	businessOrderID, _ := res.LastInsertId()
	res, err = tx.ExecContext(ctx, `INSERT INTO food_orders
		(business_order_id, store_id, member_id, total_amount_cent, fulfillment_status,
		 remark, created_at, updated_at)
		VALUES (?, ?, ?, 100, 'completed', '', ?, ?)`, businessOrderID, storeID, memberID, now, now)
	if err != nil {
		t.Fatalf("insert food order: %v", err)
	}
	foodOrderID, _ := res.LastInsertId()
	res, err = tx.ExecContext(ctx, `INSERT INTO food_order_items
		(food_order_id, item_id, name_snapshot, unit_price_cent, quantity,
		 pay_channels_snapshot, points_reward_snapshot, coupon_template_id_snapshot,
		 subtotal_cent, created_at)
		VALUES (?, 0, '售券链路校验', 50, 2, JSON_ARRAY('wechat'), 0, ?, 100, ?)`,
		foodOrderID, templateID, now)
	if err != nil {
		t.Fatalf("insert food order item: %v", err)
	}
	lineID, _ := res.LastInsertId()
	paymentOrderID := now.UnixNano()

	granted, err := GrantPurchasedCoupons(ctx, tx, paymentOrderID, businessOrderID, memberID, now)
	if err != nil || granted != 2 {
		t.Fatalf("first grant: granted=%d err=%v", granted, err)
	}
	granted, err = GrantPurchasedCoupons(ctx, tx, paymentOrderID, businessOrderID, memberID, now)
	if err != nil || granted != 0 {
		t.Fatalf("idempotent grant: granted=%d err=%v", granted, err)
	}

	var count int
	var admissionCount int
	var earliestExpiry sql.NullTime
	prefix := fmt.Sprintf("food_coupon:%d:%d:%%", paymentOrderID, lineID)
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*), MIN(expires_at), MIN(admission_count)
		FROM coupon_entitlements WHERE idem_key LIKE ?`, prefix).Scan(&count, &earliestExpiry, &admissionCount); err != nil {
		t.Fatalf("read entitlements: %v", err)
	}
	if count != 2 || admissionCount != templateAdmissionCount || !earliestExpiry.Valid || !earliestExpiry.Time.Equal(now.AddDate(0, 0, 30)) {
		t.Fatalf("unexpected entitlements: count=%d admission=%d expiry=%v", count, admissionCount, earliestExpiry)
	}
}

func TestListActivityUsableCouponsIntegration(t *testing.T) {
	if os.Getenv("RUN_MYSQL_INTEGRATION") != "1" {
		t.Skip("set RUN_MYSQL_INTEGRATION=1 to run")
	}
	ctx := context.Background()
	_, sourceFile, _, _ := runtime.Caller(0)
	env, err := godotenv.Read(filepath.Join(filepath.Dir(sourceFile), "../../../.env"))
	if err != nil {
		t.Fatalf("load .env: %v", err)
	}
	db, err := platdb.Open(ctx, env["MYSQL_DSN"])
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Truncate(time.Second)
	res, err := tx.ExecContext(ctx, `INSERT INTO members
		(nickname, profile_completed, status, created_at, updated_at)
		VALUES ('activity-coupon-list-integration', 1, 'active', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}
	memberID, _ := res.LastInsertId()
	res, err = tx.ExecContext(ctx, `INSERT INTO activities
		(scope_type, title, pay_channels, status, created_at, updated_at)
		VALUES ('global', 'activity coupon list integration', JSON_ARRAY('wechat', 'coupon'), 'published', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("insert activity: %v", err)
	}
	activityID, _ := res.LastInsertId()
	if _, err := tx.ExecContext(ctx, `INSERT INTO activity_ticket_types
		(activity_id, name, admission_count, price_cent, stock_quantity, sold_quantity, pay_channels, status, created_at, updated_at)
		VALUES (?, 'single', 1, 100, 0, 0, JSON_ARRAY('wechat'), 'active', ?, ?)`, activityID, now, now); err != nil {
		t.Fatalf("insert ticket type: %v", err)
	}
	var categoryID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM coupon_categories WHERE business_type = 'admission_ticket' LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("select admission coupon category: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO gift_coupon_usage_rules
		(coupon_category_id, daily_limit, created_at, updated_at)
		VALUES (?, 1, UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE daily_limit = 1, updated_at = UTC_TIMESTAMP()`, categoryID); err != nil {
		t.Fatalf("configure gift usage limit: %v", err)
	}
	res, err = tx.ExecContext(ctx, `INSERT INTO coupon_templates
		(scope_type, name, coupon_type, category_id, admission_count, validity_rule, applicable_scope, status, created_at, updated_at)
		VALUES ('global', 'activity coupon list integration', 'admission_ticket', ?, 1, JSON_OBJECT('days', 30), JSON_OBJECT(), 'published', ?, ?)`, categoryID, now, now)
	if err != nil {
		t.Fatalf("insert template: %v", err)
	}
	templateID, _ := res.LastInsertId()

	purchasedID := insertActivityListEntitlement(t, tx, memberID, templateID, 1, "购买券商品", "purchase", now.Add(time.Hour), 1)
	vipID := insertActivityListEntitlement(t, tx, memberID, templateID, 1, "VIP等级福利", "system", now.Add(time.Hour), 2)
	insertActivityListEntitlement(t, tx, memberID, templateID, 2, "购买券商品", "purchase", now.Add(time.Hour), 3)
	insertActivityListEntitlement(t, tx, memberID, templateID, 1, "购买券商品", "purchase", now.Add(-time.Hour), 4)
	usageDate := now.In(vipUsageLocation).Format("2006-01-02")

	coupons, total, err := listActivityUsableCoupons(ctx, tx, memberID, activityID, now, usageDate, 20, 0)
	if err != nil {
		t.Fatalf("list before VIP use: %v", err)
	}
	if total != 2 || len(coupons) != 2 {
		t.Fatalf("before VIP use got total=%d coupons=%+v", total, coupons)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO gift_coupon_daily_usages
		(member_id, category_id, usage_date, slot_number, entitlement_id, created_at)
		VALUES (?, ?, ?, 1, ?, ?)`, memberID, categoryID, usageDate, vipID, now); err != nil {
		t.Fatalf("insert VIP daily usage: %v", err)
	}
	coupons, total, err = listActivityUsableCoupons(ctx, tx, memberID, activityID, now, usageDate, 20, 0)
	if err != nil {
		t.Fatalf("list after VIP use: %v", err)
	}
	if total != 1 || len(coupons) != 1 || coupons[0].EntitlementID != purchasedID {
		t.Fatalf("after VIP use got total=%d coupons=%+v", total, coupons)
	}
}

func insertActivityListEntitlement(
	t *testing.T,
	tx *sql.Tx,
	memberID, templateID int64,
	admissionCount int,
	reason, grantedBy string,
	expiresAt time.Time,
	sequence int,
) int64 {
	t.Helper()
	now := time.Now().UTC()
	no := fmt.Sprintf("ACTIVITY-LIST-%d-%d", memberID, sequence)
	res, err := tx.Exec(`INSERT INTO coupon_entitlements
		(entitlement_no, coupon_template_id, admission_count, member_id, status, granted_reason,
		 granted_by_type, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', ?, ?, ?, ?, ?)`,
		no, templateID, admissionCount, memberID, reason, grantedBy, expiresAt, now, now)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}
