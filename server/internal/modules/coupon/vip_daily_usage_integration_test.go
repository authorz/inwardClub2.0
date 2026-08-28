package coupon

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func TestClaimGiftDailyUsageIntegration(t *testing.T) {
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

	res, err := tx.ExecContext(ctx, `INSERT INTO members
		(nickname, profile_completed, status, created_at, updated_at)
		VALUES ('vip-daily-usage-integration', 1, 'active', UTC_TIMESTAMP(), UTC_TIMESTAMP())`)
	if err != nil {
		t.Fatal(err)
	}
	memberID, _ := res.LastInsertId()
	var templateID, categoryID int64
	if err := tx.QueryRowContext(ctx, `SELECT id, category_id FROM coupon_templates
		WHERE status = 'published' ORDER BY id LIMIT 1`).Scan(&templateID, &categoryID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO gift_coupon_usage_rules
		(coupon_category_id, daily_limit, created_at, updated_at)
		VALUES (?, 2, UTC_TIMESTAMP(), UTC_TIMESTAMP())
		ON DUPLICATE KEY UPDATE daily_limit = 2, updated_at = UTC_TIMESTAMP()`, categoryID); err != nil {
		t.Fatal(err)
	}

	vip1 := insertUsageTestEntitlement(t, tx, memberID, templateID, "VIP等级福利", "system", 1)
	vip2 := insertUsageTestEntitlement(t, tx, memberID, templateID, "VIP等级福利", "system", 2)
	vip3 := insertUsageTestEntitlement(t, tx, memberID, templateID, "VIP等级福利", "system", 3)
	vip4 := insertUsageTestEntitlement(t, tx, memberID, templateID, "VIP等级福利", "system", 4)
	vip5 := insertUsageTestEntitlement(t, tx, memberID, templateID, "VIP等级福利", "system", 5)
	purchased1 := insertUsageTestEntitlement(t, tx, memberID, templateID, "购买券商品", "purchase", 6)
	purchased2 := insertUsageTestEntitlement(t, tx, memberID, templateID, "购买券商品", "purchase", 7)

	today := time.Date(2026, 8, 25, 23, 59, 0, 0, vipUsageLocation)
	if err := ClaimGiftDailyUsage(ctx, tx, memberID, vip1, today); err != nil {
		t.Fatalf("first VIP coupon: %v", err)
	}
	if err := ClaimGiftDailyUsage(ctx, tx, memberID, vip2, today); err != nil {
		t.Fatalf("second VIP coupon: %v", err)
	}
	if err := ClaimGiftDailyUsage(ctx, tx, memberID, purchased1, today); err != nil {
		t.Fatalf("first purchased coupon: %v", err)
	}
	if err := ClaimGiftDailyUsage(ctx, tx, memberID, purchased2, today); err != nil {
		t.Fatalf("second purchased coupon: %v", err)
	}
	if err := ClaimGiftDailyUsage(ctx, tx, memberID, vip3, today); err == nil || apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("third VIP coupon error = %v, want CONFLICT", err)
	}

	tomorrow := today.Add(2 * time.Minute)
	if err := ClaimGiftDailyUsage(ctx, tx, memberID, vip4, tomorrow); err != nil {
		t.Fatalf("next-day VIP coupon: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE gift_coupon_usage_rules SET daily_limit = NULL WHERE coupon_category_id = ?`, categoryID); err != nil {
		t.Fatal(err)
	}
	if err := ClaimGiftDailyUsage(ctx, tx, memberID, vip5, today); err != nil {
		t.Fatalf("unlimited VIP coupon: %v", err)
	}
	var usageCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM gift_coupon_daily_usages
		WHERE member_id = ?`, memberID).Scan(&usageCount); err != nil {
		t.Fatal(err)
	}
	if usageCount != 3 {
		t.Fatalf("gift usage rows = %d, want 3", usageCount)
	}
}

func insertUsageTestEntitlement(
	t *testing.T,
	tx *sql.Tx,
	memberID, templateID int64,
	reason, grantedBy string,
	sequence int,
) int64 {
	t.Helper()
	now := time.Now().UTC()
	no := fmt.Sprintf("VIPDU-%d-%d", memberID, sequence)
	res, err := tx.Exec(`INSERT INTO coupon_entitlements
		(entitlement_no, coupon_template_id, member_id, status, granted_reason,
		 granted_by_type, created_at, updated_at)
		VALUES (?, ?, ?, 'active', ?, ?, ?, ?)`,
		no, templateID, memberID, reason, grantedBy, now, now)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}
