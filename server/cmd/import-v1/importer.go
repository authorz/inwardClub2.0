package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/inwardclub/server/internal/modules/asset"
	"github.com/inwardclub/server/internal/platform/config"
	appdb "github.com/inwardclub/server/internal/platform/db"
)

const sourceSnapshot = "inwardclub-back-2026-08-28"

type importer struct {
	source       *sql.DB
	target       *appdb.DB
	runKey       string
	sourceCounts map[string]int64
	targetCounts map[string]int64
	metrics      map[string]int64
	assetStore   asset.ObjectStore
	appEnv       string
	sourceBase   string
	skipAssets   bool
	assetIDs     map[string]int64
	backupPath   string
	backupSHA256 string
}

type importReport struct {
	RunKey        string           `json:"runKey,omitempty"`
	Executed      bool             `json:"executed"`
	GeneratedAt   time.Time        `json:"generatedAt"`
	SourceCounts  map[string]int64 `json:"sourceCounts"`
	TargetCounts  map[string]int64 `json:"targetCounts,omitempty"`
	Metrics       map[string]int64 `json:"metrics"`
	ApprovedDiffs []string         `json:"approvedDiffs"`
	BackupPath    string           `json:"backupPath,omitempty"`
	BackupSHA256  string           `json:"backupSha256,omitempty"`
}

func newImporter(source *sql.DB, target *appdb.DB, runKey string, cfg *config.Config, sourceBase string, skipAssets bool) *importer {
	return &importer{source: source, target: target, runKey: runKey,
		sourceCounts: map[string]int64{}, targetCounts: map[string]int64{}, metrics: map[string]int64{},
		assetStore: asset.NewQiniuObjectStore(cfg.Qiniu), appEnv: cfg.AppEnv, sourceBase: sourceBase,
		skipAssets: skipAssets, assetIDs: map[string]int64{}}
}

func (i *importer) report(executed bool) importReport {
	return importReport{
		RunKey: i.runKey, Executed: executed, GeneratedAt: time.Now().UTC(),
		SourceCounts: i.sourceCounts, TargetCounts: i.targetCounts, Metrics: i.metrics,
		BackupPath: i.backupPath, BackupSHA256: i.backupSHA256,
		ApprovedDiffs: []string{
			"v1 activities has zero rows; 1,479 activity_orders are intentionally excluded by owner approval",
			"v1 admin accounts are not imported; the seven existing v2 admin accounts are retained",
			"v1 jobs, sessions, cache and Laravel migration metadata are not application business data and are excluded",
			"v1 balance_consumption_records and user_points are archived instead of replayed to avoid duplicating the canonical transaction_records ledger",
		},
	}
}

var sourceBusinessTables = []string{
	"stores", "users", "categories", "products", "tables", "seats", "food_orders",
	"food_order_items", "recharge", "transaction_records", "balance_consumption_records",
	"user_points", "save_points", "points_withdrawal", "user_coupon", "banner", "activities", "activity_orders",
}

func (i *importer) preflight(ctx context.Context) error {
	for _, table := range sourceBusinessTables {
		var count int64
		if err := i.source.QueryRowContext(ctx, "SELECT COUNT(*) FROM `"+table+"`").Scan(&count); err != nil {
			return fmt.Errorf("count v1 %s: %w", table, err)
		}
		i.sourceCounts[table] = count
	}
	if i.sourceCounts["users"] == 0 || i.sourceCounts["stores"] != 1 {
		return fmt.Errorf("unexpected v1 identity baseline: users=%d stores=%d", i.sourceCounts["users"], i.sourceCounts["stores"])
	}
	if i.sourceCounts["activities"] != 0 || i.sourceCounts["activity_orders"] != 1479 {
		return fmt.Errorf("activity exclusion baseline changed: activities=%d orders=%d", i.sourceCounts["activities"], i.sourceCounts["activity_orders"])
	}
	var orphanCount int64
	orphanChecks := []string{
		`SELECT COUNT(*) FROM food_orders o LEFT JOIN users u ON u.id=o.user_id WHERE u.id IS NULL`,
		`SELECT COUNT(*) FROM food_order_items x LEFT JOIN food_orders o ON o.id=x.food_order_id WHERE o.id IS NULL`,
		`SELECT COUNT(*) FROM food_order_items x LEFT JOIN products p ON p.id=x.product_id WHERE p.id IS NULL`,
		`SELECT COUNT(*) FROM recharge r LEFT JOIN users u ON u.id=r.user_id WHERE u.id IS NULL`,
		`SELECT COUNT(*) FROM user_coupon c LEFT JOIN users u ON u.id=c.user_id WHERE u.id IS NULL`,
	}
	for _, query := range orphanChecks {
		if err := i.source.QueryRowContext(ctx, query).Scan(&orphanCount); err != nil {
			return fmt.Errorf("run v1 orphan check: %w", err)
		}
		if orphanCount != 0 {
			return fmt.Errorf("v1 orphan check failed with %d rows", orphanCount)
		}
	}
	var duplicatePhones int64
	if err := i.source.QueryRowContext(ctx, `SELECT COUNT(*) FROM (
		SELECT phone FROM users WHERE phone IS NOT NULL AND phone<>'' GROUP BY phone HAVING COUNT(*)>1
	) d`).Scan(&duplicatePhones); err != nil {
		return err
	}
	i.metrics["duplicatePhoneValues"] = duplicatePhones
	return nil
}

