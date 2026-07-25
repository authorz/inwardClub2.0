package reporting

import "time"

// OverviewView is the dashboard overview payload returned to the console.
type OverviewView struct {
	StoreCount               int64                    `json:"storeCount"`
	MemberCount              int64                    `json:"memberCount"`
	OrderCount               int64                    `json:"orderCount"`
	GrossSalesCent           int64                    `json:"grossSalesCent"`
	TodayOrderCount          int64                    `json:"todayOrderCount"`
	TodayGrossSalesCent      int64                    `json:"todayGrossSalesCent"`
	TodayNewMemberCount      int64                    `json:"todayNewMemberCount"`
	ActivityRevenueCent      int64                    `json:"activityRevenueCent"`
	TodayActivityRevenueCent int64                    `json:"todayActivityRevenueCent"`
	CouponsIssued            int64                    `json:"couponsIssued"`
	CouponsRedeemed          int64                    `json:"couponsRedeemed"`
	WechatRevenue            OverviewBreakdownView    `json:"wechatRevenue"`
	CoinConsumption          OverviewBreakdownView    `json:"coinConsumption"`
	Trend                    []OverviewTrendPointView `json:"trend"`
}

// OverviewBreakdownView is a dashboard payment breakdown.
type OverviewBreakdownView struct {
	Total         int64 `json:"total"`
	Today         int64 `json:"today"`
	Recharge      int64 `json:"recharge"`
	Food          int64 `json:"food"`
	Activity      int64 `json:"activity"`
	TodayRecharge int64 `json:"todayRecharge"`
	TodayFood     int64 `json:"todayFood"`
	TodayActivity int64 `json:"todayActivity"`
}

// OverviewTrendPointView is one local calendar day in the dashboard trend.
type OverviewTrendPointView struct {
	Date              string `json:"date"`
	WechatRevenueCent int64  `json:"wechatRevenueCent"`
	OrderCount        int64  `json:"orderCount"`
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
