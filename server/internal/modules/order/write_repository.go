package order

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	mrand "math/rand"
	"time"

	"github.com/inwardclub/server/internal/modules/printer"
	"github.com/inwardclub/server/internal/modules/wallet"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/idempotency"
)

// Coin wallet asset type (mirrors wallet.AssetCoins). The coin balance is held
// in the same integer-cent unit as payment_orders.amount_cent, so a coin payment
// debits amount_cent one-for-one.
const assetCoins = "coins"

// ---- Create input structs ----

// FoodOrderCreate is the fully-resolved input the repository persists. Line
// prices are resolved from the catalog inside the transaction, so only the
// requested item/variant/quantity are carried here — never a client price.
type FoodOrderCreate struct {
	MemberID        int64
	StoreID         int64
	TableID         *int64
	Remark          string
	PayMethod       string
	Lines           []FoodLineItem
	BusinessOrderNo string
	PaymentOrderNo  string
	IdemKey         string
	Now             time.Time
}

// RechargeOrderCreate is the input for a wallet top-up order. Recharge has no
// dedicated table; it is a business_orders row of type "recharge".
type RechargeOrderCreate struct {
	MemberID        int64
	AmountCent      int64
	PayMethod       string
	BusinessOrderNo string
	PaymentOrderNo  string
	IdemKey         string
	Now             time.Time
}

// ActivityOrderCreate is the input for a ticket purchase. The unit price and the
// owning activity are resolved from the ticket type inside the transaction.
type ActivityOrderCreate struct {
	ActivityID      int64
	TicketTypeID    int64
	Quantity        int
	MemberID        int64
	PayMethod       string
	BusinessOrderNo string
	PaymentOrderNo  string
	IdemKey         string
	Now             time.Time
}

// CoinPayment settles a pending coin payment order from the member's wallet.
type CoinPayment struct {
	MemberID       int64
	PaymentOrderID int64
	IdemKey        string
	Now            time.Time
}

// ---- Food ----

