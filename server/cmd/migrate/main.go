// Command migrate applies goose migrations from the embedded filesystem.
//
// Usage:
//
//	go run ./cmd/migrate up|down|status|version|reset
//	go run ./cmd/migrate seed   # inserts dev accounts + sample store (dev only)
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"

	appdb "github.com/inwardclub/server/db"
	"github.com/inwardclub/server/internal/platform/config"
)

const migrationsDir = "migrations"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.MySQLDSN == "" {
		return fmt.Errorf("MYSQL_DSN is required")
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	if command == "seed" {
		return seed(db)
	}

	goose.SetBaseFS(appdb.Migrations)
	if err := goose.SetDialect("mysql"); err != nil {
		return err
	}

	switch command {
	case "up":
		return goose.Up(db, migrationsDir)
	case "down":
		return goose.Down(db, migrationsDir)
	case "status":
		return goose.Status(db, migrationsDir)
	case "version":
		return goose.Version(db, migrationsDir)
	case "reset":
		return goose.Reset(db, migrationsDir)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

// seed inserts a minimal dev dataset: one store, a super_admin, a store_admin,
// a catalog category + published item, and a published activity + active ticket
// type (enough for the /mini food-order and activity-order happy paths).
// Passwords are "password". Safe to re-run (idempotent via INSERT IGNORE).
func seed(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	now := time.Now().UTC()
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO stores
		(id, name, address, business_hours, status, created_at, updated_at)
		VALUES (1, '示例门店', '示例地址 1 号', '10:00-22:00', 'active', ?, ?)`, now, now); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO admin_accounts
		(username, password_hash, display_name, role, store_id, status, created_at, updated_at)
		VALUES ('superadmin', ?, '总后台管理员', 'super_admin', NULL, 'active', ?, ?)`, string(hash), now, now); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO admin_accounts
		(username, password_hash, display_name, role, store_id, status, created_at, updated_at)
		VALUES ('storeadmin', ?, '门店管理员', 'store_admin', 1, 'active', ?, ?)`, string(hash), now, now); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO admin_accounts
		(username, password_hash, display_name, role, store_id, status, created_at, updated_at)
		VALUES ('cashierdemo', ?, '演示收银员', 'cashier', 1, 'active', ?, ?)`, string(hash), now, now); err != nil {
		return err
	}

	// Staff: one staff_accounts row bound to store#1, keyed by a deterministic
	// fake wechat_openid so re-running seed doesn't duplicate the row.
	const seedStaffOpenID = "seed_staff_openid_001"
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO staff_accounts
		(member_id, wechat_openid, name, store_id, status, created_at, updated_at)
		VALUES (NULL, ?, '演示工作人员', 1, 'active', ?, ?)`, seedStaffOpenID, now, now); err != nil {
		return err
	}

	// Catalog: one active category + one published item scoped to store#1 so
	// GET /mini/menu returns data and POST /mini/food-orders has an item to buy.
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO catalog_categories
		(id, scope_type, store_id, parent_id, name, asset_id, sort_order, status, created_at, updated_at)
		VALUES (1, 'store', 1, NULL, '示例分类', NULL, 0, 'active', ?, ?)`, now, now); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO catalog_items
		(id, scope_type, store_id, source_item_id, category_id, name, description, asset_id,
		 item_type, price_cent, stock_quantity, pay_channels, points_reward, sort_order, status, created_at, updated_at)
		VALUES (1, 'store', 1, NULL, 1, '示例商品', '联调用示例商品', NULL,
		 'food', 1000, 0, '["wechat","coin"]', 0, 0, 'published', ?, ?)`, now, now); err != nil {
		return err
	}

	// Activity: one published activity + one active ticket type scoped to store#1
	// so GET /mini/activities returns data and POST /mini/activity-orders works.
	end := now.Add(30 * 24 * time.Hour)
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO activities
		(id, scope_type, store_id, source_activity_id, title, description, content, asset_id,
		 start_at, end_at, pay_channels, purchase_limit_per_member, assigned_store_ids, status, created_at, updated_at)
		VALUES (1, 'store', 1, NULL, '示例活动', '联调用示例活动', NULL, NULL,
		 ?, ?, '["wechat","coin"]', 0, NULL, 'published', ?, ?)`, now, end, now, now); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO activity_ticket_types
		(id, activity_id, session_id, store_id, name, price_cent, stock_quantity, sold_quantity,
		 sale_start_at, sale_end_at, pay_channels, max_tickets_per_order, status, created_at, updated_at)
		VALUES (1, 1, NULL, 1, '示例票种', 2000, 0, 0,
		 ?, ?, '["wechat","coin"]', 0, 'active', ?, ?)`, now, end, now, now); err != nil {
		return err
	}

	// Member: one sample member so coupon entitlements have an owner and the
	// /mini "my coupons" page renders a row. The openid is the deterministic fake
	// openid the mini login (FakeWeChatClient) derives from login code
	// "seed-user-001" — sha1("openid:seed-user-001")[:8] — so signing in with that
	// code resolves to THIS member and sees the seeded coupons below.
	const seedLoginCode = "seed-user-001"
	const seedLoginOpenID = "fake_openid_9cb72405ead00a32"
	// Resolve the sample member BY its deterministic fake openid, then reuse
	// that member's id for every seeded entitlement/redemption below. A plain
	// `INSERT IGNORE` keyed on id=1 was the bug: if a member row already existed
	// (from an earlier seed with a stale openid, or one the mini login created
	// for this code), IGNORE silently skipped it and the fake openid never
	// landed on any member — so signing in with "seed-user-001" kept minting a
	// brand-new member instead of hitting the seeded one. Query-then-insert
	// keyed on the openid is idempotent and guarantees the login lands here.
	var memberID int64
	err = db.QueryRowContext(ctx, `SELECT id FROM members WHERE wechat_openid = ?`, seedLoginOpenID).Scan(&memberID)
	if err == sql.ErrNoRows {
		res, ierr := db.ExecContext(ctx, `INSERT INTO members
			(wechat_openid, nickname, invite_code, status, created_at, updated_at)
			VALUES (?, '示例会员', 'SEED0001', 'active', ?, ?)`, seedLoginOpenID, now, now)
		if ierr != nil {
			return ierr
		}
		if memberID, err = res.LastInsertId(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Coupon: one published exchange template scoped to store#1 so the console
	// coupon list has a row. applicable_scope points at catalog_item#1;
	// validity_rule is an empty object (NOT NULL JSON column, richer rules not seeded).
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO coupon_templates
		(id, scope_type, store_id, name, description, coupon_type, value_cent, points_price,
		 stock_quantity, issued_quantity, validity_rule, applicable_scope, per_member_limit,
		 status, created_at, updated_at)
		VALUES (1, 'store', 1, '示例优惠券', '联调用示例兑换券', 'exchange', 0, 0,
		 100, 1, '{}', '{"itemIds":[1]}', 0, 'published', ?, ?)`, now, now); err != nil {
		return err
	}
	// Entitlement: grant template#1 to member#1 so ListMemberCoupons returns a row.
	if _, err := db.ExecContext(ctx, `INSERT INTO coupon_entitlements
		(id, entitlement_no, coupon_template_id, member_id, store_id, status, rule_version,
		 granted_reason, granted_by_type, expires_at, created_at, updated_at)
		VALUES (1, 'SEED-ENT-0001', 1, ?, 1, 'active', 1,
		 'seed', 'system', ?, ?, ?)
		ON DUPLICATE KEY UPDATE member_id = VALUES(member_id)`, memberID, end, now, now); err != nil {
		return err
	}

	// A second, already-used entitlement + its redemption so the /mini
	// "兑换订单" list (GET /mini/coupon-redemptions) and detail render a row too.
	if _, err := db.ExecContext(ctx, `INSERT INTO coupon_entitlements
		(id, entitlement_no, coupon_template_id, member_id, store_id, status, rule_version,
		 granted_reason, granted_by_type, expires_at, created_at, updated_at)
		VALUES (2, 'SEED-ENT-0002', 1, ?, 1, 'used', 1,
		 'seed', 'system', ?, ?, ?)
		ON DUPLICATE KEY UPDATE member_id = VALUES(member_id)`, memberID, end, now, now); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO coupon_redemptions
		(id, redemption_no, entitlement_id, coupon_template_id, member_id, store_id,
		 verified_by_type, created_at)
		VALUES (1, 'SEED-RD-0001', 2, 1, ?, 1, 'seed', ?)
		ON DUPLICATE KEY UPDATE member_id = VALUES(member_id)`, memberID, now); err != nil {
		return err
	}

	fmt.Printf("seed login code (mini fake WeChat): %s -> openid %s -> member#%d\n", seedLoginCode, seedLoginOpenID, memberID)
	fmt.Println("seed complete: store#1, superadmin/storeadmin/cashierdemo (password: password),")
	fmt.Println("  staff_account (演示工作人员, openid seed_staff_openid_001) -> store#1,")
	fmt.Println("  catalog_category#1, catalog_item#1 (published, 10.00),")
	fmt.Println("  activity#1 (published), activity_ticket_type#1 (active, 20.00),")
	fmt.Printf("  member#%d (示例会员), coupon_template#1 (published exchange),\n", memberID)
	fmt.Printf("  coupon_entitlement#1 (SEED-ENT-0001, active) -> member#%d,\n", memberID)
	fmt.Printf("  coupon_entitlement#2 (SEED-ENT-0002, used) + coupon_redemption#1 (SEED-RD-0001) -> member#%d\n", memberID)
	return nil
}
