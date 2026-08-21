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
	if err := tx.QueryRowContext(ctx, `SELECT id FROM members ORDER BY id LIMIT 1`).Scan(&memberID); err != nil {
		t.Fatalf("select member: %v", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM stores ORDER BY id LIMIT 1`).Scan(&storeID); err != nil {
		t.Fatalf("select store: %v", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM coupon_templates
		WHERE status = 'published' ORDER BY id LIMIT 1`).Scan(&templateID); err != nil {
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
	var earliestExpiry sql.NullTime
	prefix := fmt.Sprintf("food_coupon:%d:%d:%%", paymentOrderID, lineID)
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*), MIN(expires_at)
		FROM coupon_entitlements WHERE idem_key LIKE ?`, prefix).Scan(&count, &earliestExpiry); err != nil {
		t.Fatalf("read entitlements: %v", err)
	}
	if count != 2 || !earliestExpiry.Valid || !earliestExpiry.Time.Equal(now.AddDate(0, 0, 30)) {
		t.Fatalf("unexpected entitlements: count=%d expiry=%v", count, earliestExpiry)
	}
}
