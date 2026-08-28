package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (i *importer) migrateFoodOrders(ctx context.Context, tx *sql.Tx) error {
	rows, err := i.source.QueryContext(ctx, `SELECT id,order_no,user_id,store_id,table_id,total_amount,payment_method,payment_status,order_status,
		points_earned,transaction_id,paid_at,created_at,updated_at FROM food_orders ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now().UTC()
	orders := int64(0)
	payments := int64(0)
	transactions := int64(0)
	for rows.Next() {
		var id, member, store, points int64
		var orderNo, amount, payMethod, paymentStatus, orderStatus string
		var tableID sql.NullInt64
		var external sql.NullString
		var paidAt, created, updated sql.NullTime
		if err = rows.Scan(&id, &orderNo, &member, &store, &tableID, &amount, &payMethod, &paymentStatus, &orderStatus, &points, &external, &paidAt, &created, &updated); err != nil {
			return err
		}
		amountCent, err := centsFromString(amount)
		if err != nil {
			return err
		}
		createdAt := nullableTime(created, now)
		updatedAt := nullableTime(updated, createdAt)
		businessPayment := businessPaymentStatus(paymentStatus)
		businessStatus := orderStatus
		if businessStatus == "pending" && paymentStatus == "paid" {
			businessStatus = "paid"
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO business_orders
			(business_order_no,order_type,store_id,member_id,total_amount_cent,order_status,payment_status,snapshot_json,created_at,updated_at)
			VALUES (?,'food',?,?,?,?,?,?,?,?)`, orderNo, store, member, amountCent, businessStatus, businessPayment, jsonObject(map[string]any{"legacyTable": "food_orders", "legacyId": id}), createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("insert food business order %d: %w", id, err)
		}
		businessID, _ := res.LastInsertId()
		paymentNo := "V1P-" + orderNo
		res, err = tx.ExecContext(ctx, `INSERT INTO payment_orders
			(payment_order_no,business_order_id,store_id,member_id,amount_cent,pay_method,status,idem_key,created_at,updated_at,paid_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)`, paymentNo, businessID, store, member, amountCent, payMethod, paymentOrderStatus(paymentStatus), fmt.Sprintf("v1:food_payment:%d", id), createdAt, updatedAt, nullableNullTime(paidAt))
		if err != nil {
			return fmt.Errorf("insert food payment %d: %w", id, err)
		}
		paymentID, _ := res.LastInsertId()
		payments++
		_, err = tx.ExecContext(ctx, `INSERT INTO food_orders
			(id,business_order_id,store_id,member_id,table_id,total_amount_cent,points_earned,fulfillment_status,remark,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?, '',?,?)`, id, businessID, store, member, nullableInt(tableID), amountCent, points, orderStatus, createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("insert food order %d: %w", id, err)
		}
		if err = i.mapID(ctx, tx, "food_orders", id, "food_orders", id); err != nil {
			return err
		}
		if paymentStatus == "paid" {
			provider := payMethod
			if provider != "coin" {
				provider = "wechat"
			}
			var ext any
			if provider == "wechat" {
				ext = nullString(external)
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO payment_transactions
				(payment_order_id,provider,channel,out_trade_no,external_transaction_no,amount_cent,status,raw_payload,created_at)
				VALUES (?,?,?,?,?,?,'success',?,?)`, paymentID, provider, provider, paymentNo, ext, amountCent, jsonObject(map[string]any{"legacyTable": "food_orders", "legacyId": id}), nullableTime(paidAt, updatedAt))
			if err != nil {
				return fmt.Errorf("insert food transaction %d: %w", id, err)
			}
			transactions++
		}
		orders++
	}
	itemRows, err := i.source.QueryContext(ctx, `SELECT id,food_order_id,product_id,product_name,product_price,quantity,subtotal,created_at FROM food_order_items ORDER BY id`)
	if err != nil {
		return err
	}
	defer itemRows.Close()
	items := int64(0)
	for itemRows.Next() {
		var id, orderID, itemID int64
		var name, price, subtotal string
		var quantity int
		var created sql.NullTime
		if err = itemRows.Scan(&id, &orderID, &itemID, &name, &price, &quantity, &subtotal, &created); err != nil {
			return err
		}
		priceCent, err := centsFromString(price)
		if err != nil {
			return err
		}
		subtotalCent, err := centsFromString(subtotal)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO food_order_items
			(id,food_order_id,item_id,name_snapshot,unit_price_cent,quantity,pay_channels_snapshot,points_reward_snapshot,subtotal_cent,created_at)
			VALUES (?,?,?,?,?,?,JSON_ARRAY('wechat','coin'),0,?,?)`, id, orderID, itemID, truncateUTF8(name, 128), priceCent, quantity, subtotalCent, nullableTime(created, now))
		if err != nil {
			return fmt.Errorf("insert food item %d: %w", id, err)
		}
		if err = i.mapID(ctx, tx, "food_order_items", id, "food_order_items", id); err != nil {
			return err
		}
		items++
	}
	i.metrics["foodOrdersImported"] = orders
	i.metrics["foodOrderItemsImported"] = items
	i.metrics["foodPaymentOrdersImported"] = payments
	i.metrics["foodPaymentTransactionsImported"] = transactions
	return nil
}

