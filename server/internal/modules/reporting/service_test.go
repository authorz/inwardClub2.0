package reporting

import (
	"context"
	"testing"
	"time"
)

type fakeRepo struct {
	last       OverviewFilter
	out        Overview
	lastReport ReportFilter

	revenue      []RevenueRow
	catalogItems []CatalogItemStat
	activities   []ActivityStat
	coupons      []CouponStat
	records      []RecordRow
	members      []MemberStat
	reservations []ReservationStat
	stores       []StoreStat
	total        int64
}

func (f *fakeRepo) Overview(_ context.Context, flt OverviewFilter) (Overview, error) {
	f.last = flt
	return f.out, nil
}

func (f *fakeRepo) Revenue(_ context.Context, flt ReportFilter) ([]RevenueRow, int64, error) {
	f.lastReport = flt
	return f.revenue, f.total, nil
}

func (f *fakeRepo) CatalogItems(_ context.Context, flt ReportFilter) ([]CatalogItemStat, int64, error) {
	f.lastReport = flt
	return f.catalogItems, f.total, nil
}

func (f *fakeRepo) Activities(_ context.Context, flt ReportFilter) ([]ActivityStat, int64, error) {
	f.lastReport = flt
	return f.activities, f.total, nil
}

func (f *fakeRepo) Coupons(_ context.Context, flt ReportFilter) ([]CouponStat, int64, error) {
	f.lastReport = flt
	return f.coupons, f.total, nil
}

func (f *fakeRepo) Records(_ context.Context, flt ReportFilter) ([]RecordRow, int64, error) {
	f.lastReport = flt
	return f.records, f.total, nil
}

func (f *fakeRepo) Members(_ context.Context, flt ReportFilter) ([]MemberStat, int64, error) {
	f.lastReport = flt
	return f.members, f.total, nil
}

func (f *fakeRepo) Reservations(_ context.Context, flt ReportFilter) ([]ReservationStat, int64, error) {
	f.lastReport = flt
	return f.reservations, f.total, nil
}

func (f *fakeRepo) Stores(_ context.Context, flt ReportFilter) ([]StoreStat, int64, error) {
	f.lastReport = flt
	return f.stores, f.total, nil
}

func TestGetOverviewMaps(t *testing.T) {
	repo := &fakeRepo{out: Overview{
		StoreCount:          5,
		MemberCount:         120,
		OrderCount:          40,
		GrossSalesCent:      99900,
		TodayOrderCount:     8,
		TodayNewMemberCount: 3,
		CouponsIssued:       30,
		CouponsRedeemed:     12,
		WechatRevenue: OverviewBreakdown{
			Total: 99900, Today: 12000, Recharge: 50000, Food: 30000, Activity: 19900,
		},
		CoinConsumption: OverviewBreakdown{Total: 8800, Today: 500},
		Trend: []OverviewTrendPoint{{
			Date: time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC), WechatRevenueCent: 12000, OrderCount: 8,
		}},
	}}
	svc := NewService(repo)

	v, err := svc.GetOverview(context.Background(), OverviewFilter{})
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if v.StoreCount != 5 || v.GrossSalesCent != 99900 || v.CouponsRedeemed != 12 ||
		v.TodayOrderCount != 8 || v.WechatRevenue.Activity != 19900 ||
		v.CoinConsumption.Total != 8800 || len(v.Trend) != 1 || v.Trend[0].Date != "2026-07-24" {
		t.Fatalf("unexpected mapping: %+v", v)
	}
}

func TestGetOverviewPassesScope(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)

	id := int64(7)
	if _, err := svc.GetOverview(context.Background(), OverviewFilter{StoreID: &id}); err != nil {
		t.Fatalf("overview: %v", err)
	}
	if repo.last.StoreID == nil || *repo.last.StoreID != 7 {
		t.Fatalf("expected scope 7, got %v", repo.last.StoreID)
	}
}

func TestGetRevenueMaps(t *testing.T) {
	repo := &fakeRepo{
		revenue: []RevenueRow{{OrderCount: 3, GrossCent: 1500}},
		total:   1,
	}
	svc := NewService(repo)

	views, total, err := svc.GetRevenue(context.Background(), ReportFilter{})
	if err != nil {
		t.Fatalf("revenue: %v", err)
	}
	if total != 1 || len(views) != 1 || views[0].GrossCent != 1500 || views[0].OrderCount != 3 {
		t.Fatalf("unexpected mapping: %+v total=%d", views, total)
	}
}

func TestGetCouponsMapsAndPassesScope(t *testing.T) {
	repo := &fakeRepo{
		coupons: []CouponStat{{TemplateID: 9, Name: "welcome", Issued: 10, Redeemed: 4}},
		total:   1,
	}
	svc := NewService(repo)

	id := int64(3)
	views, total, err := svc.GetCoupons(context.Background(), ReportFilter{StoreID: &id})
	if err != nil {
		t.Fatalf("coupons: %v", err)
	}
	if total != 1 || len(views) != 1 || views[0].TemplateID != 9 || views[0].Issued != 10 || views[0].Redeemed != 4 {
		t.Fatalf("unexpected mapping: %+v total=%d", views, total)
	}
	if repo.lastReport.StoreID == nil || *repo.lastReport.StoreID != 3 {
		t.Fatalf("expected scope 3, got %v", repo.lastReport.StoreID)
	}
}
