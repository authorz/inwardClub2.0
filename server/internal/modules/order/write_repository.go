package order

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	mrand "math/rand"
	"time"

	"github.com/inwardclub/server/internal/modules/coupon"
	"github.com/inwardclub/server/internal/modules/printer"
	"github.com/inwardclub/server/internal/modules/vipbenefit"
	"github.com/inwardclub/server/internal/modules/wallet"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/idempotency"
)

// Coin wallet asset type (mirrors wallet.AssetCoins). Wallet and ledger amounts
// are integer coin counts; fiat payment amounts remain integer cents.
const assetCoins = "coins"

func coinAmountForPayment(amountCent int64) (int64, error) {
	coins, err := wallet.CoinsRequired(amountCent)
	if err != nil {
		return 0, apperr.Invalid("金币支付仅支持整元金额")
	}
	return coins, nil
}

// ---- Create input structs ----

// FoodOrderCreate is the fully-resolved input the repository persists. Line
// prices are resolved from the catalog inside the transaction, so only the
// requested item/variant/quantity are carried here — never a client price.
type FoodOrderCreate struct {
	MemberID            int64
	StoreID             int64
	TableID             *int64
	Remark              string
	PayMethod           string
	CouponEntitlementID *int64
	Lines               []FoodLineItem
	BusinessOrderNo     string
	PaymentOrderNo      string
	IdemKey             string
	Now                 time.Time
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
	ActivityID          int64
	TicketTypeID        int64
	Quantity            int
	MemberID            int64
	PayMethod           string
	CouponEntitlementID *int64
	BusinessOrderNo     string
	PaymentOrderNo      string
	IdemKey             string
	Now                 time.Time
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
		var redeemCouponTemplateID int64
		if in.PayMethod == PayMethodCoupon {
			if in.CouponEntitlementID == nil {
				return apperr.Invalid("请选择优惠券")
			}
			var (
				couponStatus string
				couponMember int64
				couponStore  sql.NullInt64
				couponExpiry sql.NullTime
				couponType   string
			)
			const selectCoupon = `SELECT e.status, e.member_id, e.store_id, e.expires_at,
				e.coupon_template_id, ct.coupon_type
				FROM coupon_entitlements e JOIN coupon_templates ct ON ct.id = e.coupon_template_id
				WHERE e.id = ? FOR UPDATE`
			err := tx.QueryRowContext(ctx, selectCoupon, *in.CouponEntitlementID).Scan(
				&couponStatus, &couponMember, &couponStore, &couponExpiry,
				&redeemCouponTemplateID, &couponType,
			)
			if errors.Is(err, sql.ErrNoRows) {
				return apperr.NotFound("优惠券不存在")
			}
			if err != nil {
				return apperr.Internal(err)
			}
			if couponMember != in.MemberID {
				return apperr.NotFound("优惠券不存在")
			}
			if couponStatus != coupon.StatusActive || !isProductCouponType(couponType) ||
				(couponExpiry.Valid && !couponExpiry.Time.After(in.Now)) {
				return apperr.Conflict("优惠券不可用或已过期")
			}
			if couponStore.Valid && couponStore.Int64 != in.StoreID {
				return apperr.Invalid("优惠券不适用于当前门店")
			}
		}
		// Resolve each line's price/name from the catalog; never trust the client.
		var total int64
		items = items[:0]
		for _, ln := range in.Lines {
			name, unit, points, couponTemplateID, payChannels, err := resolveAndReserveItem(
				ctx, tx, in.StoreID, in.PayMethod, ln, in.Now,
			)
			if err != nil {
				return err
			}
			if in.PayMethod == PayMethodCoupon {
				var eligible int
				const couponEligible = `SELECT COUNT(*) FROM catalog_items
					WHERE id = ? AND JSON_CONTAINS(
						COALESCE(coupon_template_ids, JSON_ARRAY()), CAST(? AS JSON), '$'
					)`
				if err := tx.QueryRowContext(ctx, couponEligible, ln.ItemID, redeemCouponTemplateID).Scan(&eligible); err != nil {
					return apperr.Internal(err)
				}
				if eligible == 0 {
					return apperr.Invalid("所选商品不可使用当前优惠券兑换")
				}
			}
			subtotal := unit * int64(ln.Quantity)
			total += subtotal
			items = append(items, FoodOrderItem{
				ItemID:           ln.ItemID,
				VariantID:        ln.VariantID,
				NameSnapshot:     name,
				UnitPriceCent:    unit,
				Quantity:         ln.Quantity,
				PayChannels:      payChannels,
				PointsReward:     points,
				CouponTemplateID: couponTemplateID,
				SubtotalCent:     subtotal,
			})
		}
		if in.PayMethod == PayMethodCoin {
			if _, err := coinAmountForPayment(total); err != nil {
				return err
			}
		}
		chargedTotal := total
		if in.PayMethod == PayMethodCoupon {
			chargedTotal = 0
		}
		businessID, err := insertBusinessOrder(ctx, tx, in.BusinessOrderNo, OrderTypeFood, &in.StoreID, in.MemberID, total, in.Now)
		if err != nil {
			return err
		}
		const insFood = `INSERT INTO food_orders
			(business_order_id, store_id, member_id, coupon_entitlement_id, table_id,
			 total_amount_cent, fulfillment_status, remark, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?)`
		res, err := tx.ExecContext(ctx, insFood, businessID, in.StoreID, in.MemberID,
			in.CouponEntitlementID, in.TableID, total, in.Remark, in.Now, in.Now)
		if err != nil {
			return mapWriteErr(err)
		}
		foodID, err := res.LastInsertId()
		if err != nil {
			return apperr.Internal(err)
		}
		const insItem = `INSERT INTO food_order_items
			(food_order_id, item_id, variant_id, name_snapshot, unit_price_cent, quantity,
			 pay_channels_snapshot, points_reward_snapshot, coupon_template_id_snapshot, subtotal_cent, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		for i := range items {
			res, err := tx.ExecContext(ctx, insItem, foodID, items[i].ItemID, items[i].VariantID,
				items[i].NameSnapshot, items[i].UnitPriceCent, items[i].Quantity,
				items[i].PayChannels, items[i].PointsReward, items[i].CouponTemplateID, items[i].SubtotalCent, in.Now)
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
		paymentID, err := insertPaymentOrder(ctx, tx, in.PaymentOrderNo, businessID, &in.StoreID, in.MemberID, chargedTotal, in.PayMethod, in.Now)
		if err != nil {
			return err
		}
		paymentStatus := PaymentStatusPending
		paymentStatusText := "unpaid"
		fulfillmentStatus := "pending"
		if in.PayMethod == PayMethodCoupon {
			if err := coupon.ClaimVIPDailyUsage(ctx, tx, in.MemberID, *in.CouponEntitlementID, in.Now); err != nil {
				return err
			}
			const consumeCoupon = `UPDATE coupon_entitlements SET status = 'used', updated_at = ?
				WHERE id = ? AND member_id = ? AND status = 'active'`
			res, err := tx.ExecContext(ctx, consumeCoupon, in.Now, *in.CouponEntitlementID, in.MemberID)
			if err != nil {
				return apperr.Internal(err)
			}
			if affected, _ := res.RowsAffected(); affected != 1 {
				return apperr.Conflict("优惠券已被使用")
			}
			const insTxn = `INSERT INTO payment_transactions
				(payment_order_id, provider, channel, out_trade_no, amount_cent, status, created_at)
				VALUES (?, 'coupon', '', ?, 0, 'success', ?)`
			if _, err := tx.ExecContext(ctx, insTxn, paymentID, in.PaymentOrderNo, in.Now); err != nil {
				return mapWriteErr(err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE payment_orders
				SET status = 'paid', paid_at = ?, updated_at = ? WHERE id = ?`, in.Now, in.Now, paymentID); err != nil {
				return apperr.Internal(err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE business_orders
				SET payment_status = 'paid', order_status = 'completed', updated_at = ? WHERE id = ?`, in.Now, businessID); err != nil {
				return apperr.Internal(err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE food_orders
				SET fulfillment_status = 'completed', updated_at = ? WHERE id = ?`, in.Now, foodID); err != nil {
				return apperr.Internal(err)
			}
			if err := printer.WriteReceipt(ctx, tx, printer.Receipt{
				StoreID: in.StoreID, PaymentOrderID: paymentID, BusinessOrderID: businessID,
				BusinessOrderNo: in.BusinessOrderNo, OrderType: OrderTypeFood,
				AmountCent: chargedTotal, PaidAt: in.Now,
			}); err != nil {
				return err
			}
			paymentStatus = PaymentStatusPaid
			paymentStatusText = "paid"
			fulfillmentStatus = "completed"
		}
		food = FoodOrder{
			ID:                foodID,
			BusinessOrderID:   businessID,
			BusinessOrderNo:   in.BusinessOrderNo,
			StoreID:           in.StoreID,
			MemberID:          in.MemberID,
			TableID:           in.TableID,
			TotalAmountCent:   total,
			PaymentStatus:     paymentStatusText,
			PayMethod:         in.PayMethod,
			PaidAmountCent:    chargedTotal,
			FulfillmentStatus: fulfillmentStatus,
			Remark:            in.Remark,
			CreatedAt:         in.Now,
		}
		po = PaymentOrder{ID: paymentID, PaymentOrderNo: in.PaymentOrderNo, BusinessOrderID: businessID,
			MemberID: &in.MemberID, AmountCent: chargedTotal, PayMethod: in.PayMethod, Status: paymentStatus, CreatedAt: in.Now}
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
			price               int64
			sessionID           sql.NullInt64
			storeID             sql.NullInt64
			ttActivity          int64
			ticketPayChannels   []byte
			activityPayChannels []byte
		)
		const selTT = `SELECT tt.price_cent, tt.session_id, tt.store_id, tt.activity_id,
			       tt.pay_channels, a.pay_channels
			FROM activity_ticket_types tt JOIN activities a ON a.id = tt.activity_id
			WHERE tt.id = ? AND tt.status = 'active'
			  AND (tt.sale_start_at IS NULL OR tt.sale_start_at <= ?)
			  AND (tt.sale_end_at IS NULL OR tt.sale_end_at >= ?)`
		err := tx.QueryRowContext(ctx, selTT, in.TicketTypeID, in.Now, in.Now).Scan(
			&price, &sessionID, &storeID, &ttActivity, &ticketPayChannels, &activityPayChannels,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.Conflict("该票档当前不在售卖时间内")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if ttActivity != in.ActivityID {
			return apperr.Invalid("ticket type does not belong to the activity")
		}
		allowed, err := activityPayMethodAllowed(ticketPayChannels, activityPayChannels, in.PayMethod)
		if err != nil {
			return apperr.Internal(err)
		}
		if !allowed {
			return apperr.Conflict("当前活动或票档未开启该支付方式")
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
		if in.PayMethod == PayMethodCoin {
			if _, err := coinAmountForPayment(total); err != nil {
				return err
			}
		}
		var storePtr *int64
		if storeID.Valid {
			storePtr = &storeID.Int64
		}
		chargedTotal := total
		if in.PayMethod == PayMethodCoupon {
			chargedTotal = 0
			if in.CouponEntitlementID == nil {
				return apperr.Invalid("请选择赛事门票券")
			}
			var (
				couponStatus string
				couponMember int64
				couponStore  sql.NullInt64
				couponExpiry sql.NullTime
				couponType   string
			)
			const selectCoupon = `SELECT e.status, e.member_id, e.store_id, e.expires_at, ct.coupon_type
				FROM coupon_entitlements e JOIN coupon_templates ct ON ct.id = e.coupon_template_id
				WHERE e.id = ? FOR UPDATE`
			err := tx.QueryRowContext(ctx, selectCoupon, *in.CouponEntitlementID).Scan(
				&couponStatus, &couponMember, &couponStore, &couponExpiry, &couponType,
			)
			if errors.Is(err, sql.ErrNoRows) {
				return apperr.NotFound("赛事门票券不存在")
			}
			if err != nil {
				return apperr.Internal(err)
			}
			if couponMember != in.MemberID {
				return apperr.NotFound("赛事门票券不存在")
			}
			if couponStatus != "active" || couponType != "event_ticket" ||
				(couponExpiry.Valid && !couponExpiry.Time.After(in.Now)) {
				return apperr.Conflict("赛事门票券不可用或已过期")
			}
			if couponStore.Valid && (!storeID.Valid || couponStore.Int64 != storeID.Int64) {
				return apperr.Invalid("赛事门票券不适用于当前门店")
			}
		}
		businessID, err := insertBusinessOrder(ctx, tx, in.BusinessOrderNo, OrderTypeActivity, storePtr, in.MemberID, chargedTotal, in.Now)
		if err != nil {
			return err
		}
		const insAO = `INSERT INTO activity_orders
			(business_order_id, activity_id, store_id, member_id, coupon_entitlement_id,
			 ticket_count, total_amount_cent, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'created', ?, ?)`
		res, err = tx.ExecContext(ctx, insAO, businessID, in.ActivityID, storePtr, in.MemberID,
			in.CouponEntitlementID, in.Quantity, chargedTotal, in.Now, in.Now)
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
			var ticketNo string
			for attempt := 0; attempt < 10; attempt++ {
				ticketNo = newNo("TK", in.Now)
				code, codeErr := newCode()
				if codeErr != nil {
					return apperr.Internal(codeErr)
				}
				res, err = tx.ExecContext(ctx, insTicket, ticketNo, code, activityOrderID, in.ActivityID,
					in.TicketTypeID, sessPtr, storePtr, in.MemberID, price, in.Now, in.Now)
				if err == nil {
					break
				}
				if !platdb.IsDuplicate(err) || attempt == 9 {
					return mapWriteErr(err)
				}
			}
			id, err := res.LastInsertId()
			if err != nil {
				return apperr.Internal(err)
			}
			tickets = append(tickets, Ticket{ID: id, TicketNo: ticketNo, ActivityID: in.ActivityID,
				TicketTypeID: in.TicketTypeID, SessionID: sessPtr, PriceCent: price, Status: "pending"})
		}
		paymentID, err := insertPaymentOrder(ctx, tx, in.PaymentOrderNo, businessID, storePtr, in.MemberID, chargedTotal, in.PayMethod, in.Now)
		if err != nil {
			return err
		}
		paymentStatus := PaymentStatusPending
		orderStatus := "created"
		if in.PayMethod == PayMethodCoupon {
			if err := coupon.ClaimVIPDailyUsage(ctx, tx, in.MemberID, *in.CouponEntitlementID, in.Now); err != nil {
				return err
			}
			const consumeCoupon = `UPDATE coupon_entitlements SET status = 'used', updated_at = ?
				WHERE id = ? AND member_id = ? AND status = 'active'`
			res, err := tx.ExecContext(ctx, consumeCoupon, in.Now, *in.CouponEntitlementID, in.MemberID)
			if err != nil {
				return apperr.Internal(err)
			}
			if affected, _ := res.RowsAffected(); affected != 1 {
				return apperr.Conflict("赛事门票券已被使用")
			}
			const insTxn = `INSERT INTO payment_transactions
				(payment_order_id, provider, channel, out_trade_no, amount_cent, status, created_at)
				VALUES (?, 'coupon', '', ?, 0, 'success', ?)`
			if _, err := tx.ExecContext(ctx, insTxn, paymentID, in.PaymentOrderNo, in.Now); err != nil {
				return mapWriteErr(err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE payment_orders
				SET status = 'paid', paid_at = ?, updated_at = ? WHERE id = ?`, in.Now, in.Now, paymentID); err != nil {
				return apperr.Internal(err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE business_orders
				SET payment_status = 'paid', order_status = 'paid', updated_at = ? WHERE id = ?`, in.Now, businessID); err != nil {
				return apperr.Internal(err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE activity_orders
				SET status = 'paid', updated_at = ? WHERE id = ?`, in.Now, activityOrderID); err != nil {
				return apperr.Internal(err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE tickets
				SET status = 'active', updated_at = ? WHERE activity_order_id = ?`, in.Now, activityOrderID); err != nil {
				return apperr.Internal(err)
			}
			for i := range tickets {
				tickets[i].Status = TicketStatusActive
			}
			if storePtr != nil {
				if err := printer.WriteReceipt(ctx, tx, printer.Receipt{
					StoreID: *storePtr, PaymentOrderID: paymentID, BusinessOrderID: businessID,
					BusinessOrderNo: in.BusinessOrderNo, OrderType: OrderTypeActivity,
					AmountCent: chargedTotal, PaidAt: in.Now,
				}); err != nil {
					return err
				}
			}
			paymentStatus = PaymentStatusPaid
			orderStatus = "paid"
		}
		ao = ActivityOrder{
			ID:              activityOrderID,
			BusinessOrderID: businessID,
			ActivityID:      in.ActivityID,
			StoreID:         storePtr,
			MemberID:        in.MemberID,
			TicketCount:     in.Quantity,
			TotalAmountCent: chargedTotal,
			Status:          orderStatus,
			PayMethod:       in.PayMethod,
			CreatedAt:       in.Now,
		}
		po = PaymentOrder{ID: paymentID, PaymentOrderNo: in.PaymentOrderNo, BusinessOrderID: businessID,
			MemberID: &in.MemberID, AmountCent: chargedTotal, PayMethod: in.PayMethod, Status: paymentStatus, CreatedAt: in.Now}
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
		coinAmount, err := coinAmountForPayment(amount)
		if err != nil {
			return err
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
		if available < coinAmount {
			return apperr.New(apperr.CodeInsufficientBalance, "insufficient coin balance")
		}
		newBalance := available - coinAmount
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
		if _, err := tx.ExecContext(ctx, insLedger, accountID, in.MemberID, assetCoins, coinAmount, newBalance,
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
				`UPDATE food_orders SET fulfillment_status = 'completed', updated_at = ?
				 WHERE business_order_id = ?`,
				in.Now, businessID,
			); err != nil {
				return apperr.Internal(err)
			}
			if _, err := tx.ExecContext(ctx,
				`UPDATE business_orders SET order_status = 'completed', updated_at = ? WHERE id = ?`,
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
			if _, err := coupon.GrantPurchasedCoupons(
				ctx, tx, in.PaymentOrderID, businessID, in.MemberID, in.Now,
			); err != nil {
				return err
			}
			if storeID.Valid {
				if _, err := wallet.GrantTimedLowSpendReward(
					ctx, tx, businessID, in.MemberID, storeID.Int64, in.Now,
				); err != nil {
					return err
				}
				lowSpendQualified, err := wallet.TimedLowSpendQualified(
					ctx, tx, in.MemberID, storeID.Int64, in.Now,
				)
				if err != nil {
					return err
				}
				if _, err := vipbenefit.GrantFoodPayment(ctx, tx, vipbenefit.FoodPayment{
					PaymentOrderID: in.PaymentOrderID, BusinessOrderID: businessID,
					MemberID: in.MemberID, StoreID: storeID.Int64,
					PaidAt: in.Now, LowSpend: lowSpendQualified,
				}); err != nil {
					return err
				}
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
				CoinBalance:     &newBalance,
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
) (name string, unit, points int64, couponTemplateID *int64, payChannelsSnapshot string, err error) {
	var (
		itemStoreID int64
		itemStatus  string
		payChannels []byte
		stock       int64
	)
	if line.VariantID != nil {
		var variantStatus string
		const q = `SELECT cv.name, cv.price_cent, ci.points_reward, ci.grant_coupon_template_id, ci.store_id,
				ci.status, ci.pay_channels, cv.stock_quantity, cv.status
			FROM catalog_variants cv
			JOIN catalog_items ci ON ci.id = cv.item_id
			WHERE cv.id = ? AND cv.item_id = ? FOR UPDATE`
		err = tx.QueryRowContext(ctx, q, *line.VariantID, line.ItemID).Scan(
			&name, &unit, &points, &couponTemplateID, &itemStoreID, &itemStatus,
			&payChannels, &stock, &variantStatus,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, 0, nil, "", apperr.NotFound("catalog variant not found")
		}
		if err != nil {
			return "", 0, 0, nil, "", apperr.Internal(err)
		}
		if variantStatus != "active" && variantStatus != "published" {
			return "", 0, 0, nil, "", apperr.Conflict("catalog variant is not available")
		}
	} else {
		const q = `SELECT name, price_cent, points_reward, grant_coupon_template_id, store_id, status,
				pay_channels, stock_quantity
			FROM catalog_items WHERE id = ? FOR UPDATE`
		err = tx.QueryRowContext(ctx, q, line.ItemID).Scan(
			&name, &unit, &points, &couponTemplateID, &itemStoreID, &itemStatus, &payChannels, &stock,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, 0, nil, "", apperr.NotFound("catalog item not found")
		}
		if err != nil {
			return "", 0, 0, nil, "", apperr.Internal(err)
		}
	}
	if itemStoreID != storeID {
		return "", 0, 0, nil, "", apperr.Invalid("catalog item does not belong to the store")
	}
	if itemStatus != "published" {
		return "", 0, 0, nil, "", apperr.Conflict("catalog item is not available")
	}
	var channels []string
	if err := json.Unmarshal(payChannels, &channels); err != nil {
		return "", 0, 0, nil, "", apperr.Internal(err)
	}
	allowed := payMethod == PayMethodCoupon
	for _, channel := range channels {
		if channel == payMethod || (channel == "balance" && payMethod == PayMethodCoin) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", 0, 0, nil, "", apperr.Invalid("catalog item does not support the selected pay method")
	}
	// stock_quantity=0 means unlimited; positive values are finite stock.
	if stock > 0 && stock < int64(line.Quantity) {
		return "", 0, 0, nil, "", apperr.Conflict("insufficient catalog stock")
	}
	if line.VariantID != nil {
		const reserve = `UPDATE catalog_variants
			SET stock_quantity = CASE WHEN stock_quantity = 0 THEN 0 ELSE stock_quantity - ? END, updated_at = ?
			WHERE id = ? AND (stock_quantity = 0 OR stock_quantity >= ?)`
		res, err := tx.ExecContext(ctx, reserve, line.Quantity, now, *line.VariantID, line.Quantity)
		if err != nil {
			return "", 0, 0, nil, "", apperr.Internal(err)
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return "", 0, 0, nil, "", apperr.Conflict("insufficient catalog stock")
		}
	} else {
		const reserve = `UPDATE catalog_items
			SET stock_quantity = CASE WHEN stock_quantity = 0 THEN 0 ELSE stock_quantity - ? END, updated_at = ?
			WHERE id = ? AND (stock_quantity = 0 OR stock_quantity >= ?)`
		res, err := tx.ExecContext(ctx, reserve, line.Quantity, now, line.ItemID, line.Quantity)
		if err != nil {
			return "", 0, 0, nil, "", apperr.Internal(err)
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return "", 0, 0, nil, "", apperr.Conflict("insufficient catalog stock")
		}
	}
	if points < 0 {
		points = 0
	}
	return name, unit, points, couponTemplateID, string(payChannels), nil
}

func isProductCouponType(couponType string) bool {
	switch couponType {
	case "snack", "alcohol", "beverage", "drink", "meal", "gift":
		return true
	default:
		return false
	}
}

// activityPayMethodAllowed mirrors the public activity purchase UI. Coupon
// redemption is an activity-wide switch, while WeChat and coin keep the legacy
// ticket override/activity fallback rule. The repository enforces the result so
// a client cannot bypass a disabled payment method with a crafted request.
func activityPayMethodAllowed(ticketRaw, activityRaw []byte, payMethod string) (bool, error) {
	activityChannels, err := decodeActivityPayChannels(activityRaw)
	if err != nil {
		return false, err
	}
	if payMethod == PayMethodCoupon {
		return activityPayChannelsContain(activityChannels, payMethod), nil
	}
	channels, err := decodeActivityPayChannels(ticketRaw)
	if err != nil {
		return false, err
	}
	if len(channels) == 0 {
		channels = activityChannels
	}
	return activityPayChannelsContain(channels, payMethod), nil
}

func activityPayChannelsContain(channels []string, payMethod string) bool {
	for _, channel := range channels {
		if channel == "balance" {
			channel = PayMethodCoin
		}
		if channel == payMethod {
			return true
		}
	}
	return false
}

func decodeActivityPayChannels(raw []byte) ([]string, error) {
	var channels []string
	if err := json.Unmarshal(raw, &channels); err != nil {
		return nil, err
	}
	return channels, nil
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

// newCode mints a zero-padded six-digit verification code. The database unique
// index is authoritative; ticket insertion retries the rare random collision.
func newCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
