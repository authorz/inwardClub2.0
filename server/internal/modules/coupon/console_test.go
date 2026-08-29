package coupon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/inwardclub/server/internal/platform/audit"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// fakeConsoleRepo is an in-memory stand-in for the console repository. It
// reproduces the production scope, stock, per-member-limit and double-redeem
// semantics so the service contract for the write actions is exercised
// without a database.
type fakeConsoleRepo struct {
	templates  []Template
	categories []CouponCategory
	scopes     []ConsoleScope
	applic     ApplicableScope

	nextTmplID int64
	ents       []*fakeEnt
	nextEntID  int64
}

func (r *fakeConsoleRepo) category(id int64) (CouponCategory, bool) {
	for _, category := range r.categories {
		if category.ID == id {
			return category, true
		}
	}
	return CouponCategory{}, false
}

type fakeEnt struct {
	id       int64
	no       string
	tmplID   int64
	memberID int64
	storeID  *int64
	status   string
	redeemed bool
	expires  *time.Time
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
	category, ok := r.category(in.CategoryID)
	if !ok || category.Status != CategoryStatusActive {
		return Template{}, apperr.Invalid("invalid categoryId")
	}
	t := Template{
		ID: r.allocTmplID(), ScopeType: "global", Name: in.Name,
		Description: in.Description, CategoryID: category.ID, CategoryName: category.Name,
		CouponType: category.BusinessType, Status: "draft",
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
	category, ok := r.category(in.CategoryID)
	if !ok {
		return Template{}, apperr.Invalid("invalid categoryId")
	}
	i := r.templateIndex(scope, id)
	if i < 0 {
		return Template{}, apperr.NotFound("coupon template not found")
	}
	t := &r.templates[i]
	t.Name = in.Name
	t.Description = in.Description
	t.CategoryID = category.ID
	t.CategoryName = category.Name
	t.CouponType = category.BusinessType
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

func (r *fakeConsoleRepo) SetTemplateStatus(_ context.Context, scope ConsoleScope, id int64, status string) (Template, error) {
	i := r.templateIndex(scope, id)
	if i < 0 {
		return Template{}, apperr.NotFound("coupon template not found")
	}
	r.templates[i].Status = status
	return r.templates[i], nil
}

func TestConsoleServicePublishesAndDisablesTemplate(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{{ID: 3, Name: "充值赠券", Status: "draft"}}}
	svc := NewConsoleService(repo)
	published, err := svc.PublishTemplate(context.Background(), ConsoleScope{}, 3)
	if err != nil {
		t.Fatalf("publish template: %v", err)
	}
	if published.Status != "published" {
		t.Fatalf("published status = %q", published.Status)
	}
	disabled, err := svc.DisableTemplate(context.Background(), ConsoleScope{}, 3)
	if err != nil {
		t.Fatalf("disable template: %v", err)
	}
	if disabled.Status != "disabled" {
		t.Fatalf("disabled status = %q", disabled.Status)
	}
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
		expires:  req.ExpiresAt,
	}
	r.ents = append(r.ents, ent)
	t.IssuedQty++
	return EntitlementView{EntitlementID: ent.id, EntitlementNo: ent.no, Status: StatusActive}, nil
}

func (r *fakeConsoleRepo) ListMemberEntitlements(_ context.Context, scope ConsoleScope, memberID int64, _ httpx.Page) ([]ConsoleEntitlementView, int64, error) {
	var out []ConsoleEntitlementView
	for _, e := range r.ents {
		if e.memberID != memberID {
			continue
		}
		if scope.StoreID != nil && e.storeID != nil && *e.storeID != *scope.StoreID {
			continue
		}
		view := ConsoleEntitlementView{
			EntitlementID: e.id, EntitlementNo: e.no, TemplateID: e.tmplID,
			MemberID: e.memberID, StoreID: e.storeID, Status: e.status, ExpiresAt: e.expires,
		}
		if i := r.templateIndex(ConsoleScope{}, e.tmplID); i >= 0 {
			view.TemplateName = r.templates[i].Name
			view.CouponType = r.templates[i].CouponType
		}
		out = append(out, view)
	}
	return out, int64(len(out)), nil
}

