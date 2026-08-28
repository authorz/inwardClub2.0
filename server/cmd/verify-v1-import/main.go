// Command verify-v1-import independently compares the final v1 source with the
// migrated v2 target. It is read-only and exits non-zero on any unapproved diff.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/inwardclub/server/internal/platform/config"
)

type check struct {
	name           string
	source, target int64
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "verify-v1-import error:", err)
		os.Exit(1)
	}
}

func run() error {
	sourceDSN := flag.String("source-dsn", os.Getenv("V1_MYSQL_DSN"), "v1 source MySQL DSN")
	flag.Parse()
	if *sourceDSN == "" {
		return fmt.Errorf("source DSN is required")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	source, err := sql.Open("mysql", *sourceDSN)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		return err
	}
	defer target.Close()
	checks := []check{}
	countPairs := []struct{ name, sourceTable, targetTable string }{
		{"members", "users", "members"}, {"catalog categories", "categories", "catalog_categories"}, {"catalog items", "products", "catalog_items"},
		{"tables", "tables", "tables"}, {"seats", "seats", "seats"}, {"food orders", "food_orders", "food_orders"}, {"food items", "food_order_items", "food_order_items"},
		{"point savings", "save_points", "point_savings"}, {"point withdrawals", "points_withdrawal", "point_withdrawals"},
	}
	for _, pair := range countPairs {
		s, err := scalar(ctx, source, "SELECT COUNT(*) FROM `"+pair.sourceTable+"`")
		if err != nil {
			return err
		}
		t, err := scalar(ctx, target, "SELECT COUNT(*) FROM `"+pair.targetTable+"`")
		if err != nil {
			return err
		}
		checks = append(checks, check{pair.name, s, t})
	}
	legacyCoupons, err := scalar(ctx, source, `SELECT COUNT(*) FROM user_coupon`)
	if err != nil {
		return err
	}
	migratedCoupons, err := scalar(ctx, target, `SELECT COUNT(*) FROM coupon_entitlements WHERE idem_key LIKE 'v1:user_coupon:%'`)
	if err != nil {
		return err
	}
	checks = append(checks, check{"legacy coupons", legacyCoupons, migratedCoupons})
	amountPairs := []struct{ name, sourceQuery, targetQuery string }{
		{"food amount cents", `SELECT CAST(SUM(total_amount)*100 AS SIGNED) FROM food_orders`, `SELECT SUM(total_amount_cent) FROM business_orders WHERE order_type='food'`},
		{"recharge amount cents", `SELECT CAST(SUM(total_fee)*100 AS SIGNED) FROM recharge`, `SELECT SUM(total_amount_cent) FROM business_orders WHERE order_type='recharge'`},
		{"points balance", `SELECT SUM(FLOOR(points)) FROM users`, `SELECT SUM(available_amount) FROM wallet_accounts WHERE asset_type='points'`},
		{"coin balance floored", `SELECT SUM(FLOOR(balance)) FROM users`, `SELECT SUM(available_amount) FROM wallet_accounts WHERE asset_type='coins'`},
		{"growth balance", `SELECT SUM(FLOOR(all_balance)) FROM users`, `SELECT SUM(available_amount) FROM wallet_accounts WHERE asset_type='growth_value'`},
	}
	for _, pair := range amountPairs {
		s, err := scalar(ctx, source, pair.sourceQuery)
		if err != nil {
			return err
		}
		t, err := scalar(ctx, target, pair.targetQuery)
		if err != nil {
			return err
		}
		checks = append(checks, check{pair.name, s, t})
	}
	targetChecks := []struct {
		name, query string
		want        int64
	}{
		{"stores", `SELECT COUNT(*) FROM stores`, 1}, {"admin accounts retained", `SELECT COUNT(*) FROM admin_accounts`, 7},
		{"printers retained", `SELECT COUNT(*) FROM printer_devices`, 2}, {"printers assigned to store 1", `SELECT COUNT(*) FROM printer_devices WHERE store_id<>1`, 0},
		{"activities excluded", `SELECT COUNT(*) FROM activities`, 0}, {"activity orders excluded", `SELECT COUNT(*) FROM activity_orders`, 0},
		{"active legacy alcohol coupons", `SELECT COUNT(*) FROM coupon_entitlements e JOIN coupon_templates t ON t.id=e.coupon_template_id WHERE e.status='active' AND t.coupon_type='alcohol' AND e.idem_key LIKE 'v1:user_coupon:%'`, 1714},
		{"active legacy event coupons", `SELECT COUNT(*) FROM coupon_entitlements e JOIN coupon_templates t ON t.id=e.coupon_template_id WHERE e.status='active' AND t.coupon_type='event_ticket' AND e.idem_key LIKE 'v1:user_coupon:%'`, 4},
		{"completed migration run", `SELECT COUNT(*) FROM migration_runs WHERE run_key='inwardclub-v1-final-20260828' AND status='completed'`, 1},
	}
	for _, item := range targetChecks {
		got, err := scalar(ctx, target, item.query)
		if err != nil {
			return err
		}
		checks = append(checks, check{item.name, item.want, got})
	}
	orphans := []struct{ name, query string }{
		{"wallet member orphans", `SELECT COUNT(*) FROM wallet_accounts x LEFT JOIN members m ON m.id=x.member_id WHERE m.id IS NULL`},
		{"food member orphans", `SELECT COUNT(*) FROM food_orders x LEFT JOIN members m ON m.id=x.member_id WHERE m.id IS NULL`},
		{"food business orphans", `SELECT COUNT(*) FROM food_orders x LEFT JOIN business_orders b ON b.id=x.business_order_id WHERE b.id IS NULL`},
		{"food item order orphans", `SELECT COUNT(*) FROM food_order_items x LEFT JOIN food_orders o ON o.id=x.food_order_id WHERE o.id IS NULL`},
		{"food item catalog orphans", `SELECT COUNT(*) FROM food_order_items x LEFT JOIN catalog_items c ON c.id=x.item_id WHERE c.id IS NULL`},
		{"coupon member orphans", `SELECT COUNT(*) FROM coupon_entitlements x LEFT JOIN members m ON m.id=x.member_id WHERE m.id IS NULL`},
		{"coupon template orphans", `SELECT COUNT(*) FROM coupon_entitlements x LEFT JOIN coupon_templates t ON t.id=x.coupon_template_id WHERE t.id IS NULL`},
		{"staff member orphans", `SELECT COUNT(*) FROM staff_accounts x LEFT JOIN members m ON m.id=x.member_id WHERE m.id IS NULL`},
	}
	for _, item := range orphans {
		got, err := scalar(ctx, target, item.query)
		if err != nil {
			return err
		}
		checks = append(checks, check{item.name, 0, got})
	}
	failed := false
	for _, item := range checks {
		status := "OK"
		if item.source != item.target {
			status = "DIFF"
			failed = true
		}
		fmt.Printf("%-4s %-30s source=%d target=%d\n", status, item.name, item.source, item.target)
	}
	if failed {
		return fmt.Errorf("one or more checks differ")
	}
	return nil
}

func scalar(ctx context.Context, db *sql.DB, query string) (int64, error) {
	var value sql.NullInt64
	if err := db.QueryRowContext(ctx, query).Scan(&value); err != nil {
		return 0, err
	}
	if !value.Valid {
		return 0, nil
	}
	return value.Int64, nil
}
