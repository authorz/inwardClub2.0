// Package coupon owns member coupon entitlements and their redemption. Coupons
// reuse the catalog through templates; a member holds entitlements and a
// redemption records a single hit. This package exposes the mini-program read
// path (a member's coupons) and the redemption entry point; the transactional
// redemption itself (rule matching + verification within one tx) lands in a
// later milestone and is reported as not-yet-implemented.
package coupon

import "time"

// Entitlement statuses (mirrors coupon_entitlements.status).
const (
	StatusActive  = "active"
	StatusUsed    = "used"
	StatusVoid    = "void"
	StatusExpired = "expired"
)

// Coupon types (mirrors coupon_templates.coupon_type).
const (
	TypeExchange = "exchange"
	TypeDiscount = "discount"
	TypeCash     = "cash"
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
	EntitlementID int64
	EntitlementNo string
	TemplateID    int64
	Name          string
	Description   string
	CouponType    string
	ValueCent     int64
	Status        string
	ExpiresAt     *time.Time
	CreatedAt     time.Time
}
