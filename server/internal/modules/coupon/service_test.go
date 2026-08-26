package coupon

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/modules/catalog"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

type memRepo struct {
	byMember    map[int64][]MemberCoupon
	byActivity  map[int64][]MemberCoupon
	redemptions map[int64][]RedemptionOrder
	categories  []CouponCategory
	lastRedeem  *RedeemInput
	activityID  int64
}

func (r *memRepo) ListActiveCategories(_ context.Context) ([]CouponCategory, error) {
	return r.categories, nil
}

type fakeRedeemableCatalog struct {
	items []catalog.ItemView
}

func (f fakeRedeemableCatalog) ListCouponRedeemableItems(_ context.Context, _, _, _ int64) ([]catalog.ItemView, error) {
	return f.items, nil
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

func (r *memRepo) ListActivityUsableCoupons(_ context.Context, _ int64, activityID int64, _ time.Time, _ string, _, _ int) ([]MemberCoupon, int64, error) {
	r.activityID = activityID
	out := r.byActivity[activityID]
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
	r.lastRedeem = &in
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

func TestListCouponCategoriesReturnsOnlyRepositoryManagedCategories(t *testing.T) {
	repo := &memRepo{categories: []CouponCategory{
		{ID: 8, Name: "餐饮福利", BusinessType: TypeMeal, SortOrder: 10, Status: CategoryStatusActive},
	}}
	views, err := NewService(repo).ListCouponCategories(context.Background())
	if err != nil {
		t.Fatalf("list coupon categories: %v", err)
	}
	if len(views) != 1 || views[0].ID != 8 || views[0].Name != "餐饮福利" || views[0].BusinessType != TypeMeal {
		t.Fatalf("unexpected coupon categories: %+v", views)
	}
}

func TestListCouponsFiltersByMemberAndStatus(t *testing.T) {
	expiresAt := time.Date(2026, time.September, 11, 18, 46, 5, 0, time.UTC)
	repo := &memRepo{byMember: map[int64][]MemberCoupon{
		10: {
			{EntitlementID: 1, TemplateID: 100, Name: "Free Latte", CouponType: TypeBeverage, Status: StatusActive, ExpiresAt: &expiresAt},
			{EntitlementID: 2, TemplateID: 101, Name: "Old", CouponType: TypeSnack, Status: StatusExpired},
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
	if active[0].ExpiresAt != "2026-09-12 02:46:05" {
		t.Fatalf("unexpected expiresAt format: %q", active[0].ExpiresAt)
	}
}

func TestListActivityUsableCouponsUsesActivityFilter(t *testing.T) {
	repo := &memRepo{byActivity: map[int64][]MemberCoupon{
		8: {{EntitlementID: 3, Name: "当前活动单人票券", CouponType: TypeEventTicket, AdmissionCount: 1, Status: StatusActive}},
	}}
	svc := NewService(repo)

	views, total, err := svc.ListActivityUsableCoupons(context.Background(), 10, 8, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list activity coupons: %v", err)
	}
	if repo.activityID != 8 || total != 1 || len(views) != 1 || views[0].EntitlementID != 3 {
		t.Fatalf("unexpected activity coupon result: activity=%d total=%d views=%+v", repo.activityID, total, views)
	}
	if _, _, err := svc.ListActivityUsableCoupons(context.Background(), 10, 0, httpx.Page{Page: 1, PageSize: 20}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("invalid activity id error = %v", err)
	}
}

func TestRedeemActiveMarksUsed(t *testing.T) {
	repo := &memRepo{byMember: map[int64][]MemberCoupon{
		10: {{EntitlementID: 1, CouponType: TypeBeverage, Status: StatusActive}},
	}}
	svc := NewService(repo, fakeRedeemableCatalog{items: []catalog.ItemView{
		{ID: 8, Name: "Latte", PriceCent: 5000, StockQuantity: 0},
	}})

	view, err := svc.Redeem(context.Background(), 10, "idem-1", RedeemRequest{
		EntitlementID: 1, StoreID: 5,
		Items: []RedeemItemRequest{{ItemID: 8, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if view.Status != StatusUsed {
		t.Fatalf("expected used status, got %s", view.Status)
	}
	var rule RedemptionRuleSnapshot
	if err := json.Unmarshal(repo.lastRedeem.MatchedRuleJSON, &rule); err != nil {
		t.Fatalf("decode rule snapshot: %v", err)
	}
	if rule.RedeemedAmountCent != 5000 {
		t.Fatalf("unexpected redemption snapshot: %+v", rule)
	}
}

func TestRedeemRejectsInvalidIdentifiers(t *testing.T) {
	svc := NewService(&memRepo{})
	if _, err := svc.Redeem(context.Background(), 10, "idem-invalid", RedeemRequest{}); codeOf(t, err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid identifiers rejection, got %v", err)
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

func TestRedeemRejectsMultipleItemsPerCoupon(t *testing.T) {
	repo := &memRepo{byMember: map[int64][]MemberCoupon{
		10: {{EntitlementID: 1, CouponType: TypeBeverage, Status: StatusActive}},
	}}
	svc := NewService(repo, fakeRedeemableCatalog{items: []catalog.ItemView{
		{ID: 8, Name: "Latte", PriceCent: 6000},
	}})
	_, err := svc.Redeem(context.Background(), 10, "idem-over", RedeemRequest{
		EntitlementID: 1, StoreID: 5,
		Items: []RedeemItemRequest{{ItemID: 8, Quantity: 2}},
	})
	if codeOf(t, err) != apperr.CodeInvalidArgument {
		t.Fatalf("expected quantity rejection, got %v", err)
	}
}

func TestListEligibleItemsUsesCouponTemplate(t *testing.T) {
	repo := &memRepo{byMember: map[int64][]MemberCoupon{
		10: {{EntitlementID: 1, Name: "饮料券", CouponType: TypeBeverage, Status: StatusActive}},
	}}
	svc := NewService(repo, fakeRedeemableCatalog{items: []catalog.ItemView{
		{ID: 8, Name: "Latte", PriceCent: 5000},
	}})
	view, err := svc.ListEligibleItems(context.Background(), 10, 1, 5)
	if err != nil {
		t.Fatalf("list eligible items: %v", err)
	}
	if view.Coupon.CouponType != TypeBeverage || len(view.Items) != 1 || view.Items[0].ItemID != 8 {
		t.Fatalf("unexpected eligible items: %+v", view)
	}
}
