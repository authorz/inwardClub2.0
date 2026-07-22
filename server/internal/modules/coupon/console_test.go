package coupon

import (
	"context"
	"fmt"
	"testing"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// fakeConsoleRepo is an in-memory stand-in for the console repository. It
// reproduces the production scope, stock, per-member-limit and double-redeem
// semantics so the service contract for the write actions is exercised
// without a database.
type fakeConsoleRepo struct {
	templates []Template
	scopes    []ConsoleScope
	applic    ApplicableScope

	nextTmplID int64
	ents       []*fakeEnt
	nextEntID  int64
}

type fakeEnt struct {
	id       int64
	no       string
	tmplID   int64
	memberID int64
	storeID  *int64
	status   string
	redeemed bool
}

func (r *fakeConsoleRepo) allocTmplID() int64 {
	if r.nextTmplID == 0 {
		r.nextTmplID = 1
		for _, t := range r.templates {
			if t.ID >= r.nextTmplID {
				r.nextTmplID = t.ID + 1
			}
		}
	}
	id := r.nextTmplID
	r.nextTmplID++
	return id
}

// templateIndex returns the slice index of a template visible within scope, or
// -1 when it is absent or out of scope.
func (r *fakeConsoleRepo) templateIndex(scope ConsoleScope, id int64) int {
	for i := range r.templates {
		t := r.templates[i]
		if t.ID != id {
			continue
		}
		if scope.StoreID != nil && (t.ScopeType != "store" || t.StoreID == nil || *t.StoreID != *scope.StoreID) {
			return -1
		}
		return i
	}
	return -1
}

func (r *fakeConsoleRepo) ListTemplates(_ context.Context, scope ConsoleScope, page httpx.Page) ([]Template, int64, error) {
	r.scopes = append(r.scopes, scope)
	var out []Template
	for _, t := range r.templates {
		if scope.StoreID != nil {
			if t.ScopeType != "store" || t.StoreID == nil || *t.StoreID != *scope.StoreID {
				continue
			}
		}
		out = append(out, t)
	}
	return out, int64(len(out)), nil
}

func (r *fakeConsoleRepo) GetTemplate(_ context.Context, scope ConsoleScope, id int64) (Template, error) {
	r.scopes = append(r.scopes, scope)
	if i := r.templateIndex(scope, id); i >= 0 {
		return r.templates[i], nil
	}
	return Template{}, apperr.NotFound("coupon template not found")
}

func (r *fakeConsoleRepo) CreateTemplate(_ context.Context, scope ConsoleScope, in TemplateInput) (Template, error) {
	if !validCouponType(in.CouponType) {
		return Template{}, apperr.Invalid("invalid couponType")
	}
	t := Template{
		ID:           r.allocTmplID(),
		ScopeType:    "global",
		Name:         in.Name,
		Description:  in.Description,
		CouponType:   in.CouponType,
		ValueCent:    in.ValueCent,
		PointsPrice:  in.PointsPrice,
		StockQty:     in.StockQty,
		PerMemberLim: in.PerMemberLim,
		Status:       "draft",
	}
	if scope.StoreID != nil {
		t.ScopeType = "store"
		sid := *scope.StoreID
		t.StoreID = &sid
	}
	r.templates = append(r.templates, t)
	return t, nil
}

func (r *fakeConsoleRepo) UpdateTemplate(_ context.Context, scope ConsoleScope, id int64, in TemplateInput) (Template, error) {
	if !validCouponType(in.CouponType) {
		return Template{}, apperr.Invalid("invalid couponType")
	}
	i := r.templateIndex(scope, id)
	if i < 0 {
		return Template{}, apperr.NotFound("coupon template not found")
	}
	t := &r.templates[i]
	t.Name = in.Name
	t.Description = in.Description
	t.CouponType = in.CouponType
	t.ValueCent = in.ValueCent
	t.PointsPrice = in.PointsPrice
	t.StockQty = in.StockQty
	t.PerMemberLim = in.PerMemberLim
	return *t, nil
}

func (r *fakeConsoleRepo) DeleteTemplate(_ context.Context, scope ConsoleScope, id int64) error {
	i := r.templateIndex(scope, id)
	if i < 0 {
		return apperr.NotFound("coupon template not found")
	}
	r.templates = append(r.templates[:i], r.templates[i+1:]...)
	return nil
}

func (r *fakeConsoleRepo) GetApplicableScope(_ context.Context, scope ConsoleScope, _ int64) (ApplicableScope, error) {
	r.scopes = append(r.scopes, scope)
	return r.applic, nil
}

func (r *fakeConsoleRepo) Grant(_ context.Context, scope ConsoleScope, req GrantRequest) (EntitlementView, error) {
	i := r.templateIndex(scope, req.TemplateID)
	if i < 0 {
		return EntitlementView{}, apperr.NotFound("coupon template not found")
	}
	t := &r.templates[i]
	if t.StockQty > 0 && t.IssuedQty >= t.StockQty {
		return EntitlementView{}, apperr.Conflict("coupon template is out of stock")
	}
	if t.PerMemberLim > 0 {
		var held int64
		for _, e := range r.ents {
			if e.tmplID == req.TemplateID && e.memberID == req.MemberID &&
				(e.status == StatusActive || e.status == StatusUsed) {
				held++
			}
		}
		if held >= t.PerMemberLim {
			return EntitlementView{}, apperr.Conflict("member has reached the coupon limit")
		}
	}
	r.nextEntID++
	ent := &fakeEnt{
		id:       r.nextEntID,
		no:       fmt.Sprintf("E%d-%d", req.TemplateID, r.nextEntID),
		tmplID:   req.TemplateID,
		memberID: req.MemberID,
		storeID:  t.StoreID,
		status:   StatusActive,
	}
	r.ents = append(r.ents, ent)
	t.IssuedQty++
	return EntitlementView{EntitlementID: ent.id, EntitlementNo: ent.no, Status: StatusActive}, nil
}

func (r *fakeConsoleRepo) entByID(id int64) *fakeEnt {
	for _, e := range r.ents {
		if e.id == id {
			return e
		}
	}
	return nil
}

func (r *fakeConsoleRepo) Void(_ context.Context, scope ConsoleScope, req VoidRequest) (EntitlementView, error) {
	e := r.entByID(req.EntitlementID)
	if e == nil {
		return EntitlementView{}, apperr.NotFound("coupon entitlement not found")
	}
	if scope.StoreID != nil && (e.storeID == nil || *e.storeID != *scope.StoreID) {
		return EntitlementView{}, apperr.NotFound("coupon entitlement not found")
	}
	if e.status != StatusActive {
		return EntitlementView{}, apperr.Conflict("coupon entitlement is not active")
	}
	e.status = StatusVoid
	return EntitlementView{EntitlementID: e.id, EntitlementNo: e.no, Status: StatusVoid}, nil
}

func (r *fakeConsoleRepo) Verify(_ context.Context, scope ConsoleScope, req VerifyRequest) (EntitlementView, error) {
	if req.EntitlementID == 0 && req.EntitlementNo == "" {
		return EntitlementView{}, apperr.Invalid("entitlementId or entitlementNo is required")
	}
	var e *fakeEnt
	for _, cand := range r.ents {
		if (req.EntitlementID != 0 && cand.id == req.EntitlementID) ||
			(req.EntitlementNo != "" && cand.no == req.EntitlementNo) {
			e = cand
			break
		}
	}
	if e == nil {
		return EntitlementView{}, apperr.NotFound("coupon entitlement not found")
	}
	if scope.StoreID != nil && e.storeID != nil && *e.storeID != *scope.StoreID {
		return EntitlementView{}, apperr.NotFound("coupon entitlement not found")
	}
	if e.redeemed {
		return EntitlementView{}, apperr.Conflict("coupon already redeemed")
	}
	if e.status != StatusActive {
		return EntitlementView{}, apperr.Conflict("coupon entitlement is not redeemable")
	}
	e.redeemed = true
	e.status = StatusUsed
	return EntitlementView{EntitlementID: e.id, EntitlementNo: e.no, Status: StatusUsed}, nil
}

func storeIDPtr(id int64) *int64 { return &id }

func TestConsoleListTemplatesScoping(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 1, ScopeType: "global", Name: "Global Coupon"},
		{ID: 2, ScopeType: "store", StoreID: storeIDPtr(5), Name: "Store 5 Coupon"},
		{ID: 3, ScopeType: "store", StoreID: storeIDPtr(6), Name: "Store 6 Coupon"},
	}}
	svc := NewConsoleService(repo)
	page := httpx.Page{Page: 1, PageSize: 20}

	admin, total, err := svc.ListTemplates(context.Background(), ConsoleScope{}, page)
	if err != nil {
		t.Fatalf("admin list: %v", err)
	}
	if total != 3 || len(admin) != 3 {
		t.Fatalf("expected admin to see all 3 templates, got %d", total)
	}

	store, total, err := svc.ListTemplates(context.Background(), ConsoleScope{StoreID: storeIDPtr(5)}, page)
	if err != nil {
		t.Fatalf("store list: %v", err)
	}
	if total != 1 || store[0].Name != "Store 5 Coupon" {
		t.Fatalf("expected store scope to see only its own template, got %+v", store)
	}

	if len(repo.scopes) != 2 || repo.scopes[1].StoreID == nil || *repo.scopes[1].StoreID != 5 {
		t.Fatalf("expected scope propagated to repo, got %+v", repo.scopes)
	}
}