func (r *sqlRepository) CreateFoodOrder(ctx context.Context, in FoodOrderCreate) (FoodOrder, []FoodOrderItem, PaymentOrder, error) {
	var (
		food  FoodOrder
		items []FoodOrderItem
		po    PaymentOrder
	)
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if err := claimIdem(ctx, tx, "mini/food-orders", in.IdemKey, OrderTypeFood, 0); err != nil {
			return err
		}
		if err := validateFoodOrderScope(ctx, tx, in.StoreID, in.TableID); err != nil {
			return err
		}
		// Resolve each line's price/name from the catalog; never trust the client.
		var total int64
		items = items[:0]
		for _, ln := range in.Lines {
			name, unit, points, payChannels, err := resolveAndReserveItem(
				ctx, tx, in.StoreID, in.PayMethod, ln, in.Now,
			)
			if err != nil {
				return err
			}
			subtotal := unit * int64(ln.Quantity)
			total += subtotal
			items = append(items, FoodOrderItem{
				ItemID:        ln.ItemID,
				VariantID:     ln.VariantID,
				NameSnapshot:  name,
				UnitPriceCent: unit,
				Quantity:      ln.Quantity,
				PayChannels:   payChannels,
				PointsReward:  points,
				SubtotalCent:  subtotal,
			})
		}
		businessID, err := insertBusinessOrder(ctx, tx, in.BusinessOrderNo, OrderTypeFood, &in.StoreID, in.MemberID, total, in.Now)
		if err != nil {
			return err
		}
		const insFood = `INSERT INTO food_orders
			(business_order_id, store_id, member_id, table_id, total_amount_cent, fulfillment_status, remark, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?)`
		res, err := tx.ExecContext(ctx, insFood, businessID, in.StoreID, in.MemberID, in.TableID, total, in.Remark, in.Now, in.Now)
		if err != nil {
			return mapWriteErr(err)
		}
		foodID, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		const insItem = `INSERT INTO food_order_items
			(food_order_id, item_id, variant_id, name_snapshot, unit_price_cent, quantity,
			 pay_channels_snapshot, points_reward_snapshot, subtotal_cent, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		for i := range items {
			res, err := tx.ExecContext(ctx, insItem, foodID, items[i].ItemID, items[i].VariantID,
				items[i].NameSnapshot, items[i].UnitPriceCent, items[i].Quantity,
				items[i].PayChannels, items[i].PointsReward, items[i].SubtotalCent, in.Now)
			if err != nil {
				return mapWriteErr(err)
			}
			id, err := res.LastInsertId()
			if err != nil {
				return apperr.Internal(err)
			}
			items[i].ID = id
			items[i].FoodOrderID = foodID
		}
		paymentID, err := insertPaymentOrder(ctx, tx, in.PaymentOrderNo, businessID, &in.StoreID, in.MemberID, total, in.PayMethod, in.Now)
		if err != nil {
			return err
		}
		food = FoodOrder{
			ID:                foodID,
			BusinessOrderID:   businessID,
			StoreID:           in.StoreID,
			MemberID:          in.MemberID,
			TableID:           in.TableID,
			TotalAmountCent:   total,
			FulfillmentStatus: "pending",
			Remark:            in.Remark,
			CreatedAt:         in.Now,
		}
		po = PaymentOrder{ID: paymentID, PaymentOrderNo: in.PaymentOrderNo, BusinessOrderID: businessID,
			MemberID: &in.MemberID, AmountCent: total, PayMethod: in.PayMethod, Status: PaymentStatusPending, CreatedAt: in.Now}
		return nil
	})
	if err != nil {
		return FoodOrder{}, nil, PaymentOrder{}, err
	}
	return food, items, po, nil
}

// ---- Recharge ----

func (r *sqlRepository) CreateRechargeOrder(ctx context.Context, in RechargeOrderCreate) (RechargeOrder, PaymentOrder, error) {
	var (
		ro RechargeOrder
		po PaymentOrder
	)
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if err := claimIdem(ctx, tx, "mini/recharge-orders", in.IdemKey, OrderTypeRecharge, 0); err != nil {
			return err
		}
		businessID, err := insertBusinessOrder(ctx, tx, in.BusinessOrderNo, OrderTypeRecharge, nil, in.MemberID, in.AmountCent, in.Now)
		if err != nil {
			return err
		}
		paymentID, err := insertPaymentOrder(ctx, tx, in.PaymentOrderNo, businessID, nil, in.MemberID, in.AmountCent, in.PayMethod, in.Now)
		if err != nil {
			return err
		}
		ro = RechargeOrder{
			ID:              businessID,
			BusinessOrderNo: in.BusinessOrderNo,
			MemberID:        in.MemberID,
			TotalAmountCent: in.AmountCent,
			OrderStatus:     "created",
			PaymentStatus:   "unpaid",
			CreatedAt:       in.Now,
		}
		po = PaymentOrder{ID: paymentID, PaymentOrderNo: in.PaymentOrderNo, BusinessOrderID: businessID,
			MemberID: &in.MemberID, AmountCent: in.AmountCent, PayMethod: in.PayMethod, Status: PaymentStatusPending, CreatedAt: in.Now}
		return nil
	})
	if err != nil {
		return RechargeOrder{}, PaymentOrder{}, err
	}
	return ro, po, nil
}

// ---- Activity ----

func (r *sqlRepository) CreateActivityOrder(ctx context.Context, in ActivityOrderCreate) (ActivityOrder, []Ticket, PaymentOrder, error) {
	var (
		ao      ActivityOrder
		tickets []Ticket
		po      PaymentOrder
	)
	err := r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if err := claimIdem(ctx, tx, "mini/activity-orders", in.IdemKey, OrderTypeActivity, 0); err != nil {
			return err
		}
		// Resolve the ticket type: price, session, owning activity and store.
		var (
			price      int64
			sessionID  sql.NullInt64
			storeID    sql.NullInt64
			ttActivity int64
		)
		const selTT = `SELECT price_cent, session_id, store_id, activity_id
			FROM activity_ticket_types WHERE id = ?`
		err := tx.QueryRowContext(ctx, selTT, in.TicketTypeID).Scan(&price, &sessionID, &storeID, &ttActivity)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("ticket type not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if ttActivity != in.ActivityID {
			return apperr.Invalid("ticket type does not belong to the activity")
		}
		// Reserve stock: unlimited when stock_quantity is 0, otherwise guarded.
		const reserve = `UPDATE activity_ticket_types
			SET sold_quantity = sold_quantity + ?, updated_at = ?
			WHERE id = ? AND (stock_quantity = 0 OR sold_quantity + ? <= stock_quantity)`
		res, err := tx.ExecContext(ctx, reserve, in.Quantity, in.Now, in.TicketTypeID, in.Quantity)
		if err != nil {
			return mapWriteErr(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return apperr.Conflict("insufficient ticket stock")
		}

		total := price * int64(in.Quantity)
		var storePtr *int64
		if storeID.Valid {
			storePtr = &storeID.Int64
		}
		businessID, err := insertBusinessOrder(ctx, tx, in.BusinessOrderNo, OrderTypeActivity, storePtr, in.MemberID, total, in.Now)
		if err != nil {
			return err
		}
		const insAO = `INSERT INTO activity_orders
			(business_order_id, activity_id, store_id, member_id, ticket_count, total_amount_cent, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 'created', ?, ?)`
		res, err = tx.ExecContext(ctx, insAO, businessID, in.ActivityID, storePtr, in.MemberID, in.Quantity, total, in.Now, in.Now)
		if err != nil {
			return mapWriteErr(err)
		}
		activityOrderID, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		var sessPtr *int64
		if sessionID.Valid {
			sessPtr = &sessionID.Int64
		}
		const insTicket = `INSERT INTO tickets
			(ticket_no, verification_code, activity_order_id, activity_id, ticket_type_id, session_id, store_id, member_id, price_cent, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)`
		tickets = tickets[:0]
		for i := 0; i < in.Quantity; i++ {
			ticketNo := newNo("TK", in.Now)
			code := newCode()
			res, err := tx.ExecContext(ctx, insTicket, ticketNo, code, activityOrderID, in.ActivityID,
				in.TicketTypeID, sessPtr, storePtr, in.MemberID, price, in.Now, in.Now)
			if err != nil {
				return mapWriteErr(err)
			}
			id, err := res.LastInsertId()
			if err != nil {
				return apperr.Internal(err)
			}
			tickets = append(tickets, Ticket{ID: id, TicketNo: ticketNo, ActivityID: in.ActivityID,
				TicketTypeID: in.TicketTypeID, SessionID: sessPtr, PriceCent: price, Status: "pending"})
		}
		paymentID, err := insertPaymentOrder(ctx, tx, in.PaymentOrderNo, businessID, storePtr, in.MemberID, total, in.PayMethod, in.Now)
		if err != nil {
			return err
		}
		ao = ActivityOrder{
			ID:              activityOrderID,
			BusinessOrderID: businessID,
			ActivityID:      in.ActivityID,
			StoreID:         storePtr,
			MemberID:        in.MemberID,
			TicketCount:     in.Quantity,
			TotalAmountCent: total,
			Status:          "created",
			CreatedAt:       in.Now,
		}
		po = PaymentOrder{ID: paymentID, PaymentOrderNo: in.PaymentOrderNo, BusinessOrderID: businessID,
			MemberID: &in.MemberID, AmountCent: total, PayMethod: in.PayMethod, Status: PaymentStatusPending, CreatedAt: in.Now}
		return nil
	})
	if err != nil {
		return ActivityOrder{}, nil, PaymentOrder{}, err
	}
	return ao, tickets, po, nil
}

// ---- Pay by coin ----

func (r *sqlRepository) SettleByCoin(ctx context.Context, in CoinPayment) error {
	return r.db.WithinTx(ctx, func(tx *sql.Tx) error {
		if err := claimIdem(ctx, tx, "mini/pay-by-coin", in.IdemKey, "payment_order", in.PaymentOrderID); err != nil {
			return err
		}
		// Lock the payment order and re-validate under the lock.
		var (
			businessID int64
			memberID   sql.NullInt64
			amount     int64
			payMethod  string
			status     string
			poNo       string
			storeID    sql.NullInt64
			orderType  string
			businessNo string
		)
		const selPO = `SELECT po.payment_order_no, po.business_order_id, po.member_id, po.amount_cent,
				po.pay_method, po.status, bo.store_id, bo.order_type, bo.business_order_no
			FROM payment_orders po JOIN business_orders bo ON bo.id = po.business_order_id
			WHERE po.id = ? FOR UPDATE`
		err := tx.QueryRowContext(ctx, selPO, in.PaymentOrderID).
			Scan(&poNo, &businessID, &memberID, &amount, &payMethod, &status, &storeID, &orderType, &businessNo)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.NotFound("payment order not found")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if !memberID.Valid || memberID.Int64 != in.MemberID {
			return apperr.NotFound("payment order not found")
		}
		if status != PaymentStatusPending {
			return apperr.Conflict("payment order is not payable")
		}
		if payMethod != PayMethodCoin {
			return apperr.Invalid("payment order pay method mismatch")
		}

		// Lock the coin wallet and debit it.
		var (
			accountID int64
			available int64
		)
		const selAcct = `SELECT id, available_amount FROM wallet_accounts
			WHERE member_id = ? AND asset_type = ? FOR UPDATE`
		err = tx.QueryRowContext(ctx, selAcct, in.MemberID, assetCoins).Scan(&accountID, &available)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.New(apperr.CodeInsufficientBalance, "insufficient coin balance")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if available < amount {
			return apperr.New(apperr.CodeInsufficientBalance, "insufficient coin balance")
		}
		newBalance := available - amount
		const debit = `UPDATE wallet_accounts SET available_amount = ?, version = version + 1, updated_at = ?
			WHERE id = ? AND available_amount = ?`
		res, err := tx.ExecContext(ctx, debit, newBalance, in.Now, accountID, available)
		if err != nil {
			return apperr.Internal(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			// Concurrent debit changed the balance under us; deadlock-retry-safe.
			return apperr.Conflict("wallet balance changed, retry")
		}
		const insLedger = `INSERT INTO wallet_ledger_entries
			(account_id, member_id, asset_type, direction, amount, balance_after, reason, source_type, source_id, idem_key, created_at)
			VALUES (?, ?, ?, 'debit', ?, ?, 'order_payment', 'payment_order', ?, ?, ?)`
		if _, err := tx.ExecContext(ctx, insLedger, accountID, in.MemberID, assetCoins, amount, newBalance,
			in.PaymentOrderID, in.IdemKey, in.Now); err != nil {
			return mapWriteErr(err)
		}
		const insTxn = `INSERT INTO payment_transactions
			(payment_order_id, provider, channel, out_trade_no, amount_cent, status, created_at)
			VALUES (?, 'coin', '', ?, ?, 'success', ?)`
		if _, err := tx.ExecContext(ctx, insTxn, in.PaymentOrderID, poNo, amount, in.Now); err != nil {
			return mapWriteErr(err)
		}
		const payPO = `UPDATE payment_orders SET status = 'paid', paid_at = ?, updated_at = ? WHERE id = ?`
		if _, err := tx.ExecContext(ctx, payPO, in.Now, in.Now, in.PaymentOrderID); err != nil {
			return apperr.Internal(err)
		}
		const payBO = `UPDATE business_orders SET payment_status = 'paid', updated_at = ? WHERE id = ?`
		if _, err := tx.ExecContext(ctx, payBO, in.Now, businessID); err != nil {
			return apperr.Internal(err)
		}
		if orderType == OrderTypeFood {
			if _, err := tx.ExecContext(ctx,
				`UPDATE food_orders SET fulfillment_status = 'preparing', updated_at = ?
				 WHERE business_order_id = ?`,
				in.Now, businessID,
			); err != nil {
				return apperr.Internal(err)
			}
			if _, err := tx.ExecContext(ctx,
				`UPDATE business_orders SET order_status = 'preparing', updated_at = ? WHERE id = ?`,
				in.Now, businessID,
			); err != nil {
				return apperr.Internal(err)
			}
		}
		// An activity order's tickets become usable (pending -> active) exactly when it
		// is paid; nothing else in the ticket lifecycle activates them, so both
		// settlement paths (coin here, WeChat in payment.SettleWeChat) run this.
		// Idempotent: a re-settled order finds no pending rows.
		if orderType == OrderTypeActivity {
			const actTickets = `UPDATE tickets t JOIN activity_orders ao ON ao.id = t.activity_order_id
				SET t.status = 'active', t.updated_at = ?
				WHERE ao.business_order_id = ? AND t.status = 'pending'`
			if _, err := tx.ExecContext(ctx, actTickets, in.Now, businessID); err != nil {
				return apperr.Internal(err)
			}
		}
		if orderType == OrderTypeFood {
			if _, err := wallet.GrantFoodOrderPoints(
				ctx, tx, in.PaymentOrderID, businessID, in.MemberID, in.Now,
			); err != nil {
				return err
			}
		}
		// A store-bound coin settlement (food / store activity) prints a receipt on
		// the store's printer; a store-less order (recharge) prints nothing. Rides
		// this settlement transaction, so it never fires on a rollback.
		if storeID.Valid {
			if err := printer.WriteReceipt(ctx, tx, printer.Receipt{
				StoreID:         storeID.Int64,
				PaymentOrderID:  in.PaymentOrderID,
				BusinessOrderID: businessID,
				BusinessOrderNo: businessNo,
				OrderType:       orderType,
				AmountCent:      amount,
				PaidAt:          in.Now,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// ---- shared helpers ----

// validateFoodOrderScope confirms the store exists and an optional table belongs
// to it. The table check is server-authoritative; a client cannot attach another
// store's table to the order.
func validateFoodOrderScope(ctx context.Context, tx *sql.Tx, storeID int64, tableID *int64) error {
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM stores WHERE id = ?`, storeID).Scan(&exists); err != nil {
		return apperr.Internal(err)
	}
	if exists == 0 {
		return apperr.NotFound("store not found")
	}
	if tableID == nil {
		return nil
	}
	var tableStoreID int64
	err := tx.QueryRowContext(ctx, `SELECT store_id FROM venue_tables WHERE id = ?`, *tableID).Scan(&tableStoreID)
	if errors.Is(err, sql.ErrNoRows) {
		return apperr.NotFound("table not found")
	}
	if err != nil {
		return apperr.Internal(err)
	}
	if tableStoreID != storeID {
		return apperr.Invalid("table does not belong to the store")
	}
	return nil
}

