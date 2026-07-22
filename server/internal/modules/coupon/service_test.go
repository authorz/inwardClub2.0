package coupon

import (
	"context"
	"testing"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

type memRepo struct {
	byMember    map[int64][]MemberCoupon
	redemptions map[int64][]RedemptionOrder
}

func (r *memRepo) ListMemberCoupons(_ context.Context, memberID int64, status string, _, _ int) ([]MemberCoupon, int64, error) {
	var out []MemberCoupon
	for _, c := range r.byMember[memberID] {
		if status == "" || c.Status == status {
			out = append(out, c)
		}
	}
	return out, int64(len(out)), nil
}

func (r *memRepo) GetEntitlement(_ context.Context, memberID, entitlementID int64) (MemberCoupon, error) {
	for _, c := range r.byMember[memberID] {
		if c.EntitlementID == entitlementID {
			return c, nil
		}
	}
	return MemberCoupon{}, apperr.NotFound("coupon not found")
}

// Redeem is exercised end-to-end against MySQL; the in-memory repo flips the
// entitlement to used and echoes it back so the service wiring can be tested.
func (r *memRepo) Redeem(_ context.Context, in RedeemInput) (MemberCoupon, error) {
	for i, c := range r.byMember[in.MemberID] {
		if c.EntitlementID == in.EntitlementID {
			r.byMember[in.MemberID][i].Status = StatusUsed
			return r.byMember[in.MemberID][i], nil
		}
	}
	return MemberCoupon{}, apperr.NotFound("coupon not found")
}

func (r *memRepo) ListRedemptions(_ context.Context, memberID int64, _, _ int) ([]RedemptionOrder, int64, error) {
	out := r.redemptions[memberID]
	return out, int64(len(out)), nil
}

func (r *memRepo) GetRedemption(_ context.Context, memberID, id int64) (RedemptionOrder, error) {
	for _, o := range r.redemptions[memberID] {
		if o.ID == id {
			return o, nil
		}
	}
	return RedemptionOrder{}, apperr.NotFound("redemption order not found")
}

func codeOf(t *testing.T, err error) apperr.Code {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	return apperr.From(err).Code
}

func TestListCouponsFiltersByMemberAndStatus(t *testing.T) {
	repo := &memRepo{byMember: map[int64][]MemberCoupon{
		10: {
			{EntitlementID: 1, TemplateID: 100, Name: "Free Latte", CouponType: TypeExchange, Status: StatusActive},
			{EntitlementID: 2, TemplateID: 101, Name: "Old", CouponType: TypeCash, Status: StatusExpired},
		},
	}}
	svc := NewService(repo)

	all, total, err := svc.ListCoupons(context.Background(), 10, "", httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(all) != 2 {
		t.Fatalf("expected 2 coupons, got %d", total)
	}
	active, total, err := svc.ListCoupons(context.Background(), 10, StatusActive, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if total != 1 || active[0].Name != "Free Latte" {
		t.Fatalf("unexpected active coupons: %+v", active)
	}
}

func TestRedeemActiveMarksUsed(t *testing.T) {
	repo := &memRepo{byMember: map[int64][]MemberCoupon{
		10: {{EntitlementID: 1, Status: StatusActive}},
	}}
	svc := NewService(repo)

	view, err := svc.Redeem(context.Background(), 10, "idem-1", RedeemRequest{EntitlementID: 1, StoreID: 5})
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if view.Status != StatusUsed {
		t.Fatalf("expected used status, got %s", view.Status)
	}
}

func TestListAndGetRedemptions(t *testing.T) {
	repo := &memRepo{redemptions: map[int64][]RedemptionOrder{
		10: {
			{ID: 1, RedemptionNo: "RD001", Status: StatusUsed, Title: "Free Latte", CouponName: "Free Latte", Qty: 1, Code: "RD001", StoreName: "Downtown"},
		},
	}}
	svc := NewService(repo)
	ctx := context.Background()

	list, total, err := svc.ListRedemptions(ctx, 10, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list redemptions: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].OrderNo != "RD001" || list[0].Code != "RD001" {
		t.Fatalf("unexpected redemption list: %+v", list)
	}

	one, err := svc.GetRedemption(ctx, 10, 1)
	if err != nil {
		t.Fatalf("get redemption: %v", err)
	}
	if one.CouponName != "Free Latte" || one.Qty != 1 || one.StoreName != "Downtown" {
		t.Fatalf("unexpected redemption: %+v", one)
	}

	// Another member's redemption is not found.
	if _, err := svc.GetRedemption(ctx, 99, 1); codeOf(t, err) != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for foreign redemption")
	}
}

func TestRedeemRejectsNonActiveAndForeign(t *testing.T) {
	repo := &memRepo{byMember: map[int64][]MemberCoupon{
		10: {{EntitlementID: 1, Status: StatusUsed}},
	}}
	svc := NewService(repo)
	ctx := context.Background()

	// Already-used coupon is a conflict.
	if _, err := svc.Redeem(ctx, 10, "idem-a", RedeemRequest{EntitlementID: 1, StoreID: 5}); codeOf(t, err) != apperr.CodeConflict {
		t.Fatalf("expected CONFLICT for used coupon")
	}
	// Another member's entitlement is not found.
	if _, err := svc.Redeem(ctx, 99, "idem-b", RedeemRequest{EntitlementID: 1, StoreID: 5}); codeOf(t, err) != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for foreign coupon")
	}
}
