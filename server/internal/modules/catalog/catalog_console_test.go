package catalog

import (
	"context"
	"testing"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// fakeConsoleRepo is an in-memory ConsoleRepository for tests. It records the
// last scope it was called with so tests can assert scope propagation.
type fakeConsoleRepo struct {
	categories []Category
	items      []Item
	variants   []Variant

	lastScope ConsoleScope
}

func (r *fakeConsoleRepo) ListCategories(_ context.Context, scope ConsoleScope, _ httpx.Page) ([]Category, int64, error) {
	r.lastScope = scope
	return r.categories, int64(len(r.categories)), nil
}

func (r *fakeConsoleRepo) GetCategory(_ context.Context, scope ConsoleScope, id int64) (Category, error) {
	r.lastScope = scope
	for _, c := range r.categories {
		if c.ID == id {
			return c, nil
		}
	}
	return Category{}, apperr.NotFound("catalog category not found")
}

func (r *fakeConsoleRepo) CreateCategory(_ context.Context, scope ConsoleScope, in CategoryInput) (Category, error) {
	r.lastScope = scope
	scopeType, storeID := scopeForInsert(scope)
	c := Category{ID: 100, ScopeType: scopeType, StoreID: storeID, ParentID: in.ParentID,
		Name: in.Name, AssetID: in.AssetID, SortOrder: in.SortOrder, Status: in.Status}
	r.categories = append(r.categories, c)
	return c, nil
}
func (r *fakeConsoleRepo) UpdateCategory(_ context.Context, scope ConsoleScope, id int64, in CategoryInput) (Category, error) {
	r.lastScope = scope
	for i, c := range r.categories {
		if c.ID == id {
			r.categories[i].Name = in.Name
			r.categories[i].Status = in.Status
			return r.categories[i], nil
		}
	}
	return Category{}, apperr.NotFound("catalog category not found")
}
func (r *fakeConsoleRepo) DeleteCategory(_ context.Context, scope ConsoleScope, id int64) error {
	r.lastScope = scope
	for i, c := range r.categories {
		if c.ID == id {
			r.categories = append(r.categories[:i], r.categories[i+1:]...)
			return nil
		}
	}
	return apperr.NotFound("catalog category not found")
}

func (r *fakeConsoleRepo) ListItems(_ context.Context, scope ConsoleScope, _ *int64, _ httpx.Page) ([]Item, int64, error) {
	r.lastScope = scope
	return r.items, int64(len(r.items)), nil
}

func (r *fakeConsoleRepo) GetItem(_ context.Context, scope ConsoleScope, id int64) (Item, error) {
	r.lastScope = scope
	for _, it := range r.items {
		if it.ID == id {
			return it, nil
		}
	}
	return Item{}, apperr.NotFound("catalog item not found")
}

func (r *fakeConsoleRepo) CreateItem(_ context.Context, scope ConsoleScope, in ItemInput) (Item, error) {
	r.lastScope = scope
	scopeType, storeID := scopeForInsert(scope)
	it := Item{ID: 200, ScopeType: scopeType, StoreID: storeID, CategoryID: in.CategoryID,
		Name: in.Name, Description: in.Description, ItemType: in.ItemType, PriceCent: in.PriceCent,
		StockQuantity: in.StockQuantity, PayChannels: in.PayChannels, Status: in.Status}
	r.items = append(r.items, it)
	return it, nil
}
func (r *fakeConsoleRepo) UpdateItem(_ context.Context, scope ConsoleScope, id int64, in ItemInput) (Item, error) {
	r.lastScope = scope
	for i, it := range r.items {
		if it.ID == id {
			r.items[i].Name = in.Name
			r.items[i].Status = in.Status
			return r.items[i], nil
		}
	}
	return Item{}, apperr.NotFound("catalog item not found")
}
func (r *fakeConsoleRepo) DeleteItem(_ context.Context, scope ConsoleScope, id int64) error {
	r.lastScope = scope
	for i, it := range r.items {
		if it.ID == id {
			r.items = append(r.items[:i], r.items[i+1:]...)
			return nil
		}
	}
	return apperr.NotFound("catalog item not found")
}

func (r *fakeConsoleRepo) ListVariants(_ context.Context, scope ConsoleScope, _ int64, _ httpx.Page) ([]Variant, int64, error) {
	r.lastScope = scope
	return r.variants, int64(len(r.variants)), nil
}

func (r *fakeConsoleRepo) GetVariant(_ context.Context, scope ConsoleScope, _, id int64) (Variant, error) {
	r.lastScope = scope
	for _, v := range r.variants {
		if v.ID == id {
			return v, nil
		}
	}
	return Variant{}, apperr.NotFound("catalog variant not found")
}

func (r *fakeConsoleRepo) CreateVariant(_ context.Context, scope ConsoleScope, itemID int64, in VariantInput) (Variant, error) {
	r.lastScope = scope
	v := Variant{ID: 300, ItemID: itemID, SKUCode: in.SKUCode, Name: in.Name,
		PriceCent: in.PriceCent, StockQuantity: in.StockQuantity, Status: in.Status}
	r.variants = append(r.variants, v)
	return v, nil
}
func (r *fakeConsoleRepo) UpdateVariant(_ context.Context, scope ConsoleScope, itemID, id int64, in VariantInput) (Variant, error) {
	r.lastScope = scope
	for i, v := range r.variants {
		if v.ID == id && v.ItemID == itemID {
			r.variants[i].Name = in.Name
			r.variants[i].Status = in.Status
			return r.variants[i], nil
		}
	}
	return Variant{}, apperr.NotFound("catalog variant not found")
}
func (r *fakeConsoleRepo) DeleteVariant(_ context.Context, scope ConsoleScope, itemID, id int64) error {
	r.lastScope = scope
	for i, v := range r.variants {
		if v.ID == id && v.ItemID == itemID {
			r.variants = append(r.variants[:i], r.variants[i+1:]...)
			return nil
		}
	}
	return apperr.NotFound("catalog variant not found")
}

func TestConsoleService_ListCategories_MapsAndPropagatesScope(t *testing.T) {
	repo := &fakeConsoleRepo{categories: []Category{
		{ID: 1, ScopeType: "store", Name: "Drinks", SortOrder: 1, Status: "active"},
	}}
	svc := NewConsoleService(repo)
	storeID := int64(42)
	scope := ConsoleScope{StoreID: &storeID}

	views, total, err := svc.ListCategories(context.Background(), scope, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(views) != 1 {
		t.Fatalf("expected 1 category, got total=%d len=%d", total, len(views))
	}
	if views[0].Name != "Drinks" {
		t.Fatalf("unexpected view: %+v", views[0])
	}
	if repo.lastScope.StoreID == nil || *repo.lastScope.StoreID != storeID {
		t.Fatalf("expected scope to propagate store id %d, got %+v", storeID, repo.lastScope)
	}
}

func TestConsoleService_GetItem_AdminScopePropagatesNil(t *testing.T) {
	repo := &fakeConsoleRepo{items: []Item{
		{ID: 7, Name: "Combo", Status: "draft", PayChannels: []string{"wechat"}},
	}}
	svc := NewConsoleService(repo)

	view, err := svc.GetItem(context.Background(), ConsoleScope{}, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Name != "Combo" {
		t.Fatalf("unexpected view: %+v", view)
	}
	if repo.lastScope.StoreID != nil {
		t.Fatalf("expected admin scope to propagate nil store id, got %+v", repo.lastScope)
	}
}

func TestConsoleService_GetItem_NotFound(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo)

	_, err := svc.GetItem(context.Background(), ConsoleScope{}, 999)
	if err == nil {
		t.Fatal("expected not found error")
	}
	appErr := apperr.From(err)
	if appErr.Code != apperr.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %s", appErr.Code)
	}
}

func TestConsoleService_ListVariants_PropagatesScope(t *testing.T) {
	repo := &fakeConsoleRepo{variants: []Variant{
		{ID: 1, ItemID: 7, SKUCode: "SKU-1", Name: "Small", Status: "active"},
	}}
	svc := NewConsoleService(repo)
	storeID := int64(9)
	scope := ConsoleScope{StoreID: &storeID}

	views, total, err := svc.ListVariants(context.Background(), scope, 7, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(views) != 1 || views[0].SKUCode != "SKU-1" {
		t.Fatalf("unexpected result: total=%d views=%+v", total, views)
	}
	if repo.lastScope.StoreID == nil || *repo.lastScope.StoreID != storeID {
		t.Fatalf("expected scope propagation, got %+v", repo.lastScope)
	}
}
func TestScopeForInsert(t *testing.T) {
	if st, sid := scopeForInsert(ConsoleScope{}); st != "global" || sid != nil {
		t.Fatalf("admin scope should insert global/nil, got %s/%v", st, sid)
	}
	storeID := int64(5)
	if st, sid := scopeForInsert(ConsoleScope{StoreID: &storeID}); st != "store" || sid == nil || *sid != storeID {
		t.Fatalf("store scope should insert store/%d, got %s/%v", storeID, st, sid)
	}
}

func TestScopeWhere(t *testing.T) {
	if where, args := scopeWhere(ConsoleScope{}); where != "" || args != nil {
		t.Fatalf("admin scope should not filter, got %q/%v", where, args)
	}
	storeID := int64(7)
	where, args := scopeWhere(ConsoleScope{StoreID: &storeID})
	if where != " AND scope_type = 'store' AND store_id = ?" {
		t.Fatalf("unexpected store filter clause: %q", where)
	}
	if len(args) != 1 || args[0] != storeID {
		t.Fatalf("expected store id arg, got %v", args)
	}
}

func TestEncodeChannels(t *testing.T) {
	if got := string(encodeChannels(nil)); got != "[]" {
		t.Fatalf("nil channels should encode to empty array, got %q", got)
	}
	if got := string(encodeChannels([]string{"wechat", "points"})); got != `["wechat","points"]` {
		t.Fatalf("unexpected channels encoding: %q", got)
	}
}

func TestConsoleService_CreateCategory_AdminWritesGlobal(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo)

	view, err := svc.CreateCategory(context.Background(), adminScope(), CategoryInput{Name: "Snacks", Status: "active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ScopeType != "global" || view.StoreID != nil {
		t.Fatalf("admin create should be global, got %+v", view)
	}
	if repo.lastScope.StoreID != nil {
		t.Fatalf("admin scope should propagate nil store id, got %+v", repo.lastScope)
	}
}

func TestConsoleService_CreateItem_StoreWritesOwnStore(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo)
	storeID := int64(3)

	view, err := svc.CreateItem(context.Background(), ConsoleScope{StoreID: &storeID},
		ItemInput{Name: "Latte", ItemType: ItemTypeFood, Status: "draft"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ScopeType != "store" || view.StoreID == nil || *view.StoreID != storeID {
		t.Fatalf("store create should pin store id %d, got %+v", storeID, view)
	}
}

func TestConsoleService_UpdateAndDeleteCategory(t *testing.T) {
	repo := &fakeConsoleRepo{categories: []Category{{ID: 1, ScopeType: "store", Name: "Old", Status: "active"}}}
	svc := NewConsoleService(repo)
	storeID := int64(8)
	scope := ConsoleScope{StoreID: &storeID}

	view, err := svc.UpdateCategory(context.Background(), scope, 1, CategoryInput{Name: "New", Status: "inactive"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Name != "New" || view.Status != "inactive" {
		t.Fatalf("update did not apply: %+v", view)
	}
	if repo.lastScope.StoreID == nil || *repo.lastScope.StoreID != storeID {
		t.Fatalf("update should propagate scope, got %+v", repo.lastScope)
	}
	if err := svc.DeleteCategory(context.Background(), scope, 1); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if err := svc.DeleteCategory(context.Background(), scope, 1); err == nil {
		t.Fatal("expected not found on second delete")
	}
}

func TestConsoleService_VariantWriteLifecycle(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo)
	scope := adminScope()

	v, err := svc.CreateVariant(context.Background(), scope, 7, VariantInput{SKUCode: "SKU-1", Name: "Small", Status: "active"})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if v.ItemID != 7 || v.SKUCode != "SKU-1" {
		t.Fatalf("unexpected variant: %+v", v)
	}
	updated, err := svc.UpdateVariant(context.Background(), scope, 7, v.ID, VariantInput{SKUCode: "SKU-1", Name: "Large", Status: "inactive"})
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if updated.Name != "Large" || updated.Status != "inactive" {
		t.Fatalf("update did not apply: %+v", updated)
	}
	if err := svc.DeleteVariant(context.Background(), scope, 7, v.ID); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if err := svc.DeleteVariant(context.Background(), scope, 7, v.ID); err == nil {
		t.Fatal("expected not found on second delete")
	}
}