func (r *fakeConsoleRepo) GrantMemberEntitlement(_ context.Context, scope ConsoleScope, memberID int64, req GrantRequest, _ string, _ audit.Entry) (EntitlementView, error) {
	req.MemberID = memberID
	view, err := r.Grant(context.Background(), scope, req)
	if err == nil && len(r.ents) > 0 {
		r.ents[len(r.ents)-1].storeID = req.StoreID
	}
	return view, err
}

func (r *fakeConsoleRepo) UpdateMemberEntitlementExpiry(_ context.Context, memberID, entitlementID int64, req UpdateEntitlementExpiryRequest, _ string, _ audit.Entry) (EntitlementView, error) {
	e := r.entByID(entitlementID)
	if e == nil || e.memberID != memberID {
		return EntitlementView{}, apperr.NotFound("coupon entitlement not found")
	}
	if e.status != StatusActive && e.status != StatusExpired {
		return EntitlementView{}, apperr.Conflict("coupon entitlement cannot be updated")
	}
	expires := req.ExpiresAt.UTC()
	e.expires = &expires
	e.status = StatusActive
	return EntitlementView{EntitlementID: e.id, EntitlementNo: e.no, Status: e.status}, nil
}

func (r *fakeConsoleRepo) VoidMemberEntitlement(_ context.Context, memberID, entitlementID int64, _ string, _ string, _ audit.Entry) (EntitlementView, error) {
	e := r.entByID(entitlementID)
	if e == nil || e.memberID != memberID {
		return EntitlementView{}, apperr.NotFound("coupon entitlement not found")
	}
	if e.status != StatusActive && e.status != StatusExpired {
		return EntitlementView{}, apperr.Conflict("coupon entitlement cannot be voided")
	}
	e.status = StatusVoid
	return EntitlementView{EntitlementID: e.id, EntitlementNo: e.no, Status: e.status}, nil
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
		{ID: 1, ScopeType: "global", Name: "Global Coupon", CouponType: TypeSnack, Status: "draft"},
	}}
	svc := NewConsoleService(repo)

	view, err := svc.GetTemplate(context.Background(), ConsoleScope{}, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.Name != "Global Coupon" || view.CouponType != TypeSnack || view.Status != "draft" {
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
	repo := &fakeConsoleRepo{categories: []CouponCategory{
		{ID: 1, Name: "酒水券", BusinessType: TypeAlcohol, Status: CategoryStatusActive},
		{ID: 2, Name: "饮料券", BusinessType: TypeBeverage, Status: CategoryStatusActive},
	}}
	svc := NewConsoleService(repo)
	ctx := context.Background()

	// Admin creates a global template.
	created, err := svc.CreateTemplate(ctx, ConsoleScope{}, TemplateInput{Name: "G", CategoryID: 1})
	if err != nil {
		t.Fatalf("admin create: %v", err)
	}
	if created.ScopeType != "global" || created.StoreID != nil || created.Status != "draft" {
		t.Fatalf("unexpected global template: %+v", created)
	}

	// Store creates a store-scoped template.
	store5 := ConsoleScope{StoreID: storeIDPtr(5)}
	sc, err := svc.CreateTemplate(ctx, store5, TemplateInput{Name: "S", CategoryID: 2})
	if err != nil {
		t.Fatalf("store create: %v", err)
	}
	if sc.ScopeType != "store" || sc.StoreID == nil || *sc.StoreID != 5 {
		t.Fatalf("expected store scope binding, got %+v", sc)
	}

	// Update within scope.
	up, err := svc.UpdateTemplate(ctx, store5, sc.ID, TemplateInput{Name: "S2", CategoryID: 1})
	if err != nil {
		t.Fatalf("store update: %v", err)
	}
	if up.Name != "S2" || up.CouponType != TypeAlcohol {
		t.Fatalf("update not applied: %+v", up)
	}

	// A foreign store cannot update or delete the store template.
	if _, err := svc.UpdateTemplate(ctx, ConsoleScope{StoreID: storeIDPtr(6)}, sc.ID, TemplateInput{Name: "x", CategoryID: 1}); apperr.From(err).Code != apperr.CodeNotFound {
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
	if _, err := svc.CreateTemplate(ctx, ConsoleScope{}, TemplateInput{Name: "bad", CategoryID: 99}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid categoryId INVALID_ARGUMENT, got %v", err)
	}
}

// Grant enforces stock and per-member limit, and bumps issued_quantity.
func TestConsoleGrantStockAndLimit(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 1, ScopeType: "global", CouponType: TypeAlcohol, StockQty: 2, PerMemberLim: 1},
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

func TestAdminMemberEntitlementLifecycle(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{{
		ID: 1, Name: "饮料券", CouponType: TypeBeverage, Status: "published",
	}}}
	svc := NewConsoleService(repo)
	ctx := context.Background()
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	granted, err := svc.GrantMemberEntitlement(ctx, ConsoleScope{}, 100, GrantRequest{
		TemplateID: 1, ScopeType: "global", ExpiresAt: &expiresAt, Reason: "客服补发",
	}, "idem-grant", audit.Entry{ActorID: 9})
	if err != nil {
		t.Fatalf("grant member entitlement: %v", err)
	}
	rows, total, err := svc.ListMemberEntitlements(ctx, ConsoleScope{}, 100, httpx.Page{Page: 1, PageSize: 20})
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Status != StatusActive {
		t.Fatalf("unexpected member entitlements: rows=%+v total=%d err=%v", rows, total, err)
	}

	newExpiry := expiresAt.Add(15 * 24 * time.Hour)
	updated, err := svc.UpdateMemberEntitlementExpiry(ctx, 100, granted.EntitlementID,
		UpdateEntitlementExpiryRequest{ExpiresAt: newExpiry, Reason: "延长有效期"},
		"idem-expiry", audit.Entry{ActorID: 9},
	)
	if err != nil || updated.Status != StatusActive || repo.ents[0].expires == nil || !repo.ents[0].expires.Equal(newExpiry) {
		t.Fatalf("unexpected expiry update: view=%+v entitlement=%+v err=%v", updated, repo.ents[0], err)
	}

	voided, err := svc.VoidMemberEntitlement(ctx, 100, granted.EntitlementID,
		"重复补发，删除旧券", "idem-void", audit.Entry{ActorID: 9},
	)
	if err != nil || voided.Status != StatusVoid {
		t.Fatalf("unexpected void result: view=%+v err=%v", voided, err)
	}
	if _, err := svc.UpdateMemberEntitlementExpiry(ctx, 100, granted.EntitlementID,
		UpdateEntitlementExpiryRequest{ExpiresAt: newExpiry, Reason: "不应成功"},
		"idem-after-void", audit.Entry{ActorID: 9},
	); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected voided entitlement update conflict, got %v", err)
	}
}

