package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (i *importer) migratePointWorkflows(ctx context.Context, tx *sql.Tx) error {
	now := time.Now().UTC()
	savingRows := [][]any{}
	rows, err := i.source.QueryContext(ctx, `SELECT id,store_id,user_id,save_points,points,verify_staff,verify_time,COALESCE(remark,''),status,created_at,updated_at FROM save_points ORDER BY id`)
	if err != nil {
		return err
	}
	savings := int64(0)
	for rows.Next() {
		var id, store, member, rawPoints, awarded int64
		var reviewer sql.NullInt64
		var reviewed, created, updated sql.NullTime
		var remark string
		var status int
		if err = rows.Scan(&id, &store, &member, &rawPoints, &awarded, &reviewer, &reviewed, &remark, &status, &created, &updated); err != nil {
			rows.Close()
			return err
		}
		state := map[int]string{1: "pending", 2: "approved", 3: "rejected"}[status]
		savingRows = append(savingRows, []any{id, store, member, rawPoints, rawPoints, awarded, state, truncateUTF8(remark, 255), fmt.Sprintf("v1:save_points:%d", id), nullableInt(reviewer), nullableReviewerType(reviewer), nullableNullTime(reviewed), nullableTime(created, now), nullableTime(updated, now)})
		if err = i.mapID(ctx, tx, "save_points", id, "point_savings", id); err != nil {
			rows.Close()
			return err
		}
		savings++
	}
	rows.Close()
	if err := execBatches(ctx, tx, `INSERT INTO point_savings
		(id,store_id,member_id,points,base_points,awarded_points,status,remark,idem_key,reviewed_by,reviewed_by_type,reviewed_at,created_at,updated_at)`, 14, savingRows); err != nil {
		return err
	}
	rows, err = i.source.QueryContext(ctx, `SELECT id,user_id,store_id,points,audit_status,auditor_id,audited_at,COALESCE(audit_remark,description,''),created_at,updated_at FROM points_withdrawal ORDER BY id`)
	if err != nil {
		return err
	}
	withdrawals := int64(0)
	withdrawalRows := [][]any{}
	for rows.Next() {
		var id, member, store, points int64
		var status, remark string
		var reviewer sql.NullInt64
		var reviewed, created, updated sql.NullTime
		if err = rows.Scan(&id, &member, &store, &points, &status, &reviewer, &reviewed, &remark, &created, &updated); err != nil {
			rows.Close()
			return err
		}
		withdrawalRows = append(withdrawalRows, []any{id, store, member, points, status, truncateUTF8(remark, 255), fmt.Sprintf("v1:points_withdrawal:%d", id), nullableInt(reviewer), nullableNullTime(reviewed), nullableTime(created, now), nullableTime(updated, now)})
		if err = i.mapID(ctx, tx, "points_withdrawal", id, "point_withdrawals", id); err != nil {
			rows.Close()
			return err
		}
		withdrawals++
	}
	rows.Close()
	if err := execBatches(ctx, tx, `INSERT INTO point_withdrawals
		(id,store_id,member_id,points,status,remark,idem_key,reviewed_by,reviewed_at,created_at,updated_at)`, 11, withdrawalRows); err != nil {
		return err
	}
	i.metrics["pointSavingsImported"] = savings
	i.metrics["pointWithdrawalsImported"] = withdrawals
	return nil
}

