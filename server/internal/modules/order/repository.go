package order

import (
	"context"
	"database/sql"
	"errors"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Repository is the order persistence port. Every read is scoped by member_id so
// a member can only ever see their own orders. Write orchestration (creating
// business_orders + payment_orders atomically, reserving stock, snapshotting
// prices, settling coin payments) lives in write_repository.go; each write runs
// in one transaction and claims the request Idempotency-Key as its duplicate guard.
type Repository interface {
	ListFoodOrders(ctx context.Context, memberID int64, limit, offset int) ([]FoodOrder, int64, error)
	GetFoodOrder(ctx context.Context, memberID, id int64) (FoodOrder, []FoodOrderItem, error)

	ListRechargeOrders(ctx context.Context, memberID int64, limit, offset int) ([]RechargeOrder, int64, error)
	GetRechargeOrder(ctx context.Context, memberID, id int64) (RechargeOrder, error)

	ListActivityOrders(ctx context.Context, memberID int64, limit, offset int) ([]ActivityOrder, int64, error)
	GetActivityOrder(ctx context.Context, memberID, id int64) (ActivityOrder, []Ticket, error)

	// ListMemberTickets returns every issued ticket owned by the member, joined
	// with its activity, ticket type and store for the "my tickets" screen.
	ListMemberTickets(ctx context.Context, memberID int64) ([]MemberTicket, error)

	GetPaymentOrder(ctx context.Context, id int64) (PaymentOrder, error)

	// Write orchestration. Each runs in one transaction, resolves catalog prices
	// server-side (never trusting client amounts), claims the idempotency key as
	// the duplicate guard and creates the business + business-specific + payment
	// rows atomically.
	CreateFoodOrder(ctx context.Context, in FoodOrderCreate) (FoodOrder, []FoodOrderItem, PaymentOrder, error)
	CreateRechargeOrder(ctx context.Context, in RechargeOrderCreate) (RechargeOrder, PaymentOrder, error)
	CreateActivityOrder(ctx context.Context, in ActivityOrderCreate) (ActivityOrder, []Ticket, PaymentOrder, error)
	// SettleByCoin debits the member's coin wallet and settles the payment order
	// in one transaction (append-only ledger + payment transaction + status flip).
	SettleByCoin(ctx context.Context, in CoinPayment) error

	// ExpireActivityOrders closes unpaid activity orders created before
	// createdBefore (spec §11 activity-order:expire): per order, in one
	// transaction, it releases the reserved ticket stock, expires the order's
	// pending tickets, expires the order and its pending payment order and closes
	// the business order. Each order is re-checked under a row lock so a
	// concurrent settlement always wins cleanly. Returns the number expired.
	ExpireActivityOrders(ctx context.Context, createdBefore, now time.Time) (int64, error)

	// ExpireTickets expires paid-but-unused tickets whose event has ended — the
	// ticket half of the ticket-coupon:expire sweep (spec §11). The deadline is
	// the ticket's session end, falling back to the activity end. Guarded by
	// status='active' so it is idempotent. Returns the number expired.
	ExpireTickets(ctx context.Context, now time.Time) (int64, error)
}

type sqlRepository struct{ db *platdb.DB }

// NewRepository builds the MySQL order repository.
func NewRepository(db *platdb.DB) Repository { return &sqlRepository{db: db} }

func (r *sqlRepository) ListFoodOrders(ctx context.Context, memberID int64, limit, offset int) ([]FoodOrder, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM food_orders WHERE member_id = ?`, memberID).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	const q = `SELECT fo.id, fo.business_order_id, bo.business_order_no, fo.store_id, fo.member_id, fo.table_id, fo.total_amount_cent,
		fo.points_earned, bo.payment_status, COALESCE(po.pay_method, ''),
		COALESCE((SELECT pt.amount_cent FROM payment_transactions pt WHERE pt.payment_order_id = po.id AND pt.status = 'success' ORDER BY pt.id DESC LIMIT 1), 0),
		COALESCE((SELECT SUM(ro.amount_cent) FROM refund_orders ro WHERE ro.payment_order_id = po.id AND ro.status = 'succeeded'), 0),
		COALESCE(s.name, ''), COALESCE(t.name, ''), fo.fulfillment_status, fo.remark, fo.created_at, fo.updated_at
		FROM food_orders fo JOIN business_orders bo ON bo.id = fo.business_order_id
		LEFT JOIN payment_orders po ON po.business_order_id = bo.id
		LEFT JOIN stores s ON s.id = fo.store_id LEFT JOIN tables t ON t.id = fo.table_id
		WHERE fo.member_id = ? ORDER BY fo.id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, memberID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []FoodOrder
	for rows.Next() {
		o, err := scanFoodOrder(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) GetFoodOrder(ctx context.Context, memberID, id int64) (FoodOrder, []FoodOrderItem, error) {
	const q = `SELECT fo.id, fo.business_order_id, bo.business_order_no, fo.store_id, fo.member_id, fo.table_id, fo.total_amount_cent,
		fo.points_earned, bo.payment_status, COALESCE(po.pay_method, ''),
		COALESCE((SELECT pt.amount_cent FROM payment_transactions pt WHERE pt.payment_order_id = po.id AND pt.status = 'success' ORDER BY pt.id DESC LIMIT 1), 0),
		COALESCE((SELECT SUM(ro.amount_cent) FROM refund_orders ro WHERE ro.payment_order_id = po.id AND ro.status = 'succeeded'), 0),
		COALESCE(s.name, ''), COALESCE(t.name, ''), fo.fulfillment_status, fo.remark, fo.created_at, fo.updated_at
		FROM food_orders fo JOIN business_orders bo ON bo.id = fo.business_order_id
		LEFT JOIN payment_orders po ON po.business_order_id = bo.id
		LEFT JOIN stores s ON s.id = fo.store_id LEFT JOIN tables t ON t.id = fo.table_id
		WHERE fo.id = ? AND fo.member_id = ?`
	o, err := scanFoodOrder(r.db.QueryRowContext(ctx, q, id, memberID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return FoodOrder{}, nil, apperr.NotFound("food order not found")
		}
		return FoodOrder{}, nil, apperr.Internal(err)
	}
	const iq = `SELECT id, food_order_id, item_id, variant_id, name_snapshot, unit_price_cent,
		quantity, points_reward_snapshot, subtotal_cent
		FROM food_order_items WHERE food_order_id = ? ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, iq, o.ID)
	if err != nil {
		return FoodOrder{}, nil, apperr.Internal(err)
	}
	defer rows.Close()
	var items []FoodOrderItem
	for rows.Next() {
		var it FoodOrderItem
		if err := rows.Scan(&it.ID, &it.FoodOrderID, &it.ItemID, &it.VariantID, &it.NameSnapshot,
			&it.UnitPriceCent, &it.Quantity, &it.PointsReward, &it.SubtotalCent); err != nil {
			return FoodOrder{}, nil, apperr.Internal(err)
		}
		items = append(items, it)
	}
	return o, items, rows.Err()
}

func (r *sqlRepository) ListRechargeOrders(ctx context.Context, memberID int64, limit, offset int) ([]RechargeOrder, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM business_orders WHERE member_id = ? AND order_type = ?`,
		memberID, OrderTypeRecharge).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	const q = `SELECT id, business_order_no, store_id, member_id, total_amount_cent,
		order_status, payment_status, created_at, updated_at
		FROM business_orders WHERE member_id = ? AND order_type = ?
		ORDER BY id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, memberID, OrderTypeRecharge, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []RechargeOrder
	for rows.Next() {
		o, err := scanRechargeOrder(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) GetRechargeOrder(ctx context.Context, memberID, id int64) (RechargeOrder, error) {
	const q = `SELECT id, business_order_no, store_id, member_id, total_amount_cent,
		order_status, payment_status, created_at, updated_at
		FROM business_orders WHERE id = ? AND member_id = ? AND order_type = ?`
	o, err := scanRechargeOrder(r.db.QueryRowContext(ctx, q, id, memberID, OrderTypeRecharge))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RechargeOrder{}, apperr.NotFound("recharge order not found")
		}
		return RechargeOrder{}, apperr.Internal(err)
	}
	return o, nil
}

func (r *sqlRepository) ListActivityOrders(ctx context.Context, memberID int64, limit, offset int) ([]ActivityOrder, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM activity_orders WHERE member_id = ?`, memberID).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	const q = `SELECT id, business_order_id, activity_id, store_id, member_id, ticket_count,
		total_amount_cent, status, created_at, updated_at
		FROM activity_orders WHERE member_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, memberID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var out []ActivityOrder
	for rows.Next() {
		o, err := scanActivityOrder(rows)
		if err != nil {
			return nil, 0, apperr.Internal(err)
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *sqlRepository) GetActivityOrder(ctx context.Context, memberID, id int64) (ActivityOrder, []Ticket, error) {
	const q = `SELECT id, business_order_id, activity_id, store_id, member_id, ticket_count,
		total_amount_cent, status, created_at, updated_at
		FROM activity_orders WHERE id = ? AND member_id = ?`
	o, err := scanActivityOrder(r.db.QueryRowContext(ctx, q, id, memberID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ActivityOrder{}, nil, apperr.NotFound("activity order not found")
		}
		return ActivityOrder{}, nil, apperr.Internal(err)
	}
	const tq = `SELECT id, ticket_no, activity_id, ticket_type_id, session_id, price_cent, status
		FROM tickets WHERE activity_order_id = ? ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, tq, o.ID)
	if err != nil {
		return ActivityOrder{}, nil, apperr.Internal(err)
	}
	defer rows.Close()
	var tickets []Ticket
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(&t.ID, &t.TicketNo, &t.ActivityID, &t.TicketTypeID, &t.SessionID,
			&t.PriceCent, &t.Status); err != nil {
			return ActivityOrder{}, nil, apperr.Internal(err)
		}
		tickets = append(tickets, t)
	}
	return o, tickets, rows.Err()
}