func TestConsoleGetTemplateMapping(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 1, ScopeType: "global", Name: "Global Coupon", CouponType: TypeExchange, ValueCent: 500, StockQty: 10, IssuedQty: 2, Status: "draft"},
	}}
	svc := NewConsoleService(repo)

	view, err := svc.GetTemplate(context.Background(), ConsoleScope{}, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.Name != "Global Coupon" || view.CouponType != TypeExchange || view.ValueCent != 500 || view.StockQty != 10 || view.IssuedQty != 2 || view.Status != "draft" {
		t.Fatalf("unexpected view mapping: %+v", view)
	}
}

func TestConsoleGetTemplateForeignStoreNotFound(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 2, ScopeType: "store", StoreID: storeIDPtr(5), Name: "Store 5 Coupon"},
	}}
	svc := NewConsoleService(repo)

	if _, err := svc.GetTemplate(context.Background(), ConsoleScope{StoreID: storeIDPtr(6)}, 2); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for foreign store scope, got %v", err)
	}
}

func TestConsoleApplicableItemsMapping(t *testing.T) {
	repo := &fakeConsoleRepo{
		templates: []Template{{ID: 1, ScopeType: "global"}},
		applic:    ApplicableScope{ItemIDs: []int64{10, 11}, CategoryIDs: []int64{3}},
	}
	svc := NewConsoleService(repo)

	view, err := svc.GetApplicableItems(context.Background(), ConsoleScope{}, 1)
	if err != nil {
		t.Fatalf("applicable items: %v", err)
	}
	if view.TemplateID != 1 || len(view.ItemIDs) != 2 || len(view.CategoryIDs) != 1 {
		t.Fatalf("unexpected applicable items view: %+v", view)
	}
}