// resolveAndReserveItem locks the requested item (and optional variant),
// validates store/status/payment-channel scope, snapshots its reward, and
// atomically reserves stock in the same transaction as order creation.
func resolveAndReserveItem(
	ctx context.Context,
	tx *sql.Tx,
	storeID int64,
	payMethod string,
	line FoodLineItem,
	now time.Time,
) (name string, unit, points int64, payChannelsSnapshot string, err error) {
	var (
		itemStoreID int64
		itemStatus  string
		payChannels []byte
		stock       int64
	)
	if line.VariantID != nil {
		var variantStatus string
		const q = `SELECT cv.name, cv.price_cent, ci.points_reward, ci.store_id,
				ci.status, ci.pay_channels, cv.stock_quantity, cv.status
			FROM catalog_variants cv
			JOIN catalog_items ci ON ci.id = cv.item_id
			WHERE cv.id = ? AND cv.item_id = ? FOR UPDATE`
		err = tx.QueryRowContext(ctx, q, *line.VariantID, line.ItemID).Scan(
			&name, &unit, &points, &itemStoreID, &itemStatus,
			&payChannels, &stock, &variantStatus,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, 0, "", apperr.NotFound("catalog variant not found")
		}
		if err != nil {
			return "", 0, 0, "", apperr.Internal(err)
		}
		if variantStatus != "active" && variantStatus != "published" {
			return "", 0, 0, "", apperr.Conflict("catalog variant is not available")
		}
	} else {
		const q = `SELECT name, price_cent, points_reward, store_id, status,
				pay_channels, stock_quantity
			FROM catalog_items WHERE id = ? FOR UPDATE`
		err = tx.QueryRowContext(ctx, q, line.ItemID).Scan(
			&name, &unit, &points, &itemStoreID, &itemStatus, &payChannels, &stock,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, 0, "", apperr.NotFound("catalog item not found")
		}
		if err != nil {
			return "", 0, 0, "", apperr.Internal(err)
		}
	}
	if itemStoreID != storeID {
		return "", 0, 0, "", apperr.Invalid("catalog item does not belong to the store")
	}
	if itemStatus != "published" {
		return "", 0, 0, "", apperr.Conflict("catalog item is not available")
	}
	var channels []string
	if err := json.Unmarshal(payChannels, &channels); err != nil {
		return "", 0, 0, "", apperr.Internal(err)
	}
	allowed := false
	for _, channel := range channels {
		if channel == payMethod || (channel == "balance" && payMethod == PayMethodCoin) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", 0, 0, "", apperr.Invalid("catalog item does not support the selected pay method")
	}
	if stock < int64(line.Quantity) {
		return "", 0, 0, "", apperr.Conflict("insufficient catalog stock")
	}
	if line.VariantID != nil {
		const reserve = `UPDATE catalog_variants SET stock_quantity = stock_quantity - ?, updated_at = ?
			WHERE id = ? AND stock_quantity >= ?`
		res, err := tx.ExecContext(ctx, reserve, line.Quantity, now, *line.VariantID, line.Quantity)
		if err != nil {
			return "", 0, 0, "", apperr.Internal(err)
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return "", 0, 0, "", apperr.Conflict("insufficient catalog stock")
		}
	} else {
		const reserve = `UPDATE catalog_items SET stock_quantity = stock_quantity - ?, updated_at = ?
			WHERE id = ? AND stock_quantity >= ?`
		res, err := tx.ExecContext(ctx, reserve, line.Quantity, now, line.ItemID, line.Quantity)
		if err != nil {
			return "", 0, 0, "", apperr.Internal(err)
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return "", 0, 0, "", apperr.Conflict("insufficient catalog stock")
		}
	}
	if points < 0 {
		points = 0
	}
	return name, unit, points, string(payChannels), nil
}