func (r *sqlRepository) GetPaymentOrder(ctx context.Context, id int64) (PaymentOrder, error) {
	const q = `SELECT id, payment_order_no, business_order_id, member_id, amount_cent,
		pay_method, status, created_at FROM payment_orders WHERE id = ?`
	var o PaymentOrder
	err := r.db.QueryRowContext(ctx, q, id).Scan(&o.ID, &o.PaymentOrderNo, &o.BusinessOrderID,
		&o.MemberID, &o.AmountCent, &o.PayMethod, &o.Status, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PaymentOrder{}, apperr.NotFound("payment order not found")
		}
		return PaymentOrder{}, apperr.Internal(err)
	}
	return o, nil
}

// scanner abstracts *sql.Row and *sql.Rows for the row-scan helpers.
type scanner interface {
	Scan(dest ...any) error
}

func scanFoodOrder(s scanner) (FoodOrder, error) {
	var o FoodOrder
	err := s.Scan(&o.ID, &o.BusinessOrderID, &o.BusinessOrderNo, &o.StoreID, &o.MemberID, &o.TableID,
		&o.TotalAmountCent, &o.PointsEarned, &o.PaymentStatus, &o.PayMethod, &o.PaidAmountCent,
		&o.RefundAmountCent, &o.StoreName, &o.TableName, &o.FulfillmentStatus, &o.Remark, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}

func scanRechargeOrder(s scanner) (RechargeOrder, error) {
	var o RechargeOrder
	err := s.Scan(&o.ID, &o.BusinessOrderNo, &o.StoreID, &o.MemberID, &o.TotalAmountCent,
		&o.OrderStatus, &o.PaymentStatus, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}

func scanActivityOrder(s scanner) (ActivityOrder, error) {
	var o ActivityOrder
	err := s.Scan(&o.ID, &o.BusinessOrderID, &o.ActivityID, &o.StoreID, &o.MemberID,
		&o.TicketCount, &o.TotalAmountCent, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}