// Create/update/delete round-trips within scope, and cross-scope writes are
// invisible (NOT_FOUND).
func TestConsoleTemplateCRUD(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo)
	ctx := context.Background()

	// Admin creates a global template.
	created, err := svc.CreateTemplate(ctx, ConsoleScope{}, TemplateInput{Name: "G", CouponType: TypeCash, ValueCent: 100})
	if err != nil {
		t.Fatalf("admin create: %v", err)
	}
	if created.ScopeType != "global" || created.StoreID != nil || created.Status != "draft" {
		t.Fatalf("unexpected global template: %+v", created)
	}

	// Store creates a store-scoped template.
	store5 := ConsoleScope{StoreID: storeIDPtr(5)}
	sc, err := svc.CreateTemplate(ctx, store5, TemplateInput{Name: "S", CouponType: TypeDiscount})
	if err != nil {
		t.Fatalf("store create: %v", err)
	}
	if sc.ScopeType != "store" || sc.StoreID == nil || *sc.StoreID != 5 {
		t.Fatalf("expected store scope binding, got %+v", sc)
	}

	// Update within scope.
	up, err := svc.UpdateTemplate(ctx, store5, sc.ID, TemplateInput{Name: "S2", CouponType: TypeCash, ValueCent: 42})
	if err != nil {
		t.Fatalf("store update: %v", err)
	}
	if up.Name != "S2" || up.ValueCent != 42 {
		t.Fatalf("update not applied: %+v", up)
	}

	// A foreign store cannot update or delete the store template.
	if _, err := svc.UpdateTemplate(ctx, ConsoleScope{StoreID: storeIDPtr(6)}, sc.ID, TemplateInput{Name: "x", CouponType: TypeCash}); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected foreign update NOT_FOUND, got %v", err)
	}
	if err := svc.DeleteTemplate(ctx, ConsoleScope{StoreID: storeIDPtr(6)}, sc.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected foreign delete NOT_FOUND, got %v", err)
	}

	// Owner deletes; then it is gone.
	if err := svc.DeleteTemplate(ctx, store5, sc.ID); err != nil {
		t.Fatalf("store delete: %v", err)
	}
	if _, err := svc.GetTemplate(ctx, store5, sc.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected deleted template NOT_FOUND, got %v", err)
	}

	// Invalid coupon type is rejected.
	if _, err := svc.CreateTemplate(ctx, ConsoleScope{}, TemplateInput{Name: "bad", CouponType: "bogus"}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid couponType INVALID_ARGUMENT, got %v", err)
	}
}