func TestResolveMemberGrantStore(t *testing.T) {
	global := Template{ID: 1, ScopeType: "global"}
	storeFive := Template{ID: 2, ScopeType: "store", StoreID: storeIDPtr(5)}

	if storeID, err := resolveMemberGrantStore(global, GrantRequest{ScopeType: "global"}); err != nil || storeID != nil {
		t.Fatalf("global template should support all stores: store=%v err=%v", storeID, err)
	}
	if storeID, err := resolveMemberGrantStore(global, GrantRequest{ScopeType: "store", StoreID: storeIDPtr(6)}); err != nil || storeID == nil || *storeID != 6 {
		t.Fatalf("global template should support one store: store=%v err=%v", storeID, err)
	}
	if storeID, err := resolveMemberGrantStore(storeFive, GrantRequest{ScopeType: "store", StoreID: storeIDPtr(5)}); err != nil || storeID == nil || *storeID != 5 {
		t.Fatalf("store template should support its own store: store=%v err=%v", storeID, err)
	}
	if _, err := resolveMemberGrantStore(storeFive, GrantRequest{ScopeType: "global"}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("store template must not become global: %v", err)
	}
	if _, err := resolveMemberGrantStore(storeFive, GrantRequest{ScopeType: "store", StoreID: storeIDPtr(6)}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("store template must not move to another store: %v", err)
	}
}

