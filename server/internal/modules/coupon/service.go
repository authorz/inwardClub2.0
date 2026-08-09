package coupon

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Service provides the mini-program coupon read path and redemption entry point.
type Service struct {
	repo Repository
}

// NewService builds the coupon service.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

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

// Redeem records a redemption for a member-held entitlement. It pre-checks
// ownership/state for a friendly error, then records the redemption and flips
// the entitlement to used in one transaction (the repository re-validates under
// a row lock and the unique redemption index is the authoritative guard). idemKey
// is the request's Idempotency-Key. Rule matching / item snapshotting is layered
// on later; this records the hit and consumes the entitlement.
func (s *Service) Redeem(ctx context.Context, memberID int64, idemKey string, req RedeemRequest) (MemberCouponView, error) {
	if req.EntitlementID <= 0 || req.StoreID <= 0 {
		return MemberCouponView{}, apperr.Invalid("优惠券和门店信息不正确")
	}
	c, err := s.repo.GetEntitlement(ctx, memberID, req.EntitlementID)
	if err != nil {
		return MemberCouponView{}, err
	}
	if c.Status != StatusActive {
		return MemberCouponView{}, apperr.Conflict("coupon is not redeemable")
	}
	now := time.Now().UTC()
	redeemed, err := s.repo.Redeem(ctx, RedeemInput{
		MemberID:      memberID,
		EntitlementID: req.EntitlementID,
		StoreID:       req.StoreID,
		RedemptionNo:  fmt.Sprintf("RD%s%04d", now.Format("20060102150405"), rand.Intn(10000)),
		IdemKey:       idemKey,
		Now:           now,
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
		EntitlementID: c.EntitlementID,
		EntitlementNo: c.EntitlementNo,
		TemplateID:    c.TemplateID,
		Name:          c.Name,
		Description:   c.Description,
		CouponType:    c.CouponType,
		ValueCent:     c.ValueCent,
		Status:        c.Status,
		ExpiresAt:     c.ExpiresAt,
		CreatedAt:     c.CreatedAt,
	}
}
