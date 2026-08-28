package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (i *importer) migrateStore(ctx context.Context, tx *sql.Tx) error {
	var id int64
	var name, address, phone, status string
	var latitude, longitude sql.NullString
	var created, updated sql.NullTime
	err := i.source.QueryRowContext(ctx, `SELECT id,name,address,phone,status,latitude,longitude,created_at,updated_at FROM stores LIMIT 1`).Scan(
		&id, &name, &address, &phone, &status, &latitude, &longitude, &created, &updated)
	if err != nil {
		return fmt.Errorf("read v1 store: %w", err)
	}
	if id != 1 {
		return fmt.Errorf("v1 store id must be 1, got %d", id)
	}
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO stores
		(id,legacy_store_id,name,phone,address,latitude,longitude,business_hours,status,created_at,updated_at)
		VALUES (1,1,?,?,?,?,?,'',?,?,?)
		ON DUPLICATE KEY UPDATE legacy_store_id=1,name=VALUES(name),phone=VALUES(phone),address=VALUES(address),
		latitude=VALUES(latitude),longitude=VALUES(longitude),status=VALUES(status),updated_at=VALUES(updated_at)`,
		name, nullString(sql.NullString{String: phone, Valid: phone != ""}), address, nullString(latitude), nullString(longitude),
		status, nullableTime(created, now), nullableTime(updated, now))
	if err != nil {
		return fmt.Errorf("upsert v1 store: %w", err)
	}
	if err = i.mapID(ctx, tx, "stores", 1, "stores", 1); err != nil {
		return err
	}
	i.metrics["storesImported"] = 1
	return nil
}

func (i *importer) migrateMembers(ctx context.Context, tx *sql.Tx) error {
	tiers := map[int]int64{}
	rows, err := tx.QueryContext(ctx, `SELECT id,level FROM membership_tiers`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var level int
		if err = rows.Scan(&id, &level); err != nil {
			rows.Close()
			return err
		}
		tiers[level] = id
	}
	rows.Close()
	phoneCounts := map[string]int{}
	phoneRows, err := i.source.QueryContext(ctx, `SELECT phone,COUNT(*) FROM users WHERE phone IS NOT NULL AND phone<>'' GROUP BY phone`)
	if err != nil {
		return err
	}
	for phoneRows.Next() {
		var phone string
		var count int
		if err = phoneRows.Scan(&phone, &count); err != nil {
			phoneRows.Close()
			return err
		}
		phoneCounts[phone] = count
	}
	phoneRows.Close()
	rows, err = i.source.QueryContext(ctx, `SELECT id,openid,COALESCE(nickname,''),COALESCE(avatar,''),phone,user_type,store_id,
		invite_code,invited_by,sex,level,created_at,updated_at FROM users ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now().UTC()
	count := int64(0)
	staff := int64(0)
	duplicatePhones := int64(0)
	memberRows := [][]any{}
	staffRows := [][]any{}
	for rows.Next() {
		var id int64
		var openid, nickname, avatar, userType, inviteCode string
		var phone sql.NullString
		var storeID, invitedBy sql.NullInt64
		var sex, level sql.NullInt64
		var created, updated sql.NullTime
		if err = rows.Scan(&id, &openid, &nickname, &avatar, &phone, &userType, &storeID, &inviteCode, &invitedBy, &sex, &level, &created, &updated); err != nil {
			return err
		}
		var phoneValue any
		if phone.Valid && phone.String != "" && phoneCounts[phone.String] == 1 {
			phoneValue = phone.String
		} else if phone.Valid && phone.String != "" {
			duplicatePhones++
		}
		gender := any(nil)
		if sex.Valid {
			if sex.Int64 == 1 {
				gender = "male"
			} else if sex.Int64 == 2 {
				gender = "female"
			}
		}
		var tier any
		if level.Valid && tiers[int(level.Int64)] > 0 {
			tier = tiers[int(level.Int64)]
		}
		createdAt := nullableTime(created, now)
		updatedAt := nullableTime(updated, createdAt)
		memberRows = append(memberRows, []any{id, id, truncateUTF8(openid, 64), truncateUTF8(nickname, 64), nullString(sql.NullString{String: avatar, Valid: avatar != ""}), gender, phoneValue, inviteCode, nullableInt(invitedBy), phoneValue != nil && nickname != "", "active", tier, int64(0), createdAt, updatedAt})
		if err = i.mapID(ctx, tx, "users", id, "members", id); err != nil {
			return err
		}
		if userType == "staff" {
			store := int64(1)
			if storeID.Valid && storeID.Int64 > 0 {
				store = storeID.Int64
			}
			staffRows = append(staffRows, []any{id, nil, id, truncateUTF8(openid, 64), truncateUTF8(nickname, 64), store, "active", int64(0), createdAt, updatedAt})
			staff++
		}
		count++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if err = execBatches(ctx, tx, `INSERT INTO members
		(id,legacy_user_id,wechat_openid,nickname,avatar_url,gender,phone,invite_code,invited_by_member_id,profile_completed,status,current_tier_id,token_version,created_at,updated_at)`, 15, memberRows); err != nil {
		return err
	}
	if err = execBatches(ctx, tx, `INSERT INTO staff_accounts
		(id,legacy_staff_id,member_id,wechat_openid,name,store_id,status,token_version,created_at,updated_at)`, 10, staffRows); err != nil {
		return err
	}
	i.metrics["membersImported"] = count
	i.metrics["staffAccountsImported"] = staff
	i.metrics["phonesClearedForDuplicateValue"] = duplicatePhones
	return nil
}

func (i *importer) migrateCatalog(ctx context.Context, tx *sql.Tx) error {
	couponCategories := map[int64]bool{}
	rows, err := i.source.QueryContext(ctx, `SELECT DISTINCT category_id FROM products WHERE type=2`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		couponCategories[id] = true
	}
	rows.Close()
	rows, err = i.source.QueryContext(ctx, `SELECT id,parent_id,name,sort_order,status,created_at,updated_at FROM categories ORDER BY id`)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	categories := int64(0)
	for rows.Next() {
		var id, sort int64
		var parent sql.NullInt64
		var name, status string
		var created, updated sql.NullTime
		if err = rows.Scan(&id, &parent, &name, &sort, &status, &created, &updated); err != nil {
			rows.Close()
			return err
		}
		categoryType := "product"
		if couponCategories[id] {
			categoryType = "coupon"
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO catalog_categories
			(id,scope_type,store_id,parent_id,name,category_type,asset_id,sort_order,status,created_at,updated_at)
			VALUES (?,'store',1,?,?,?,?,?,?,?,?)`, id, nullableInt(parent), truncateUTF8(name, 128), categoryType, nullableAsset(i.assetIDs[assetMapKey("category", id)]), sort, mapStatus(status), nullableTime(created, now), nullableTime(updated, now))
		if err != nil {
			rows.Close()
			return fmt.Errorf("insert category %d: %w", id, err)
		}
		if err = i.mapID(ctx, tx, "categories", id, "catalog_categories", id); err != nil {
			rows.Close()
			return err
		}
		categories++
	}
	rows.Close()
	rows, err = i.source.QueryContext(ctx, `SELECT id,category_id,name,COALESCE(description,''),price,stock,type,COALESCE(points,0),COALESCE(sort_order,0),status,created_at,updated_at FROM products ORDER BY id`)
	if err != nil {
		return err
	}
	items := int64(0)
	for rows.Next() {
		var id, categoryID, stock, typeID, points, sort int64
		var name, description, price, status string
		var created, updated sql.NullTime
		if err = rows.Scan(&id, &categoryID, &name, &description, &price, &stock, &typeID, &points, &sort, &status, &created, &updated); err != nil {
			rows.Close()
			return err
		}
		priceCent, err := centsFromString(price)
		if err != nil {
			rows.Close()
			return err
		}
		itemType := "food"
		if typeID == 2 {
			itemType = "coupon"
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO catalog_items
			(id,scope_type,store_id,source_item_id,category_id,name,description,asset_id,item_type,price_cent,stock_quantity,
			 pay_channels,points_reward,sort_order,status,created_at,updated_at)
			VALUES (?,'store',1,?,?,?,?,?,?,?,?,JSON_ARRAY('wechat','coin'),?,?,?,?,?)`, id, id, categoryID, truncateUTF8(name, 128), description, nullableAsset(i.assetIDs[assetMapKey("product", id)]), itemType, priceCent, stock, points, sort, mapStatus(status), nullableTime(created, now), nullableTime(updated, now))
		if err != nil {
			rows.Close()
			return fmt.Errorf("insert product %d: %w", id, err)
		}
		if err = i.mapID(ctx, tx, "products", id, "catalog_items", id); err != nil {
			rows.Close()
			return err
		}
		items++
	}
	rows.Close()
	i.metrics["catalogCategoriesImported"] = categories
	i.metrics["catalogItemsImported"] = items
	return nil
}

func (i *importer) migrateVenue(ctx context.Context, tx *sql.Tx) error {
	rows, err := i.source.QueryContext(ctx, `SELECT id,table_name,table_number,basic_coin,seat_count,status,created_at,updated_at FROM tables ORDER BY id`)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	tables := int64(0)
	for rows.Next() {
		var id int64
		var name, code, base string
		var capacity, status int
		var created, updated sql.NullTime
		if err = rows.Scan(&id, &name, &code, &base, &capacity, &status, &created, &updated); err != nil {
			rows.Close()
			return err
		}
		basePoints := parseLeadingInt(base)
		state := "available"
		if status != 1 {
			state = "disabled"
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO tables(id,store_id,name,code,capacity,base_points,status,created_at,updated_at) VALUES (?,1,?,?,?,?,?,?,?)`, id, truncateUTF8(name, 64), truncateUTF8(code, 64), capacity, basePoints, state, nullableTime(created, now), nullableTime(updated, now))
		if err != nil {
			rows.Close()
			return err
		}
		if err = i.mapID(ctx, tx, "tables", id, "tables", id); err != nil {
			rows.Close()
			return err
		}
		tables++
	}
	rows.Close()
	rows, err = i.source.QueryContext(ctx, `SELECT id,table_id,seat_number,status,created_at,updated_at FROM seats ORDER BY id`)
	if err != nil {
		return err
	}
	seats := int64(0)
	for rows.Next() {
		var id, tableID int64
		var number string
		var status int
		var created, updated sql.NullTime
		if err = rows.Scan(&id, &tableID, &number, &status, &created, &updated); err != nil {
			rows.Close()
			return err
		}
		state := "available"
		if status != 1 {
			state = "disabled"
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO seats(id,store_id,table_id,name,status,created_at,updated_at) VALUES (?,1,?,?,?,?,?)`, id, tableID, truncateUTF8(number, 64), state, nullableTime(created, now), nullableTime(updated, now))
		if err != nil {
			rows.Close()
			return err
		}
		if err = i.mapID(ctx, tx, "seats", id, "seats", id); err != nil {
			rows.Close()
			return err
		}
		seats++
	}
	rows.Close()
	i.metrics["tablesImported"] = tables
	i.metrics["seatsImported"] = seats
	return nil
}

func (i *importer) migrateBanner(ctx context.Context, tx *sql.Tx) error {
	if i.skipAssets {
		i.metrics["bannersSkippedForRehearsal"] = i.sourceCounts["banner"]
		return nil
	}
	rows, err := i.source.QueryContext(ctx, `SELECT id,sort,status,created_at,updated_at FROM banner ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now().UTC()
	count := int64(0)
	for rows.Next() {
		var id, sort, status int64
		var created, updated sql.NullTime
		if err = rows.Scan(&id, &sort, &status, &created, &updated); err != nil {
			return err
		}
		assetID := i.assetIDs[assetMapKey("banner", id)]
		if assetID == 0 {
			return fmt.Errorf("banner %d asset is missing", id)
		}
		state := "active"
		if status != 1 {
			state = "disabled"
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO banners(id,scope_type,store_id,title,asset_id,link_url,sort_order,status,created_at,updated_at)
			VALUES (?,'store',1,'',?,'',?,?,?,?)`, id, assetID, sort, state, nullableTime(created, now), nullableTime(updated, now))
		if err != nil {
			return err
		}
		if err = i.mapID(ctx, tx, "banner", id, "banners", id); err != nil {
			return err
		}
		count++
	}
	i.metrics["bannersImported"] = count
	return rows.Err()
}

func nullableInt(value sql.NullInt64) any {
	if value.Valid {
		return value.Int64
	}
	return nil
}
func nullableAsset(id int64) any {
	if id > 0 {
		return id
	}
	return nil
}
func mapStatus(value string) string {
	if value == "active" {
		return "active"
	}
	if value == "sold_out" {
		return "sold_out"
	}
	return "disabled"
}
func parseLeadingInt(value string) int64 {
	part := strings.Split(value, "/")[0]
	n, _ := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
	return n
}
