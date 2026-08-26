package coupon

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/inwardclub/server/internal/modules/catalog"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

const couponDateTimeLayout = "2006-01-02 15:04:05"

var couponBusinessLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// Service provides the mini-program coupon read path and redemption entry point.
type Service struct {
	repo    Repository
	catalog RedeemableCatalog
}

type RedeemableCatalog interface {
	ListCouponRedeemableItems(ctx context.Context, storeID, couponTemplateID, maxPriceCent int64) ([]catalog.ItemView, error)
}

// NewService builds the coupon service.
func NewService(repo Repository, catalogs ...RedeemableCatalog) *Service {
	var catalogSvc RedeemableCatalog
	if len(catalogs) > 0 {
		catalogSvc = catalogs[0]
	}
	return &Service{repo: repo, catalog: catalogSvc}
}

// ListCouponCategories returns the enabled categories configured by the admin
// console. Clients must not infer categories from a member's held coupons.
func (s *Service) ListCouponCategories(ctx context.Context) ([]CouponCategoryView, error) {
	categories, err := s.repo.ListActiveCategories(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]CouponCategoryView, 0, len(categories))
	for _, category := range categories {
		views = append(views, CouponCategoryView{
			ID: category.ID, Name: category.Name, BusinessType: category.BusinessType, SortOrder: category.SortOrder,
		})
	}
	return views, nil
}

// ListCoupons returns the member's coupons, optionally filtered by status.
func (s *Service) ListCoupons(ctx context.Context, memberID int64, status string, page httpx.Page) ([]MemberCouponView, int64, error) {
	coupons, total, err := s.repo.ListMemberCoupons(ctx, memberID, status, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]MemberCouponView, 0, len(coupons))
	for _, c := range coupons {
		views = append(views, couponView(c))
	}
	return views, total, nil
}