func (i *importer) migrateCoupons(ctx context.Context, tx *sql.Tx) error {
	categoryIDs := map[string]int64{}
	rows, err := tx.QueryContext(ctx, `SELECT business_type,id FROM coupon_categories WHERE business_type IN ('alcohol','event_ticket')`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var kind string
		var id int64
		if err = rows.Scan(&kind, &id); err != nil {
			rows.Close()
			return err
		}
		categoryIDs[kind] = id
	}
	rows.Close()
	if categoryIDs["alcohol"] == 0 || categoryIDs["event_ticket"] == 0 {
		return fmt.Errorf("required coupon categories are missing")
	}
	type templateInfo struct {
		id   int64
		kind string
	}
	templates := map[string]templateInfo{}
	rows, err = i.source.QueryContext(ctx, `SELECT name,MIN(COALESCE(description,'')),COUNT(*),MIN(created_at),MAX(updated_at) FROM user_coupon GROUP BY name ORDER BY name`)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	templateCount := int64(0)
	for rows.Next() {
		var name, description string
		var issued int64
		var created, updated sql.NullTime
		if err = rows.Scan(&name, &description, &issued, &created, &updated); err != nil {
			rows.Close()
			return err
		}
		kind := legacyCouponType(name)
		res, err := tx.ExecContext(ctx, `INSERT INTO coupon_templates
			(scope_type,store_id,name,description,coupon_type,category_id,admission_count,value_cent,points_price,
			 stock_quantity,issued_quantity,validity_rule,applicable_scope,per_member_limit,status,created_at,updated_at)
			VALUES ('store',1,?,?,?,?,1,0,0,0,?,JSON_OBJECT(),JSON_OBJECT(),0,'published',?,?)`, truncateUTF8(name, 128), truncateUTF8(defaultString(description, "1.0历史券"), 65535), kind, categoryIDs[kind], issued, nullableTime(created, now), nullableTime(updated, now))
		if err != nil {
			rows.Close()
			return err
		}
		id, _ := res.LastInsertId()
		templates[name] = templateInfo{id, kind}
		templateCount++
	}
	rows.Close()
	rows, err = i.source.QueryContext(ctx, `SELECT id,user_id,name,expired_time,status,created_at,updated_at FROM user_coupon ORDER BY id`)
	if err != nil {
		return err
	}
	entitlements := int64(0)
	redemptions := int64(0)
	active := int64(0)
	entitlementRows := [][]any{}
	redemptionRows := [][]any{}
	for rows.Next() {
		var id, member int64
		var name string
		var expires, created, updated sql.NullTime
		var status int
		if err = rows.Scan(&id, &member, &name, &expires, &status, &created, &updated); err != nil {
			rows.Close()
			return err
		}
		info := templates[name]
		state := map[int]string{1: "active", 2: "used", 3: "expired"}[status]
		if state == "" {
			state = "void"
		}
		if state == "active" {
			active++
		}
		createdAt := nullableTime(created, now)
		updatedAt := nullableTime(updated, createdAt)
		entitlementRows = append(entitlementRows, []any{id, fmt.Sprintf("V1C%010d", id), info.id, 1, member, 1, state, 1, "1.0历史迁移", "legacy", nullableNullTime(expires), fmt.Sprintf("v1:user_coupon:%d", id), createdAt, updatedAt})
		if err = i.mapID(ctx, tx, "user_coupon", id, "coupon_entitlements", id); err != nil {
			rows.Close()
			return err
		}
		if state == "used" {
			redemptionRows = append(redemptionRows, []any{id, fmt.Sprintf("V1CR%010d", id), id, info.id, member, 1, jsonObject(map[string]any{"couponType": info.kind, "legacy": true}), "[]", "legacy", updatedAt})
			redemptions++
		}
		entitlements++
	}
	rows.Close()
	if err := execBatches(ctx, tx, `INSERT INTO coupon_entitlements
		(id,entitlement_no,coupon_template_id,admission_count,member_id,store_id,status,rule_version,granted_reason,granted_by_type,expires_at,idem_key,created_at,updated_at)`, 14, entitlementRows); err != nil {
		return err
	}
	if err := execBatches(ctx, tx, `INSERT INTO coupon_redemptions
		(id,redemption_no,entitlement_id,coupon_template_id,member_id,store_id,matched_rule_json,item_snapshot_json,verified_by_type,created_at)`, 10, redemptionRows); err != nil {
		return err
	}
	i.metrics["legacyCouponTemplatesCreated"] = templateCount
	i.metrics["couponEntitlementsImported"] = entitlements
	i.metrics["activeCouponEntitlementsImported"] = active
	i.metrics["couponRedemptionsImported"] = redemptions
	return nil
}

