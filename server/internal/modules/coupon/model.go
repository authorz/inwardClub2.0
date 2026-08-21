// Package coupon owns member coupon entitlements and their redemption. Coupons
// reuse the catalog through templates; a member holds entitlements and a
// redemption records a single hit. This package exposes the mini-program read
// path, eligible-product query, and transactional redemption entry point.
package coupon

import (
	"encoding/json"
	"time"
)

// Entitlement statuses (mirrors coupon_entitlements.status).
const (
	StatusActive  = "active"
	StatusUsed    = "used"
	StatusVoid    = "void"
	StatusExpired = "expired"
)

// Coupon types. One entitlement always exchanges one product or one ticket.
const (
	TypeEventTicket = "event_ticket"
	TypeSnack       = "snack"
	TypeAlcohol     = "alcohol"
	TypeBeverage    = "beverage"
	TypeMeal        = "meal"
)

// RedemptionOrder is one coupon_redemptions row joined with its entitlement,
// template and store for the mini-program "兑换订单" read path.
type RedemptionOrder struct {
	ID           int64
	RedemptionNo string
	Status       string // entitlement status (used/void/expired/active)
	Title        string
	CouponName   string
	Qty          int
	ValidUntil   *time.Time
	Code         string
	StoreName    string
	CreatedAt    time.Time
}

// MemberCoupon is one entitlement joined with its template display fields.
type MemberCoupon struct {
	RedemptionID  int64
	EntitlementID int64
	EntitlementNo string
	TemplateID    int64
	Name          string
	Description   string
	CouponType    string
	ValueCent     int64
	StoreID       *int64
	Status        string
	ExpiresAt     *time.Time
	CreatedAt     time.Time
}

// RedemptionItemSnapshot is the server-priced product selection persisted on
// a coupon redemption. Amounts are integer RMB cents.
type RedemptionItemSnapshot struct {
	ItemID        int64  `json:"itemId"`
	Name          string `json:"name"`
	ImageURL      string `json:"imageUrl,omitempty"`
	UnitPriceCent int64  `json:"unitPriceCent"`
	Quantity      int    `json:"quantity"`
	SubtotalCent  int64  `json:"subtotalCent"`
}

type RedemptionRuleSnapshot struct {
	CouponTemplateID   int64  `json:"couponTemplateId"`
	CouponType         string `json:"couponType"`
	RedeemedAmountCent int64  `json:"redeemedAmountCent"`
}

func marshalSnapshot(value any) ([]byte, error) { return json.Marshal(value) }