func (i *importer) execute(ctx context.Context) error {
	var migrationTable int
	if err := i.target.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema=DATABASE() AND table_name='legacy_v1_archives'`).Scan(&migrationTable); err != nil {
		return err
	}
	if migrationTable != 1 {
		return errors.New("target migration 00079 is not applied")
	}
	if !i.skipAssets {
		if err := i.migrateAssets(ctx); err != nil {
			return err
		}
	}
	return i.target.WithinTx(ctx, func(tx *sql.Tx) error {
		if err := clearTarget(ctx, tx); err != nil {
			return err
		}
		if err := i.migrateStore(ctx, tx); err != nil {
			return err
		}
		if err := i.migrateMembers(ctx, tx); err != nil {
			return err
		}
		if err := i.migrateCatalog(ctx, tx); err != nil {
			return err
		}
		if err := i.migrateVenue(ctx, tx); err != nil {
			return err
		}
		if err := i.migrateBanner(ctx, tx); err != nil {
			return err
		}
		if err := i.migrateWallet(ctx, tx); err != nil {
			return err
		}
		if err := i.migrateFoodOrders(ctx, tx); err != nil {
			return err
		}
		if err := i.migrateRecharges(ctx, tx); err != nil {
			return err
		}
		if err := i.migratePointWorkflows(ctx, tx); err != nil {
			return err
		}
		if err := i.migrateCoupons(ctx, tx); err != nil {
			return err
		}
		if err := i.archiveOverlappingLedgers(ctx, tx); err != nil {
			return err
		}
		if err := i.finishRun(ctx, tx); err != nil {
			return err
		}
		return i.reconcile(ctx, tx)
	})
}

var clearTables = []string{
	"vip_coupon_daily_usages", "coupon_redemptions", "coupon_entitlements", "verifications", "tickets",
	"activity_orders", "activity_ticket_types", "activity_sessions", "activities", "tournament_events",
	"food_order_cancellations", "food_order_items", "food_orders", "refund_orders", "payment_transactions",
	"offline_collection_orders", "payment_orders", "business_orders", "wallet_holds", "wallet_ledger_entries",
	"wallet_accounts", "point_savings", "point_withdrawals", "sign_in_records", "reservation_daily_claims",
	"arrival_records", "waitlist_entries", "reservations", "invitation_reward_events", "invitation_reward_accounts",
	"benefit_grants", "rule_executions", "member_recharge_history", "legacy_recharge_members",
	"legacy_recharge_growth_totals", "staff_accounts", "members", "print_jobs", "outbox_events",
	"idempotency_keys", "audit_logs", "error_events", "reporting_daily", "franchise_inquiries",
	"banners", "store_item_overrides", "catalog_variants", "catalog_items", "catalog_categories",
	"seats", "tables",
	"legacy_id_maps", "reconciliation_results", "migration_runs", "legacy_v1_archives",
}

func clearTarget(ctx context.Context, tx *sql.Tx) error {
	for _, table := range clearTables {
		if _, err := tx.ExecContext(ctx, "DELETE FROM `"+table+"`"); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE admin_accounts SET store_id=1 WHERE store_id IS NOT NULL AND store_id<>1`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE printer_devices SET store_id=1 WHERE store_id<>1`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE x FROM store_rules x JOIN store_rules k ON k.store_id=1 AND k.rule_key=x.rule_key WHERE x.store_id<>1`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE store_rules SET store_id=1 WHERE store_id<>1`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM store_settings WHERE store_id<>1 AND EXISTS (SELECT 1 FROM (SELECT store_id FROM store_settings) k WHERE k.store_id=1)`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE store_settings SET store_id=1 WHERE store_id<>1`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE coupon_templates SET store_id=1 WHERE store_id IS NOT NULL AND store_id<>1`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE rule_definitions SET store_id=1 WHERE store_id IS NOT NULL AND store_id<>1`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM stores WHERE id<>1`); err != nil {
		return err
	}
	return nil
}

func (i *importer) mapID(ctx context.Context, tx *sql.Tx, sourceTable string, sourceID int64, targetTable string, targetID int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO legacy_id_maps
		(source_system,source_table,source_id,target_table,target_id,created_at)
		VALUES ('v1',?,?,?,?,UTC_TIMESTAMP())`, sourceTable, sourceID, targetTable, targetID)
	return err
}

func nullableTime(value sql.NullTime, fallback time.Time) time.Time {
	if value.Valid {
		return value.Time.UTC()
	}
	return fallback
}

func truncateUTF8(value string, max int) string {
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}

func nullString(value sql.NullString) any {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	return value.String
}

func centsFromString(value string) (int64, error) {
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(value, "-")
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid decimal %q", value)
	}
	whole := int64(0)
	for _, ch := range parts[0] {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid decimal %q", value)
		}
		whole = whole*10 + int64(ch-'0')
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > 2 {
		return 0, fmt.Errorf("too many decimal places: %q", value)
	}
	fraction += strings.Repeat("0", 2-len(fraction))
	frac := int64(0)
	for _, ch := range fraction {
		frac = frac*10 + int64(ch-'0')
	}
	result := whole*100 + frac
	if negative {
		result = -result
	}
	return result, nil
}

func jsonObject(value any) string {
	b, _ := json.Marshal(value)
	return string(b)
}
