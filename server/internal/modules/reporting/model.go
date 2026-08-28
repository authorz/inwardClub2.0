// Package reporting owns the console analytics read models for dashboard and
// list-style reports across admin and store scopes.
package reporting

import (
	"time"

	"github.com/inwardclub/server/internal/platform/httpx"
)

// OverviewFilter scopes an overview query. A nil StoreID aggregates across every
// store; a set StoreID selects one store. Store-console scope always comes from
// the JWT, while the headquarters console may select a store by query.
type OverviewFilter struct {
	StoreID *int64
}

// Overview holds the headline counters shown on the console dashboard.
type Overview struct {
	StoreCount                   int64
	MemberCount                  int64
	OrderCount                   int64
	GrossSalesCent               int64
	OfflineCollectionRevenueCent int64
	TodayOrderCount              int64
	TodayGrossSalesCent          int64
	TodayNewMemberCount          int64
	ActivityRevenueCent          int64
	TodayActivityRevenueCent     int64
	CouponsIssued                int64
	CouponsRedeemed              int64
	WechatRevenue                OverviewBreakdown
	CoinConsumption              OverviewBreakdown
	Trend                        []OverviewTrendPoint
}

// OverviewBreakdown splits one payment asset by business type and today.
type OverviewBreakdown struct {
	Total         int64
	Today         int64
	Recharge      int64
	Food          int64
	Activity      int64
	TodayRecharge int64
	TodayFood     int64
	TodayActivity int64
}

// OverviewTrendPoint is one local calendar day in the dashboard trend.
type OverviewTrendPoint struct {
	Date              time.Time
	WechatRevenueCent int64
	OrderCount        int64
}

// ReportFilter scopes a list-style report query. A nil StoreID aggregates across
// every store; a set StoreID pins the report to one store. From/To bound the
// query by date; either may be nil for an unbounded edge. Store-console scope
// comes from the JWT; the headquarters console may select a store by query.
type ReportFilter struct {
	StoreID *int64
	From    *time.Time
	To      *time.Time
	Page    httpx.Page
}

// RevenueRow is one day's revenue rollup.
type RevenueRow struct {
	Date              time.Time
	OrderCount        int64
	GrossCent         int64
	WechatOrderCount  int64
	WechatRevenueCent int64
	CoinOrderCount    int64
	CoinConsumption   int64
}

// CatalogItemStat is a per-item sales rollup.
type CatalogItemStat struct {
	ItemID    int64
	ItemName  string
	SoldQty   int64
	GrossCent int64
}

// ActivityStat is a per-activity participation rollup.
type ActivityStat struct {
	ActivityID   int64
	ActivityName string
	OrderCount   int64
	TicketCount  int64
}

// CouponStat is a per-coupon-template issuance/redemption rollup.
type CouponStat struct {
	TemplateID int64
	Name       string
	Issued     int64
	Redeemed   int64
}

// RecordRow is one audit/redemption record line.
type RecordRow struct {
	ID        int64
	Kind      string
	CreatedAt time.Time
}

// MemberStat is a per-member points/order rollup.
type MemberStat struct {
	MemberID      int64
	PointsBalance int64
	OrderCount    int64
}

// ReservationStat is a per-day reservation rollup.
type ReservationStat struct {
	Date  time.Time
	Count int64
}

// StoreStat is a per-store rollup.
type StoreStat struct {
	StoreID               int64
	StoreName             string
	OrderCount            int64
	PaidOrderCount        int64
	GrossCent             int64
	FoodOrderCount        int64
	FoodGrossCent         int64
	ActivityOrderCount    int64
	ActivityGrossCent     int64
	UniqueMemberCount     int64
	ReservationCount      int64
	CouponRedemptionCount int64
}