// ListActivityUsableCoupons returns the member's currently redeemable event
// ticket coupons for one activity. The repository applies all transactional
// eligibility constraints that can be known before a ticket tier is selected.
func (s *Service) ListActivityUsableCoupons(ctx context.Context, memberID, activityID int64, page httpx.Page) ([]MemberCouponView, int64, error) {
	if activityID <= 0 {
		return nil, 0, apperr.Invalid("活动信息不正确")
	}
	now := time.Now().UTC()
	usageDate := now.In(couponBusinessLocation).Format("2006-01-02")
	coupons, total, err := s.repo.ListActivityUsableCoupons(
		ctx, memberID, activityID, now, usageDate, page.Limit(), page.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	views := make([]MemberCouponView, 0, len(coupons))
	for _, coupon := range coupons {
		views = append(views, couponView(coupon))
	}
	return views, total, nil
}

// ListEligibleItems returns the authenticated coupon plus the products that
// the selected store has explicitly enabled for that concrete coupon template.
func (s *Service) ListEligibleItems(ctx context.Context, memberID, entitlementID, storeID int64) (EligibleItemsView, error) {
	c, items, err := s.eligibleCouponAndItems(ctx, memberID, entitlementID, storeID)
	if err != nil {
		return EligibleItemsView{}, err
	}
	views := make([]EligibleItemView, 0, len(items))
	for _, item := range items {
		views = append(views, EligibleItemView{
			ItemID: item.ID, Name: item.Name, Description: item.Description,
			ImageURL: item.ImageURL, UnitPriceCent: item.PriceCent,
			StockQuantity: item.StockQuantity, SortOrder: item.SortOrder,
		})
	}
	return EligibleItemsView{Coupon: couponView(c), Items: views}, nil
}

func (s *Service) eligibleCouponAndItems(ctx context.Context, memberID, entitlementID, storeID int64) (MemberCoupon, []catalog.ItemView, error) {
	if entitlementID <= 0 || storeID <= 0 {
		return MemberCoupon{}, nil, apperr.Invalid("优惠券和门店信息不正确")
	}
	c, err := s.repo.GetEntitlement(ctx, memberID, entitlementID)
	if err != nil {
		return MemberCoupon{}, nil, err
	}
	if c.Status != StatusActive {
		return MemberCoupon{}, nil, apperr.Conflict("优惠券当前不可使用")
	}
	if c.ExpiresAt != nil && c.ExpiresAt.Before(time.Now().UTC()) {
		return MemberCoupon{}, nil, apperr.Conflict("优惠券已过期")
	}
	if c.StoreID != nil && *c.StoreID != storeID {
		return MemberCoupon{}, nil, apperr.Invalid("优惠券不适用于当前门店")
	}
	if s.catalog == nil {
		return MemberCoupon{}, nil, apperr.Internal(fmt.Errorf("coupon: catalog service is not configured"))
	}
	items, err := s.catalog.ListCouponRedeemableItems(ctx, storeID, c.TemplateID, math.MaxInt64)
	if err != nil {
		return MemberCoupon{}, nil, err
	}
	return c, items, nil
}

// Redeem prices the selected products from the server catalog, persists the
// product/rule snapshots, and consumes the entitlement atomically.
func (s *Service) Redeem(ctx context.Context, memberID int64, idemKey string, req RedeemRequest) (MemberCouponView, error) {
	c, eligibleItems, err := s.eligibleCouponAndItems(ctx, memberID, req.EntitlementID, req.StoreID)
	if err != nil {
		return MemberCouponView{}, err
	}
	if len(req.Items) != 1 || req.Items[0].Quantity != 1 {
		return MemberCouponView{}, apperr.Invalid("一张券只能兑换一件商品")
	}
	eligible := make(map[int64]catalog.ItemView, len(eligibleItems))
	for _, item := range eligibleItems {
		eligible[item.ID] = item
	}
	seen := make(map[int64]struct{}, len(req.Items))
	snapshot := make([]RedemptionItemSnapshot, 0, len(req.Items))
	var totalCent int64
	for _, selected := range req.Items {
		item, ok := eligible[selected.ItemID]
		if !ok || selected.ItemID <= 0 {
			return MemberCouponView{}, apperr.Invalid("所选商品不可使用当前优惠券兑换")
		}
		if _, duplicated := seen[selected.ItemID]; duplicated {
			return MemberCouponView{}, apperr.Invalid("兑换商品不能重复提交")
		}
		seen[selected.ItemID] = struct{}{}
		if selected.Quantity != 1 {
			return MemberCouponView{}, apperr.Invalid("一张券只能兑换一件商品")
		}
		// stock_quantity=0 means unlimited; positive values are finite stock.
		if item.StockQuantity > 0 && int64(selected.Quantity) > item.StockQuantity {
			return MemberCouponView{}, apperr.Conflict("所选商品库存不足")
		}
		subtotalCent := item.PriceCent * int64(selected.Quantity)
		totalCent += subtotalCent
		snapshot = append(snapshot, RedemptionItemSnapshot{
			ItemID: item.ID, Name: item.Name, ImageURL: item.ImageURL,
			UnitPriceCent: item.PriceCent, Quantity: selected.Quantity, SubtotalCent: subtotalCent,
		})
	}
	itemSnapshotJSON, err := marshalSnapshot(snapshot)
	if err != nil {
		return MemberCouponView{}, apperr.Internal(err)
	}
	ruleSnapshotJSON, err := marshalSnapshot(RedemptionRuleSnapshot{
		CouponTemplateID: c.TemplateID, CouponType: c.CouponType,
		RedeemedAmountCent: totalCent,
	})
	if err != nil {
		return MemberCouponView{}, apperr.Internal(err)
	}
	now := time.Now().UTC()
	redeemed, err := s.repo.Redeem(ctx, RedeemInput{
		MemberID:         memberID,
		EntitlementID:    req.EntitlementID,
		StoreID:          req.StoreID,
		RedemptionNo:     fmt.Sprintf("RD%s%04d", now.Format("20060102150405"), rand.Intn(10000)),
		IdemKey:          idemKey,
		Now:              now,
		ItemSnapshotJSON: itemSnapshotJSON,
		MatchedRuleJSON:  ruleSnapshotJSON,
		Items:            snapshot,
	})
	if err != nil {
		return MemberCouponView{}, err
	}
	return couponView(redeemed), nil
}

// ListRedemptions returns the member's redemption orders (兑换订单), newest first.
func (s *Service) ListRedemptions(ctx context.Context, memberID int64, page httpx.Page) ([]RedemptionOrderView, int64, error) {
	orders, total, err := s.repo.ListRedemptions(ctx, memberID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]RedemptionOrderView, 0, len(orders))
	for _, o := range orders {
		views = append(views, redemptionView(o))
	}
	return views, total, nil
}

// GetRedemption returns one redemption order owned by the member.
func (s *Service) GetRedemption(ctx context.Context, memberID, id int64) (RedemptionOrderView, error) {
	o, err := s.repo.GetRedemption(ctx, memberID, id)
	if err != nil {
		return RedemptionOrderView{}, err
	}
	return redemptionView(o), nil
}

func redemptionView(o RedemptionOrder) RedemptionOrderView {
	return RedemptionOrderView{
		ID:         o.ID,
		OrderNo:    o.RedemptionNo,
		Status:     o.Status,
		Title:      o.Title,
		CouponName: o.CouponName,
		Qty:        o.Qty,
		ValidUntil: o.ValidUntil,
		Code:       o.Code,
		StoreName:  o.StoreName,
		CreatedAt:  o.CreatedAt,
	}
}

func couponView(c MemberCoupon) MemberCouponView {
	return MemberCouponView{
		ID:             c.RedemptionID,
		EntitlementID:  c.EntitlementID,
		EntitlementNo:  c.EntitlementNo,
		TemplateID:     c.TemplateID,
		Name:           c.Name,
		Description:    c.Description,
		CategoryID:     c.CategoryID,
		CategoryName:   c.CategoryName,
		CouponType:     c.CouponType,
		AdmissionCount: c.AdmissionCount,
		StoreID:        c.StoreID,
		Status:         c.Status,
		ExpiresAt:      formatCouponDateTime(c.ExpiresAt),
		CreatedAt:      c.CreatedAt,
	}
}

func formatCouponDateTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.In(couponBusinessLocation).Format(couponDateTimeLayout)
}