// Grant enforces stock and per-member limit, and bumps issued_quantity.
func TestConsoleGrantStockAndLimit(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 1, ScopeType: "global", CouponType: TypeCash, StockQty: 2, PerMemberLim: 1},
	}}
	svc := NewConsoleService(repo)
	ctx := context.Background()

	if _, err := svc.Grant(ctx, ConsoleScope{}, GrantRequest{TemplateID: 1, MemberID: 100}); err != nil {
		t.Fatalf("first grant: %v", err)
	}
	if repo.templates[0].IssuedQty != 1 {
		t.Fatalf("expected issued_quantity 1, got %d", repo.templates[0].IssuedQty)
	}

	// Per-member limit reached.
	if _, err := svc.Grant(ctx, ConsoleScope{}, GrantRequest{TemplateID: 1, MemberID: 100}); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected per-member CONFLICT, got %v", err)
	}

	// Second member consumes the last unit.
	if _, err := svc.Grant(ctx, ConsoleScope{}, GrantRequest{TemplateID: 1, MemberID: 101}); err != nil {
		t.Fatalf("second member grant: %v", err)
	}
	if repo.templates[0].IssuedQty != 2 {
		t.Fatalf("expected issued_quantity 2, got %d", repo.templates[0].IssuedQty)
	}

	// Out of stock.
	if _, err := svc.Grant(ctx, ConsoleScope{}, GrantRequest{TemplateID: 1, MemberID: 102}); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected out-of-stock CONFLICT, got %v", err)
	}
}

// A store may only grant from a template it can see.
func TestConsoleGrantScopeBoundary(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 1, ScopeType: "store", StoreID: storeIDPtr(5), CouponType: TypeCash},
	}}
	svc := NewConsoleService(repo)
	ctx := context.Background()

	if _, err := svc.Grant(ctx, ConsoleScope{StoreID: storeIDPtr(6)}, GrantRequest{TemplateID: 1, MemberID: 100}); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected foreign-store grant NOT_FOUND, got %v", err)
	}
	if _, err := svc.Grant(ctx, ConsoleScope{StoreID: storeIDPtr(5)}, GrantRequest{TemplateID: 1, MemberID: 100}); err != nil {
		t.Fatalf("owning-store grant: %v", err)
	}
}