func TestGrantMemberEntitlementHandlerUsesPathMemberID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeConsoleRepo{templates: []Template{{
		ID: 1, Name: "酒水券", CouponType: TypeAlcohol, ScopeType: "global", Status: "published",
	}}}
	handler := NewConsoleHandler(NewConsoleService(repo))
	router := gin.New()
	router.POST("/admin/members/:memberID/coupon-entitlements", handler.GrantMemberEntitlement)

	body := `{"templateId":1,"scopeType":"store","storeId":5,"expiresAt":"2099-09-30T15:24:14Z","reason":"测试补发"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/members/100/coupon-entitlements", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201 without memberId in body, got %d: %s", response.Code, response.Body.String())
	}
	if len(repo.ents) != 1 || repo.ents[0].memberID != 100 || repo.ents[0].storeID == nil || *repo.ents[0].storeID != 5 {
		t.Fatalf("path member and selected store were not persisted: %+v", repo.ents)
	}
}

func TestStoreGrantMemberEntitlementForcesAuthenticatedStore(t *testing.T) {
	storeFive := int64(5)
	storeSix := int64(6)
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 1, Name: "本店饮料券", CouponType: TypeBeverage, ScopeType: "store", StoreID: &storeFive, Status: "published"},
		{ID: 2, Name: "其他门店饮料券", CouponType: TypeBeverage, ScopeType: "store", StoreID: &storeSix, Status: "published"},
	}}
	svc := NewConsoleService(repo)
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	granted, err := svc.GrantMemberEntitlement(
		context.Background(), ConsoleScope{StoreID: &storeFive}, 100,
		GrantRequest{
			TemplateID: 1, ScopeType: "global", StoreID: &storeSix,
			ExpiresAt: &expiresAt, Reason: "门店补发",
		},
		"store-idem-grant", audit.Entry{ActorID: 9, StoreID: storeFive},
	)
	if err != nil {
		t.Fatalf("grant own-store entitlement: %v", err)
	}
	if granted.Status != StatusActive || len(repo.ents) != 1 || repo.ents[0].storeID == nil || *repo.ents[0].storeID != storeFive {
		t.Fatalf("store scope was not forced onto entitlement: view=%+v entitlements=%+v", granted, repo.ents)
	}
	repo.ents = append(repo.ents, &fakeEnt{
		id: 99, no: "E2-99", tmplID: 2, memberID: 100, storeID: &storeSix, status: StatusActive,
	})
	rows, total, err := svc.ListMemberEntitlements(
		context.Background(), ConsoleScope{StoreID: &storeFive}, 100,
		httpx.Page{Page: 1, PageSize: 20},
	)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].StoreID == nil || *rows[0].StoreID != storeFive {
		t.Fatalf("store entitlement list leaked another store: rows=%+v total=%d err=%v", rows, total, err)
	}

	_, err = svc.GrantMemberEntitlement(
		context.Background(), ConsoleScope{StoreID: &storeFive}, 100,
		GrantRequest{TemplateID: 2, ExpiresAt: &expiresAt, Reason: "越权补发"},
		"store-idem-foreign", audit.Entry{ActorID: 9, StoreID: storeFive},
	)
	if apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected foreign-store template NOT_FOUND, got %v", err)
	}
}

func TestStoreMemberEntitlementsIncludeGlobalAndExcludeOtherStores(t *testing.T) {
	storeFive := int64(5)
	storeSix := int64(6)
	repo := &fakeConsoleRepo{
		templates: []Template{
			{ID: 1, Name: "全局酒水券", CouponType: TypeAlcohol, ScopeType: "global", Status: "published"},
			{ID: 2, Name: "本店券", CouponType: TypeBeverage, ScopeType: "store", StoreID: &storeFive, Status: "published"},
			{ID: 3, Name: "其他门店券", CouponType: TypeSnack, ScopeType: "store", StoreID: &storeSix, Status: "published"},
		},
		ents: []*fakeEnt{
			{id: 1, no: "GLOBAL", tmplID: 1, memberID: 100, status: StatusActive},
			{id: 2, no: "OWN", tmplID: 2, memberID: 100, storeID: &storeFive, status: StatusActive},
			{id: 3, no: "FOREIGN", tmplID: 3, memberID: 100, storeID: &storeSix, status: StatusActive},
		},
	}
	svc := NewConsoleService(repo)
	rows, total, err := svc.ListMemberEntitlements(
		context.Background(), ConsoleScope{StoreID: &storeFive}, 100,
		httpx.Page{Page: 1, PageSize: 20},
	)
	if err != nil {
		t.Fatalf("list store member entitlements: %v", err)
	}
	if total != 2 || len(rows) != 2 {
		t.Fatalf("expected global and own-store coupons only, rows=%+v total=%d", rows, total)
	}
	if rows[0].EntitlementNo != "GLOBAL" || rows[1].EntitlementNo != "OWN" {
		t.Fatalf("unexpected scoped entitlements: %+v", rows)
	}
}

func TestStoreGrantMemberEntitlementHandlerUsesPinnedStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	storeFive := int64(5)
	storeSix := int64(6)
	repo := &fakeConsoleRepo{templates: []Template{{
		ID: 1, Name: "本店酒水券", CouponType: TypeAlcohol,
		ScopeType: "store", StoreID: &storeFive, Status: "published",
	}}}
	handler := NewConsoleHandler(NewConsoleService(repo))
	router := gin.New()
	router.POST(
		"/store/members/:memberID/coupon-entitlements",
		func(c *gin.Context) {
			c.Set(httpx.CtxStoreScope, storeFive)
			c.Next()
		},
		handler.StoreGrantMemberEntitlement,
	)

	body := `{"templateId":1,"scopeType":"global","storeId":6,"expiresAt":"2099-09-30T15:24:14Z","reason":"测试门店补发"}`
	req := httptest.NewRequest(http.MethodPost, "/store/members/100/coupon-entitlements", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	if len(repo.ents) != 1 || repo.ents[0].memberID != 100 || repo.ents[0].storeID == nil || *repo.ents[0].storeID != storeFive {
		t.Fatalf("handler did not pin entitlement to authenticated store: %+v (foreign=%d)", repo.ents, storeSix)
	}
}

// A store may only grant from a template it can see.
func TestConsoleGrantScopeBoundary(t *testing.T) {
	repo := &fakeConsoleRepo{templates: []Template{
		{ID: 1, ScopeType: "store", StoreID: storeIDPtr(5), CouponType: TypeAlcohol},
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
		{ID: 1, ScopeType: "store", StoreID: storeIDPtr(5), CouponType: TypeAlcohol},
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
		{ID: 1, ScopeType: "global", CouponType: TypeAlcohol},
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
		{ID: 1, ScopeType: "store", StoreID: storeIDPtr(5), CouponType: TypeAlcohol},
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
	for _, ok := range []string{TypeEventTicket, TypeAdmissionTicket, TypeSnack, TypeAlcohol, TypeBeverage, TypeMeal} {
		if !validCouponType(ok) {
			t.Fatalf("expected %q to be valid", ok)
		}
	}
	if validCouponType("bogus") {
		t.Fatalf("expected bogus coupon type to be rejected")
	}
}

func TestNormalizedAdmissionCount(t *testing.T) {
	if got, err := normalizedAdmissionCount(TypeAdmissionTicket, 3); err != nil || got != 3 {
		t.Fatalf("expected three-person admission coupon, got count=%d err=%v", got, err)
	}
	if _, err := normalizedAdmissionCount(TypeAdmissionTicket, 0); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid zero admission count, got %v", err)
	}
	if got, err := normalizedAdmissionCount(TypeEventTicket, 8); err != nil || got != 1 {
		t.Fatalf("event coupons must normalize to one, got count=%d err=%v", got, err)
	}
	if got, err := normalizedAdmissionCount(TypeAlcohol, 8); err != nil || got != 1 {
		t.Fatalf("non-event coupons must normalize to one, got count=%d err=%v", got, err)
	}
}

// Verify rejects a request that identifies no entitlement before touching the DB.
func TestSQLConsoleRepositoryVerifyRequiresIdentifier(t *testing.T) {
	repo := &sqlConsoleRepository{}
	if _, err := repo.Verify(context.Background(), ConsoleScope{}, VerifyRequest{StoreID: 1}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}
