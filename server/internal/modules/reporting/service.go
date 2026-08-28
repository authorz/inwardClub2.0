package reporting

import "context"

// Service provides console analytics read operations.
type Service struct {
	repo Repository
}

// NewService builds the reporting service.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// GetOverview returns the dashboard overview for the given scope.
func (s *Service) GetOverview(ctx context.Context, f OverviewFilter) (OverviewView, error) {
	o, err := s.repo.Overview(ctx, f)
	if err != nil {
		return OverviewView{}, err
	}
	trend := make([]OverviewTrendPointView, 0, len(o.Trend))
	for _, point := range o.Trend {
		trend = append(trend, OverviewTrendPointView{
			Date:              point.Date.Format("2006-01-02"),
			WechatRevenueCent: point.WechatRevenueCent,
			OrderCount:        point.OrderCount,
		})
	}
	return OverviewView{
		StoreCount:                   o.StoreCount,
		MemberCount:                  o.MemberCount,
		OrderCount:                   o.OrderCount,
		GrossSalesCent:               o.GrossSalesCent,
		OfflineCollectionRevenueCent: o.OfflineCollectionRevenueCent,
		TodayOrderCount:              o.TodayOrderCount,
		TodayGrossSalesCent:          o.TodayGrossSalesCent,
		TodayNewMemberCount:          o.TodayNewMemberCount,
		ActivityRevenueCent:          o.ActivityRevenueCent,
		TodayActivityRevenueCent:     o.TodayActivityRevenueCent,
		CouponsIssued:                o.CouponsIssued,
		CouponsRedeemed:              o.CouponsRedeemed,
		WechatRevenue:                overviewBreakdownView(o.WechatRevenue),
		CoinConsumption:              overviewBreakdownView(o.CoinConsumption),
		Trend:                        trend,
	}, nil
}

func overviewBreakdownView(b OverviewBreakdown) OverviewBreakdownView {
	return OverviewBreakdownView{
		Total:         b.Total,
		Today:         b.Today,
		Recharge:      b.Recharge,
		Food:          b.Food,
		Activity:      b.Activity,
		TodayRecharge: b.TodayRecharge,
		TodayFood:     b.TodayFood,
		TodayActivity: b.TodayActivity,
	}
}

// GetRevenue returns the daily revenue rollup for the given scope.
func (s *Service) GetRevenue(ctx context.Context, f ReportFilter) ([]RevenueView, int64, error) {
	rows, total, err := s.repo.Revenue(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	views := make([]RevenueView, 0, len(rows))
	for _, r := range rows {
		views = append(views, RevenueView{
			Date: r.Date, OrderCount: r.OrderCount, GrossCent: r.GrossCent,
			WechatOrderCount: r.WechatOrderCount, WechatRevenueCent: r.WechatRevenueCent,
			CoinOrderCount: r.CoinOrderCount, CoinConsumption: r.CoinConsumption,
		})
	}
	return views, total, nil
}

// GetCatalogItems returns the per-item sales rollup for the given scope.
func (s *Service) GetCatalogItems(ctx context.Context, f ReportFilter) ([]CatalogItemView, int64, error) {
	rows, total, err := s.repo.CatalogItems(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	views := make([]CatalogItemView, 0, len(rows))
	for _, r := range rows {
		views = append(views, CatalogItemView{ItemID: r.ItemID, ItemName: r.ItemName, SoldQty: r.SoldQty, GrossCent: r.GrossCent})
	}
	return views, total, nil
}

// GetActivities returns the per-activity participation rollup for the given scope.
func (s *Service) GetActivities(ctx context.Context, f ReportFilter) ([]ActivityView, int64, error) {
	rows, total, err := s.repo.Activities(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	views := make([]ActivityView, 0, len(rows))
	for _, r := range rows {
		views = append(views, ActivityView{ActivityID: r.ActivityID, ActivityName: r.ActivityName, OrderCount: r.OrderCount, TicketCount: r.TicketCount})
	}
	return views, total, nil
}

// GetCoupons returns the per-coupon-template issuance/redemption rollup for the given scope.
func (s *Service) GetCoupons(ctx context.Context, f ReportFilter) ([]CouponView, int64, error) {
	rows, total, err := s.repo.Coupons(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	views := make([]CouponView, 0, len(rows))
	for _, r := range rows {
		views = append(views, CouponView{TemplateID: r.TemplateID, Name: r.Name, Issued: r.Issued, Redeemed: r.Redeemed})
	}
	return views, total, nil
}

// GetRecords returns the audit/redemption record lines for the given scope.
func (s *Service) GetRecords(ctx context.Context, f ReportFilter) ([]RecordView, int64, error) {
	rows, total, err := s.repo.Records(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	views := make([]RecordView, 0, len(rows))
	for _, r := range rows {
		views = append(views, RecordView{ID: r.ID, Kind: r.Kind, CreatedAt: r.CreatedAt})
	}
	return views, total, nil
}

// GetMembers returns the per-member points/order rollup for the given scope.
func (s *Service) GetMembers(ctx context.Context, f ReportFilter) ([]MemberView, int64, error) {
	rows, total, err := s.repo.Members(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	views := make([]MemberView, 0, len(rows))
	for _, r := range rows {
		views = append(views, MemberView{MemberID: r.MemberID, PointsBalance: r.PointsBalance, OrderCount: r.OrderCount})
	}
	return views, total, nil
}

// GetReservations returns the per-day reservation rollup for the given scope.
func (s *Service) GetReservations(ctx context.Context, f ReportFilter) ([]ReservationView, int64, error) {
	rows, total, err := s.repo.Reservations(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	views := make([]ReservationView, 0, len(rows))
	for _, r := range rows {
		views = append(views, ReservationView{Date: r.Date, Count: r.Count})
	}
	return views, total, nil
}

// GetStores returns the per-store rollup for the given scope.
func (s *Service) GetStores(ctx context.Context, f ReportFilter) ([]StoreView, int64, error) {
	rows, total, err := s.repo.Stores(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	views := make([]StoreView, 0, len(rows))
	for _, r := range rows {
		averageOrderCent := int64(0)
		if r.PaidOrderCount > 0 {
			averageOrderCent = r.GrossCent / r.PaidOrderCount
		}
		views = append(views, StoreView{
			StoreID: r.StoreID, StoreName: r.StoreName,
			OrderCount: r.OrderCount, PaidOrderCount: r.PaidOrderCount,
			GrossCent: r.GrossCent, AverageOrderCent: averageOrderCent,
			FoodOrderCount: r.FoodOrderCount, FoodGrossCent: r.FoodGrossCent,
			ActivityOrderCount: r.ActivityOrderCount, ActivityGrossCent: r.ActivityGrossCent,
			UniqueMemberCount: r.UniqueMemberCount, ReservationCount: r.ReservationCount,
			CouponRedemptionCount: r.CouponRedemptionCount,
		})
	}
	return views, total, nil
}