// Void flips an active entitlement, rejects a second void, and hides foreign
// entitlements from a store console.
func TestConsoleVoid(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 1, ScopeType: "store", StoreID: storeIDPtr(5), CouponType: TypeCash},
	}}
	svc := NewConsoleService(repo)
	ctx := context.Background()
	store5 := ConsoleScope{StoreID: storeIDPtr(5)}

	granted, err := svc.Grant(ctx, store5, GrantRequest{TemplateID: 1, MemberID: 100})
	if err != nil {
		t.Fatalf("grant: %v", err)
	}

	// A foreign store cannot void it.
	if _, err := svc.Void(ctx, ConsoleScope{StoreID: storeIDPtr(6)}, VoidRequest{EntitlementID: granted.EntitlementID}); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected foreign void NOT_FOUND, got %v", err)
	}

	voided, err := svc.Void(ctx, store5, VoidRequest{EntitlementID: granted.EntitlementID})
	if err != nil {
		t.Fatalf("void: %v", err)
	}
	if voided.Status != StatusVoid {
		t.Fatalf("expected void status, got %s", voided.Status)
	}

	// Voiding again conflicts (no longer active).
	if _, err := svc.Void(ctx, store5, VoidRequest{EntitlementID: granted.EntitlementID}); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected re-void CONFLICT, got %v", err)
	}
}

// Verify redeems once and refuses a second redemption of the same entitlement.
func TestConsoleVerifyNoDoubleRedeem(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 1, ScopeType: "global", CouponType: TypeCash},
	}}
	svc := NewConsoleService(repo)
	ctx := context.Background()

	granted, err := svc.Grant(ctx, ConsoleScope{}, GrantRequest{TemplateID: 1, MemberID: 100})
	if err != nil {
		t.Fatalf("grant: %v", err)
	}

	used, err := svc.Verify(ctx, ConsoleScope{}, VerifyRequest{EntitlementID: granted.EntitlementID, StoreID: 5})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if used.Status != StatusUsed {
		t.Fatalf("expected used status, got %s", used.Status)
	}

	// Second redemption is rejected.
	if _, err := svc.Verify(ctx, ConsoleScope{}, VerifyRequest{EntitlementID: granted.EntitlementID, StoreID: 5}); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected double-redeem CONFLICT, got %v", err)
	}
}

// A store console cannot redeem an entitlement bound to another store, and a
// request that names no entitlement is rejected.
func TestConsoleVerifyScopeAndIdentifier(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 1, ScopeType: "store", StoreID: storeIDPtr(5), CouponType: TypeCash},
	}}
	svc := NewConsoleService(repo)
	ctx := context.Background()

	granted, err := svc.Grant(ctx, ConsoleScope{StoreID: storeIDPtr(5)}, GrantRequest{TemplateID: 1, MemberID: 100})
	if err != nil {
		t.Fatalf("grant: %v", err)
	}

	if _, err := svc.Verify(ctx, ConsoleScope{StoreID: storeIDPtr(6)}, VerifyRequest{EntitlementID: granted.EntitlementID, StoreID: 6}); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected foreign-store verify NOT_FOUND, got %v", err)
	}
	if _, err := svc.Verify(ctx, ConsoleScope{}, VerifyRequest{StoreID: 5}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected missing-identifier INVALID_ARGUMENT, got %v", err)
	}
}

// validCouponType guards the template writes against unknown coupon types.
func TestValidCouponType(t *testing.T) {
	for _, ok := range []string{TypeExchange, TypeDiscount, TypeCash} {
		if !validCouponType(ok) {
			t.Fatalf("expected %q to be valid", ok)
		}
	}
	if validCouponType("bogus") {
		t.Fatalf("expected bogus coupon type to be rejected")
	}
}

// Verify rejects a request that identifies no entitlement before touching the DB.
func TestSQLConsoleRepositoryVerifyRequiresIdentifier(t *testing.T) {
	repo := &sqlConsoleRepository{}
	if _, err := repo.Verify(context.Background(), ConsoleScope{}, VerifyRequest{StoreID: 1}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}