func (i *importer) migrateRecharges(ctx context.Context, tx *sql.Tx) error {
	rows, err := i.source.QueryContext(ctx, `SELECT id,user_id,total_fee,coin,COALESCE(points,0),order_no,transaction_no,payment_status,status,paid_at,created_at,updated_at FROM recharge ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now().UTC()
	orders := int64(0)
	paid := int64(0)
	first := map[int64]struct {
		businessID int64
		at         time.Time
	}{}
	for rows.Next() {
		var id, member, coins, points int64
		var amount, orderNo string
		var external sql.NullString
		var paymentStatus, status int
		var paidAt, created, updated sql.NullTime
		if err = rows.Scan(&id, &member, &amount, &coins, &points, &orderNo, &external, &paymentStatus, &status, &paidAt, &created, &updated); err != nil {
			return err
		}
		amountCent, err := centsFromString(amount)
		if err != nil {
			return err
		}
		createdAt := nullableTime(created, now)
		updatedAt := nullableTime(updated, createdAt)
		isPaid := paymentStatus == 2 && status == 2
		boStatus := "created"
		boPayment := "unpaid"
		poStatus := "pending"
		if isPaid {
			boStatus = "completed"
			boPayment = "paid"
			poStatus = "paid"
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO business_orders
			(business_order_no,order_type,store_id,member_id,total_amount_cent,order_status,payment_status,snapshot_json,created_at,updated_at)
			VALUES (?,'recharge',1,?,?,?,?,?,?,?)`, orderNo, member, amountCent, boStatus, boPayment, jsonObject(map[string]any{"legacyTable": "recharge", "legacyId": id, "coins": coins, "points": points}), createdAt, updatedAt)
		if err != nil {
			return fmt.Errorf("insert recharge %d: %w", id, err)
		}
		businessID, _ := res.LastInsertId()
		paymentNo := "V1R-" + orderNo
		res, err = tx.ExecContext(ctx, `INSERT INTO payment_orders
			(payment_order_no,business_order_id,store_id,member_id,amount_cent,pay_method,status,idem_key,created_at,updated_at,paid_at)
			VALUES (?,?,1,?,?,'wechat',?,?,?,?,?)`, paymentNo, businessID, member, amountCent, poStatus, fmt.Sprintf("v1:recharge_payment:%d", id), createdAt, updatedAt, nullableNullTime(paidAt))
		if err != nil {
			return err
		}
		paymentID, _ := res.LastInsertId()
		if isPaid {
			_, err = tx.ExecContext(ctx, `INSERT INTO payment_transactions
			(payment_order_id,provider,channel,out_trade_no,external_transaction_no,amount_cent,status,raw_payload,created_at)
			VALUES (?,'wechat','wechat',?,?,?,'success',?,?)`, paymentID, paymentNo, nullString(external), amountCent, jsonObject(map[string]any{"legacyTable": "recharge", "legacyId": id}), nullableTime(paidAt, updatedAt))
			if err != nil {
				return err
			}
			paidAtValue := nullableTime(paidAt, updatedAt)
			if current, ok := first[member]; !ok || paidAtValue.Before(current.at) {
				first[member] = struct {
					businessID int64
					at         time.Time
				}{businessID, paidAtValue}
			}
			paid++
		}
		if err = i.mapID(ctx, tx, "recharge", id, "business_orders", businessID); err != nil {
			return err
		}
		orders++
	}
	for member, value := range first {
		_, err = tx.ExecContext(ctx, `INSERT INTO legacy_recharge_members(legacy_user_id,first_paid_at,source_snapshot) VALUES (?,?,?)`, member, value.at, sourceSnapshot)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO member_recharge_history(member_id,first_business_order_id,first_paid_at,origin,created_at) VALUES (?,?,?,'legacy',UTC_TIMESTAMP())`, member, value.businessID, value.at)
		if err != nil {
			return err
		}
	}
	i.metrics["rechargesImported"] = orders
	i.metrics["paidRechargesImported"] = paid
	i.metrics["legacyRechargeMembersImported"] = int64(len(first))
	return nil
}

func businessPaymentStatus(value string) string {
	switch value {
	case "paid":
		return "paid"
	case "refunded":
		return "refunded"
	case "failed":
		return "failed"
	default:
		return "unpaid"
	}
}
func paymentOrderStatus(value string) string {
	switch value {
	case "paid":
		return "paid"
	case "failed":
		return "failed"
	case "refunded":
		return "refunded"
	default:
		return "pending"
	}
}
func nullableNullTime(value sql.NullTime) any {
	if value.Valid {
		return value.Time.UTC()
	}
	return nil
}
