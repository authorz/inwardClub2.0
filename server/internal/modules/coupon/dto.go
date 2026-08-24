package coupon

import "time"

// RedeemRequest is the body for POST /mini/coupon-redemptions. The member
// redeems one held entitlement, optionally at a specific store.
type RedeemRequest struct {
	EntitlementID int64               `json:"entitlementId" binding:"required"`
	StoreID       int64               `json:"storeId" binding:"required"`
	Items         []RedeemItemRequest `json:"items" binding:"required"`
}

type RedeemItemRequest struct {
	ItemID   int64 `json:"itemId" binding:"required"`
	Quantity int   `json:"quantity" binding:"required"`
}

// EligibleItemsView is returned before selection. Coupon data comes from the
// authenticated entitlement; product prices come from the catalog.
type EligibleItemsView struct {
	Coupon MemberCouponView   `json:"coupon"`
	Items  []EligibleItemView `json:"items"`
}

type EligibleItemView struct {
	ItemID        int64  `json:"itemId"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	ImageURL      string `json:"imageUrl,omitempty"`
	UnitPriceCent int64  `json:"unitPriceCent"`
	StockQuantity int64  `json:"stockQuantity"`
	SortOrder     int    `json:"sortOrder"`
}

// RedemptionOrderView is the mini-program "兑换订单" representation, consumed by
// pages/order-center and pages/redemption-order-detail.
type RedemptionOrderView struct {
	ID         int64      `json:"id"`
	OrderNo    string     `json:"orderNo"`
	Status     string     `json:"status"`
	Title      string     `json:"title"`
	CouponName string     `json:"couponName"`
	Qty        int        `json:"qty"`
	ValidUntil *time.Time `json:"validUntil,omitempty"`
	Code       string     `json:"code"`
	StoreName  string     `json:"storeName"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// MemberCouponView is the mini-program coupon representation.
type MemberCouponView struct {
	ID             int64     `json:"id,omitempty"`
	EntitlementID  int64     `json:"entitlementId"`
	EntitlementNo  string    `json:"entitlementNo"`
	TemplateID     int64     `json:"templateId"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	CouponType     string    `json:"couponType"`
	AdmissionCount int       `json:"admissionCount"`
	StoreID        *int64    `json:"storeId,omitempty"`
	Status         string    `json:"status"`
	ExpiresAt      string    `json:"expiresAt,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}
