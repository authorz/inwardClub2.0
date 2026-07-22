// Package reporting owns the console analytics read models. Phase-1 exposes the
// admin overview dashboard; per-store and time-series reports are layered on
// later against the same OverviewFilter scope model.
package reporting

import (
	"time"

	"github.com/inwardclub/server/internal/platform/httpx"
)

// OverviewFilter scopes an overview query. A nil StoreID aggregates across every
// store (admin dashboard); a set StoreID pins the metrics to one store (store
// console). It is never read from a request parameter for store scope — it comes
// from the JWT scope at the handler.
type OverviewFilter struct {
	StoreID *int64
}

// Overview holds the headline counters shown on the console dashboard.
type Overview struct {
	StoreCount      int64
	MemberCount     int64
	OrderCount      int64
	GrossSalesCent  int64
	CouponsIssued   int64
	CouponsRedeemed int64
}

// ReportFilter scopes a list-style report query. A nil StoreID aggregates across
// every store; a set StoreID pins the report to one store. From/To bound the
// query by date; either may be nil for an unbounded edge. Store scope is never
// read from a request parameter — it comes from the JWT scope at the handler.
type ReportFilter struct {
	StoreID *int64
	From    *time.Time
	To      *time.Time
	Page    httpx.Page
}

// RevenueRow is one day's revenue rollup.
type RevenueRow struct {
	Date       time.Time
	OrderCount int64
	GrossCent  int64
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
	StoreID    int64
	StoreName  string
	OrderCount int64
	GrossCent  int64
}
