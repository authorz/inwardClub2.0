package order

import (
	"time"

	"github.com/inwardclub/server/internal/modules/payment"
)

// CreateFoodOrderRequest is the mini-program body for POST /mini/food-orders.
type CreateFoodOrderRequest struct {
	StoreID             int64          `json:"storeId" binding:"required"`
	TableID             *int64         `json:"tableId"`
	Items               []FoodLineItem `json:"items" binding:"required,min=1,dive"`
	Remark              string         `json:"remark"`
	PayMethod           string         `json:"payMethod" binding:"required,oneof=wechat coin coupon"`
	CouponEntitlementID *int64         `json:"couponEntitlementId,omitempty"`
}

// FoodLineItem is one requested line of a food order.
type FoodLineItem struct {
	ItemID    int64  `json:"itemId" binding:"required"`
	VariantID *int64 `json:"variantId"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

// CreateRechargeOrderRequest is the body for POST /mini/recharge-orders.
type CreateRechargeOrderRequest struct {
	AmountCent int64  `json:"amountCent" binding:"required,min=1"`
	PayMethod  string `json:"payMethod" binding:"required,oneof=wechat coin"`
}

// CreateActivityOrderRequest is the body for POST /mini/activity-orders.
type CreateActivityOrderRequest struct {
	ActivityID          int64  `json:"activityId" binding:"required"`
	TicketTypeID        int64  `json:"ticketTypeId" binding:"required"`
	Quantity            int    `json:"quantity" binding:"required,min=1"`
	PayMethod           string `json:"payMethod" binding:"required,oneof=wechat coin coupon"`
	CouponEntitlementID *int64 `json:"couponEntitlementId,omitempty"`
}

// FoodOrderView is the mini-program food order representation.
type FoodOrderView struct {
	ID                int64               `json:"id"`
	BusinessOrderID   int64               `json:"businessOrderId"`
	OrderNo           string              `json:"orderNo"`
	StoreID           int64               `json:"storeId"`
	TableID           *int64              `json:"tableId,omitempty"`
	TotalAmountCent   int64               `json:"totalAmountCent"`
	PointsEarned      int64               `json:"pointsEarned"`
	PaymentStatus     string              `json:"paymentStatus"`
	PayMethod         string              `json:"payMethod,omitempty"`
	PaidAmountCent    int64               `json:"paidAmountCent"`
	RefundAmountCent  int64               `json:"refundAmountCent"`
	StoreName         string              `json:"storeName,omitempty"`
	TableName         string              `json:"tableName,omitempty"`
	FulfillmentStatus string              `json:"fulfillmentStatus"`
	Remark            string              `json:"remark,omitempty"`
	Items             []FoodOrderItemView `json:"items,omitempty"`
	// PaymentOrderID / PaymentOrderNo are set on creation so the client can
	// continue to pay; they are omitted on list/read responses.
	PaymentOrderID int64     `json:"paymentOrderId,omitempty"`
	PaymentOrderNo string    `json:"paymentOrderNo,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// FoodOrderItemView is one line of a food order response.
type FoodOrderItemView struct {
	ItemID        int64  `json:"itemId"`
	VariantID     *int64 `json:"variantId,omitempty"`
	Name          string `json:"name"`
	UnitPriceCent int64  `json:"unitPriceCent"`
	Quantity      int    `json:"quantity"`
	PointsReward  int64  `json:"pointsReward"`
	SubtotalCent  int64  `json:"subtotalCent"`
	ImageURL      string `json:"imageUrl,omitempty"`
}

// RechargeOrderView is the mini-program recharge order representation.
type RechargeOrderView struct {
	ID              int64     `json:"id"`
	BusinessOrderNo string    `json:"businessOrderNo"`
	TotalAmountCent int64     `json:"totalAmountCent"`
	OrderStatus     string    `json:"orderStatus"`
	PaymentStatus   string    `json:"paymentStatus"`
	PaymentOrderID  int64     `json:"paymentOrderId,omitempty"`
	PaymentOrderNo  string    `json:"paymentOrderNo,omitempty"`
	PayMethod       string    `json:"payMethod,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// ActivityOrderView is the mini-program activity order representation.
type ActivityOrderView struct {
	ID              int64        `json:"id"`
	BusinessOrderID int64        `json:"businessOrderId"`
	ActivityID      int64        `json:"activityId"`
	TicketCount     int          `json:"ticketCount"`
	TotalAmountCent int64        `json:"totalAmountCent"`
	Status          string       `json:"status"`
	Tickets         []TicketView `json:"tickets,omitempty"`
	PaymentOrderID  int64        `json:"paymentOrderId,omitempty"`
	PaymentOrderNo  string       `json:"paymentOrderNo,omitempty"`
	PayMethod       string       `json:"payMethod,omitempty"`
	CreatedAt       time.Time    `json:"createdAt"`
}

// TicketView is one ticket of an activity order response.
type TicketView struct {
	ID           int64  `json:"id"`
	TicketNo     string `json:"ticketNo"`
	TicketTypeID int64  `json:"ticketTypeId"`
	SessionID    *int64 `json:"sessionId,omitempty"`
	PriceCent    int64  `json:"priceCent"`
	Status       string `json:"status"`
}

// WeChatJSAPIResponse wraps the prepay parameters handed to the mini program.
type WeChatJSAPIResponse struct {
	PaymentOrderID int64                `json:"paymentOrderId"`
	Prepay         payment.WeChatPrepay `json:"prepay"`
}
