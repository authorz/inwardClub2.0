package reporting

import "time"

// OverviewView is the dashboard overview payload returned to the console.
type OverviewView struct {
	StoreCount      int64 `json:"storeCount"`
	MemberCount     int64 `json:"memberCount"`
	OrderCount      int64 `json:"orderCount"`
	GrossSalesCent  int64 `json:"grossSalesCent"`
	CouponsIssued   int64 `json:"couponsIssued"`
	CouponsRedeemed int64 `json:"couponsRedeemed"`
}

// RevenueView is one day's revenue rollup.
type RevenueView struct {
	Date       time.Time `json:"date"`
	OrderCount int64     `json:"orderCount"`
	GrossCent  int64     `json:"grossCent"`
}

// CatalogItemView is a per-item sales rollup.
type CatalogItemView struct {
	ItemID    int64  `json:"itemId"`
	ItemName  string `json:"itemName"`
	SoldQty   int64  `json:"soldQty"`
	GrossCent int64  `json:"grossCent"`
}

// ActivityView is a per-activity participation rollup.
type ActivityView struct {
	ActivityID   int64  `json:"activityId"`
	ActivityName string `json:"activityName"`
	OrderCount   int64  `json:"orderCount"`
	TicketCount  int64  `json:"ticketCount"`
}

// CouponView is a per-coupon-template issuance/redemption rollup.
type CouponView struct {
	TemplateID int64  `json:"templateId"`
	Name       string `json:"name"`
	Issued     int64  `json:"issued"`
	Redeemed   int64  `json:"redeemed"`
}

// RecordView is one audit/redemption record line.
type RecordView struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"createdAt"`
}

// MemberView is a per-member points/order rollup.
type MemberView struct {
	MemberID      int64 `json:"memberId"`
	PointsBalance int64 `json:"pointsBalance"`
	OrderCount    int64 `json:"orderCount"`
}

// ReservationView is a per-day reservation rollup.
type ReservationView struct {
	Date  time.Time `json:"date"`
	Count int64     `json:"count"`
}

// StoreView is a per-store rollup.
type StoreView struct {
	StoreID    int64  `json:"storeId"`
	StoreName  string `json:"storeName"`
	OrderCount int64  `json:"orderCount"`
	GrossCent  int64  `json:"grossCent"`
}