func insertBusinessOrder(ctx context.Context, tx *sql.Tx, no, orderType string, storeID *int64, memberID, total int64, now time.Time) (int64, error) {
	const q = `INSERT INTO business_orders
		(business_order_no, order_type, store_id, member_id, total_amount_cent, order_status, payment_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'created', 'unpaid', ?, ?)`
	res, err := tx.ExecContext(ctx, q, no, orderType, storeID, memberID, total, now, now)
	if err != nil {
		return 0, mapWriteErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return id, nil
}

func insertPaymentOrder(ctx context.Context, tx *sql.Tx, no string, businessID int64, storeID *int64, memberID, amount int64, payMethod string, now time.Time) (int64, error) {
	const q = `INSERT INTO payment_orders
		(payment_order_no, business_order_id, store_id, member_id, amount_cent, pay_method, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?)`
	res, err := tx.ExecContext(ctx, q, no, businessID, storeID, memberID, amount, payMethod, now, now)
	if err != nil {
		return 0, mapWriteErr(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return id, nil
}

// claimIdem claims the idempotency key when present. A missing key is allowed by
// this helper; the router enforces the Idempotency-Key header for these routes.
func claimIdem(ctx context.Context, tx *sql.Tx, scope, key, targetType string, targetID int64) error {
	if key == "" {
		return nil
	}
	return idempotency.Claim(ctx, tx, scope, key, targetType, targetID)
}

func mapWriteErr(err error) error {
	if platdb.IsDuplicate(err) {
		return apperr.Conflict("duplicate order record")
	}
	return apperr.Internal(err)
}

// newNo builds a human-readable, collision-resistant business number.
func newNo(prefix string, now time.Time) string {
	return fmt.Sprintf("%s%s%04d", prefix, now.Format("20060102150405"), mrand.Intn(10000))
}

// newCode mints a random verification code for a ticket instance.
func newCode() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