func (i *importer) archiveOverlappingLedgers(ctx context.Context, tx *sql.Tx) error {
	total := int64(0)
	archiveRows := [][]any{}
	queries := []struct{ table, query string }{
		{"balance_consumption_records", `SELECT id,user_id,amount_consumed,balance_before,balance_after,consumption_type,related_type,related_id,description,extra_data,created_at FROM balance_consumption_records ORDER BY id`},
		{"user_points", `SELECT id,user_id,points,type,source,source_id,description,created_at FROM user_points ORDER BY id`},
	}
	for _, item := range queries {
		rows, err := i.source.QueryContext(ctx, item.query)
		if err != nil {
			return err
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return err
		}
		for rows.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for idx := range values {
				pointers[idx] = &values[idx]
			}
			if err = rows.Scan(pointers...); err != nil {
				rows.Close()
				return err
			}
			payload := map[string]any{}
			for idx, column := range columns {
				payload[column] = jsonSafe(values[idx])
			}
			sourceID, ok := toInt64(values[0])
			if !ok {
				rows.Close()
				return fmt.Errorf("invalid archive id for %s", item.table)
			}
			var sourceCreated any
			for idx, column := range columns {
				if column == "created_at" {
					sourceCreated = values[idx]
				}
			}
			archiveRows = append(archiveRows, []any{item.table, sourceID, jsonObject(payload), sourceCreated, time.Now().UTC()})
			total++
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}
	if err := execBatches(ctx, tx, `INSERT INTO legacy_v1_archives
		(source_table,source_id,payload_json,source_created_at,imported_at)`, 5, archiveRows); err != nil {
		return err
	}
	i.metrics["legacyRowsArchived"] = total
	return nil
}

func (i *importer) finishRun(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO migration_runs(run_key,status,started_at,finished_at,summary_json,created_at) VALUES (?,'completed',UTC_TIMESTAMP(),UTC_TIMESTAMP(),?,UTC_TIMESTAMP())`, i.runKey, jsonObject(i.metrics))
	return err
}

func (i *importer) reconcile(ctx context.Context, tx *sql.Tx) error {
	checks := map[string]struct {
		table    string
		expected int64
	}{
		"members": {"members", i.sourceCounts["users"]}, "catalog_items": {"catalog_items", i.sourceCounts["products"]}, "food_orders": {"food_orders", i.sourceCounts["food_orders"]}, "food_order_items": {"food_order_items", i.sourceCounts["food_order_items"]}, "point_savings": {"point_savings", i.sourceCounts["save_points"]}, "point_withdrawals": {"point_withdrawals", i.sourceCounts["points_withdrawal"]}, "coupon_entitlements": {"coupon_entitlements", i.sourceCounts["user_coupon"]},
	}
	for name, check := range checks {
		var actual int64
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM `"+check.table+"`").Scan(&actual); err != nil {
			return err
		}
		i.targetCounts[check.table] = actual
		status := "ok"
		if actual != check.expected {
			status = "diff"
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO reconciliation_results(run_key,check_name,status,expected_json,actual_json,note,created_at) VALUES (?,?,?,?,?,'',UTC_TIMESTAMP())`, i.runKey, name, status, jsonObject(map[string]int64{"count": check.expected}), jsonObject(map[string]int64{"count": actual}))
		if err != nil {
			return err
		}
		if status != "ok" {
			return fmt.Errorf("reconciliation %s: expected %d got %d", name, check.expected, actual)
		}
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO reconciliation_results(run_key,check_name,status,expected_json,actual_json,note,created_at) VALUES (?,'activity_orders_excluded','diff',?,?,?,UTC_TIMESTAMP())`, i.runKey, jsonObject(map[string]int64{"sourceCount": i.sourceCounts["activity_orders"]}), jsonObject(map[string]int64{"targetCount": 0}), "approved by owner: v1 activity definitions are absent")
	return err
}

func nullableReviewerType(value sql.NullInt64) any {
	if value.Valid {
		return "staff"
	}
	return nil
}
func legacyCouponType(name string) string {
	if strings.Contains(name, "周赛") || strings.Contains(name, "月赛") || strings.Contains(name, "新客专享") {
		return "event_ticket"
	}
	return "alcohol"
}
func jsonSafe(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano)
	default:
		return v
	}
}
func toInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case uint64:
		if v > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(v), true
	case int32:
		return int64(v), true
	case uint32:
		return int64(v), true
	case []byte:
		var n int64
		_, err := fmt.Sscan(string(v), &n)
		return n, err == nil
	default:
		return 0, false
	}
}
