package coupon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/joho/godotenv"

	platdb "github.com/inwardclub/server/internal/platform/db"
)

func TestUseEventCouponIntegration(t *testing.T) {
	if os.Getenv("RUN_MYSQL_INTEGRATION") != "1" {
		t.Skip("set RUN_MYSQL_INTEGRATION=1 to run")
	}
	ctx := context.Background()
	_, sourceFile, _, _ := runtime.Caller(0)
	env, err := godotenv.Read(filepath.Join(filepath.Dir(sourceFile), "../../../.env"))
	if err != nil {
		t.Fatalf("load .env: %v", err)
	}
	database, err := platdb.Open(ctx, env["MYSQL_DSN"])
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer database.Close()

	now := time.Now().UTC().Truncate(time.Second)
	suffix := now.UnixNano()
	storeResult, err := database.ExecContext(ctx, `INSERT INTO stores
		(name, status, created_at, updated_at) VALUES (?, 'active', ?, ?)`,
		fmt.Sprintf("event-coupon-integration-%d", suffix), now, now)
	if err != nil {
		t.Fatalf("insert store: %v", err)
	}
	storeID, _ := storeResult.LastInsertId()
	memberResult, err := database.ExecContext(ctx, `INSERT INTO members
		(nickname, profile_completed, status, created_at, updated_at)
		VALUES (?, 1, 'active', ?, ?)`, fmt.Sprintf("event-coupon-integration-%d", suffix), now, now)
	if err != nil {
		database.ExecContext(ctx, `DELETE FROM stores WHERE id = ?`, storeID)
		t.Fatalf("insert member: %v", err)
	}
	memberID, _ := memberResult.LastInsertId()
	var templateID, entitlementID, redemptionID int64
	idemKey := fmt.Sprintf("event-use-integration-%d", suffix)
	defer func() {
		if redemptionID > 0 {
			database.ExecContext(ctx, `DELETE FROM coupon_redemptions WHERE id = ?`, redemptionID)
		}
		database.ExecContext(ctx, `DELETE FROM idempotency_keys WHERE scope = 'mini/event-coupon-redemptions' AND idem_key = ?`, idemKey)
		if entitlementID > 0 {
			database.ExecContext(ctx, `DELETE FROM coupon_entitlements WHERE id = ?`, entitlementID)
		}
		if templateID > 0 {
			database.ExecContext(ctx, `DELETE FROM coupon_templates WHERE id = ?`, templateID)
		}
		database.ExecContext(ctx, `DELETE FROM members WHERE id = ?`, memberID)
		database.ExecContext(ctx, `DELETE FROM stores WHERE id = ?`, storeID)
	}()

	var categoryID int64
	if err := database.QueryRowContext(ctx, `SELECT id FROM coupon_categories WHERE business_type = 'event_ticket' LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("select event category: %v", err)
	}
	templateResult, err := database.ExecContext(ctx, `INSERT INTO coupon_templates
		(scope_type, name, coupon_type, category_id, admission_count, validity_rule,
		 applicable_scope, status, created_at, updated_at)
		VALUES ('global', ?, 'event_ticket', ?, 1, JSON_OBJECT('days', 30), JSON_OBJECT(), 'published', ?, ?)`,
		fmt.Sprintf("event-coupon-integration-%d", suffix), categoryID, now, now)
	if err != nil {
		t.Fatalf("insert template: %v", err)
	}
	templateID, _ = templateResult.LastInsertId()
	entitlementResult, err := database.ExecContext(ctx, `INSERT INTO coupon_entitlements
		(entitlement_no, coupon_template_id, admission_count, member_id, store_id, status,
		 granted_reason, granted_by_type, expires_at, created_at, updated_at)
		VALUES (?, ?, 1, ?, ?, 'active', '集成测试', 'system', ?, ?, ?)`,
		fmt.Sprintf("EVENT-USE-%d", suffix), templateID, memberID, storeID, now.Add(time.Hour), now, now)
	if err != nil {
		t.Fatalf("insert entitlement: %v", err)
	}
	entitlementID, _ = entitlementResult.LastInsertId()

	used, err := NewRepository(database).UseEventCoupon(ctx, UseEventCouponInput{
		MemberID: memberID, EntitlementID: entitlementID, StoreID: storeID,
		RedemptionNo: fmt.Sprintf("ER-TEST-%d", suffix), IdemKey: idemKey, Now: now,
		RuleJSON: []byte(`{"couponType":"event_ticket","redeemedAmountCent":0}`),
	})
	if err != nil {
		t.Fatalf("use event coupon: %v", err)
	}
	redemptionID = used.RedemptionID
	if redemptionID <= 0 || used.Status != StatusUsed || used.CouponType != TypeEventTicket {
		t.Fatalf("unexpected used coupon: %+v", used)
	}
	detail, err := NewRepository(database).GetRedemption(ctx, memberID, redemptionID)
	if err != nil {
		t.Fatalf("get redemption: %v", err)
	}
	if detail.CouponType != TypeEventTicket || detail.StoreName == "" || detail.Status != StatusUsed {
		t.Fatalf("unexpected redemption detail: %+v", detail)
	}
}
