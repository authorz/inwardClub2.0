// Package order owns the mini-program order lifecycle across the three order
// families that share the unified payment spine: food, recharge and activity.
// Every order is backed by a business_orders row and paid through a
// payment_orders row; this package never settles payments itself (settlement is
// driven by the payment callbacks in the payment module) — it creates orders and
// initiates payment.
package order

import "time"

// Pay methods a mini-program member may choose for a payment order.
const (
	PayMethodWeChat = "wechat"
	PayMethodCoin   = "coin"
	PayMethodCoupon = "coupon"
)

// Business order types (mirrors business_orders.order_type).
const (
	OrderTypeFood     = "food"
	OrderTypeRecharge = "recharge"
	OrderTypeActivity = "activity"
)

// Payment order statuses (mirrors payment_orders.status).
const (
	PaymentStatusPending = "pending"
	PaymentStatusPaid    = "paid"
	PaymentStatusFailed  = "failed"
	// PaymentStatusExpired is the terminal state for an unpaid payment order that
	// timed out (spec §11 activity-order:expire). Settlement already refuses any
	// non-pending order, so an expired order can never be resurrected to paid.
	PaymentStatusExpired = "expired"
)

// Ticket lifecycle values (mirrors tickets.status in 00008_activity.sql). Only
// the states the order lifecycle and the expiry sweeps transition through are
// named here; used/refunded stay inline where they are read.
const (
	TicketStatusPending = "pending" // issued, awaiting payment
	TicketStatusActive  = "active"  // paid, usable until the event ends
	TicketStatusExpired = "expired" // unpaid order timed out, or event passed unused
)

// businessStatusExpired is the business_orders.order_status a timed-out unpaid
// order is closed to. business_orders.order_status has no Go enum (the create
// path inlines 'created'), so the sweep inlines this literal in its SQL and this
// constant documents the value for readers of the state machine.
const businessStatusExpired = "expired"

// FoodOrder is a dine-in / pickup food order.
type FoodOrder struct {
	ID                int64
	BusinessOrderID   int64
	BusinessOrderNo   string
	StoreID           int64
	MemberID          int64
	TableID           *int64
	TotalAmountCent   int64
	PointsEarned      int64
	PaymentStatus     string
	PayMethod         string
	PaidAmountCent    int64
	RefundAmountCent  int64
	StoreName         string
	TableName         string
	FulfillmentStatus string
	Remark            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// FoodOrderItem is one snapshotted line of a food order.
type FoodOrderItem struct {
	ID               int64
	FoodOrderID      int64
	ItemID           int64
	VariantID        *int64
	NameSnapshot     string
	UnitPriceCent    int64
	Quantity         int
	PayChannels      string
	PointsReward     int64
	CouponTemplateID *int64
	SubtotalCent     int64
	AssetID          *int64
	ImageURL         string
}

// RechargeOrder is a wallet top-up order. Recharge has no dedicated table; it is
// a business_orders row of type "recharge".
type RechargeOrder struct {
	ID              int64
	BusinessOrderNo string
	StoreID         *int64
	MemberID        int64
	TotalAmountCent int64
	OrderStatus     string
	PaymentStatus   string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ActivityOrder is a ticket purchase order.
type ActivityOrder struct {
	ID              int64
	BusinessOrderID int64
	ActivityID      int64
	StoreID         *int64
	MemberID        int64
	TicketCount     int
	TotalAmountCent int64
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Ticket is one issued ticket instance belonging to an activity order.
type Ticket struct {
	ID           int64
	TicketNo     string
	ActivityID   int64
	TicketTypeID int64
	SessionID    *int64
	PriceCent    int64
	Status       string
}

// PaymentOrder is the unified payment record a member pays against.
type PaymentOrder struct {
	ID              int64
	PaymentOrderNo  string
	BusinessOrderID int64
	MemberID        *int64
	AmountCent      int64
	PayMethod       string
	Status          string
	CreatedAt       time.Time
}
