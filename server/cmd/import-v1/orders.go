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
	businessRows := [][]any{}
	paymentRows := [][]any{}
	foodRows := [][]any{}
	transactionRows := [][]any{}
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
		businessID := id
		businessRows = append(businessRows, []any{businessID, orderNo, "food", store, member, amountCent, businessStatus, businessPayment, jsonObject(map[string]any{"legacyTable": "food_orders", "legacyId": id}), createdAt, updatedAt})
		paymentNo := "V1P-" + orderNo
		paymentID := id
		paymentRows = append(paymentRows, []any{paymentID, paymentNo, businessID, store, member, amountCent, payMethod, paymentOrderStatus(paymentStatus), fmt.Sprintf("v1:food_payment:%d", id), createdAt, updatedAt, nullableNullTime(paidAt)})
		payments++
		foodRows = append(foodRows, []any{id, businessID, store, member, nullableInt(tableID), amountCent, points, orderStatus, "", createdAt, updatedAt})
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
			transactionRows = append(transactionRows, []any{id, paymentID, provider, provider, paymentNo, ext, amountCent, "success", jsonObject(map[string]any{"legacyTable": "food_orders", "legacyId": id}), nullableTime(paidAt, updatedAt)})
			transactions++
		}
		orders++
	}
	if err := execBatches(ctx, tx, `INSERT INTO business_orders
		(id,business_order_no,order_type,store_id,member_id,total_amount_cent,order_status,payment_status,snapshot_json,created_at,updated_at)`, 11, businessRows); err != nil {
		return err
	}
	if err := execBatches(ctx, tx, `INSERT INTO payment_orders
		(id,payment_order_no,business_order_id,store_id,member_id,amount_cent,pay_method,status,idem_key,created_at,updated_at,paid_at)`, 12, paymentRows); err != nil {
		return err
	}
	if err := execBatches(ctx, tx, `INSERT INTO food_orders
		(id,business_order_id,store_id,member_id,table_id,total_amount_cent,points_earned,fulfillment_status,remark,created_at,updated_at)`, 11, foodRows); err != nil {
		return err
	}
	if err := execBatches(ctx, tx, `INSERT INTO payment_transactions
		(id,payment_order_id,provider,channel,out_trade_no,external_transaction_no,amount_cent,status,raw_payload,created_at)`, 10, transactionRows); err != nil {
		return err
	}
	itemRows, err := i.source.QueryContext(ctx, `SELECT id,food_order_id,product_id,product_name,product_price,quantity,subtotal,created_at FROM food_order_items ORDER BY id`)
	if err != nil {
		return err
	}
	defer itemRows.Close()
	items := int64(0)
	itemBatch := [][]any{}
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
		itemBatch = append(itemBatch, []any{id, orderID, itemID, truncateUTF8(name, 128), priceCent, quantity, "[\"wechat\",\"coin\"]", int64(0), subtotalCent, nullableTime(created, now)})
		if err = i.mapID(ctx, tx, "food_order_items", id, "food_order_items", id); err != nil {
			return err
		}
		items++
	}
	if err := execBatches(ctx, tx, `INSERT INTO food_order_items
		(id,food_order_id,item_id,name_snapshot,unit_price_cent,quantity,pay_channels_snapshot,points_reward_snapshot,subtotal_cent,created_at)`, 10, itemBatch); err != nil {
		return err
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
	businessRows := [][]any{}
	paymentRows := [][]any{}
	transactionRows := [][]any{}
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
		businessID := int64(1_000_000) + id
		businessRows = append(businessRows, []any{businessID, orderNo, "recharge", 1, member, amountCent, boStatus, boPayment, jsonObject(map[string]any{"legacyTable": "recharge", "legacyId": id, "coins": coins, "points": points}), createdAt, updatedAt})
		paymentNo := "V1R-" + orderNo
		paymentID := businessID
		paymentRows = append(paymentRows, []any{paymentID, paymentNo, businessID, 1, member, amountCent, "wechat", poStatus, fmt.Sprintf("v1:recharge_payment:%d", id), createdAt, updatedAt, nullableNullTime(paidAt)})
		if isPaid {
			transactionRows = append(transactionRows, []any{paymentID, paymentID, "wechat", "wechat", paymentNo, nullString(external), amountCent, "success", jsonObject(map[string]any{"legacyTable": "recharge", "legacyId": id}), nullableTime(paidAt, updatedAt)})
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
	if err := execBatches(ctx, tx, `INSERT INTO business_orders
		(id,business_order_no,order_type,store_id,member_id,total_amount_cent,order_status,payment_status,snapshot_json,created_at,updated_at)`, 11, businessRows); err != nil {
		return err
	}
	if err := execBatches(ctx, tx, `INSERT INTO payment_orders
		(id,payment_order_no,business_order_id,store_id,member_id,amount_cent,pay_method,status,idem_key,created_at,updated_at,paid_at)`, 12, paymentRows); err != nil {
		return err
	}
	if err := execBatches(ctx, tx, `INSERT INTO payment_transactions
		(id,payment_order_id,provider,channel,out_trade_no,external_transaction_no,amount_cent,status,raw_payload,created_at)`, 10, transactionRows); err != nil {
		return err
	}
	legacyMemberRows := [][]any{}
	historyRows := [][]any{}
	for member, value := range first {
		legacyMemberRows = append(legacyMemberRows, []any{member, value.at, sourceSnapshot})
		historyRows = append(historyRows, []any{member, value.businessID, value.at, "legacy", time.Now().UTC()})
	}
	if err := execBatches(ctx, tx, `INSERT INTO legacy_recharge_members(legacy_user_id,first_paid_at,source_snapshot)`, 3, legacyMemberRows); err != nil {
		return err
	}
	if err := execBatches(ctx, tx, `INSERT INTO member_recharge_history(member_id,first_business_order_id,first_paid_at,origin,created_at)`, 5, historyRows); err != nil {
		return err
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
