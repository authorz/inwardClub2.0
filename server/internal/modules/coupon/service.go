package coupon

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/inwardclub/server/internal/modules/catalog"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Service provides the mini-program coupon read path and redemption entry point.
type Service struct {
	repo    Repository
	catalog RedeemableCatalog
}

type RedeemableCatalog interface {
	ListCouponRedeemableItems(ctx context.Context, storeID int64, couponType string, maxPriceCent int64) ([]catalog.ItemView, error)
}

// NewService builds the coupon service.
func NewService(repo Repository, catalogs ...RedeemableCatalog) *Service {
	var catalogSvc RedeemableCatalog
	if len(catalogs) > 0 {
		catalogSvc = catalogs[0]
	}
	return &Service{repo: repo, catalog: catalogSvc}
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

// ListEligibleItems returns the authenticated coupon plus the products that
// the selected store has explicitly enabled for that coupon type.
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
	if c.ValueCent <= 0 {
		return MemberCoupon{}, nil, apperr.Invalid("优惠券面额不正确")
	}
	if s.catalog == nil {
		return MemberCoupon{}, nil, apperr.Internal(fmt.Errorf("coupon: catalog service is not configured"))
	}
	items, err := s.catalog.ListCouponRedeemableItems(ctx, storeID, c.CouponType, c.ValueCent)
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
	if len(req.Items) == 0 {
		return MemberCouponView{}, apperr.Invalid("请至少选择一件商品")
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
		if selected.Quantity <= 0 {
			return MemberCouponView{}, apperr.Invalid("兑换数量必须大于 0")
		}
		// stock_quantity=0 means unlimited; positive values are finite stock.
		if item.StockQuantity > 0 && int64(selected.Quantity) > item.StockQuantity {
			return MemberCouponView{}, apperr.Conflict("所选商品库存不足")
		}
		if item.PriceCent <= 0 || int64(selected.Quantity) > c.ValueCent/item.PriceCent {
			return MemberCouponView{}, apperr.Invalid("所选商品金额不能超过券面额")
		}
		subtotalCent := item.PriceCent * int64(selected.Quantity)
		if totalCent > c.ValueCent-subtotalCent {
			return MemberCouponView{}, apperr.Invalid("所选商品金额不能超过券面额")
		}
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
		CouponType: c.CouponType, CouponValueCent: c.ValueCent,
		RedeemedAmountCent: totalCent, UnusedAmountCent: c.ValueCent - totalCent,
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
		CouponType:       c.CouponType,
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
		ID:            c.RedemptionID,
		EntitlementID: c.EntitlementID,
		EntitlementNo: c.EntitlementNo,
		TemplateID:    c.TemplateID,
		Name:          c.Name,
		Description:   c.Description,
		CouponType:    c.CouponType,
		ValueCent:     c.ValueCent,
		StoreID:       c.StoreID,
		Status:        c.Status,
		ExpiresAt:     c.ExpiresAt,
		CreatedAt:     c.CreatedAt,
	}
}
