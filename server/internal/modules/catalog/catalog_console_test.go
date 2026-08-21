package catalog

import (
	"context"
	"strconv"
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

	lastScope             ConsoleScope
	lastFilter            ConsoleListFilter
	rejectCouponTemplates bool
	incompatibleCategory  bool
}

type fakeCatalogAssets struct{}

func (fakeCatalogAssets) PublicURLByID(_ context.Context, id int64) (string, error) {
	return "https://cdn.example.com/assets/" + strconv.FormatInt(id, 10), nil
}

func (r *fakeConsoleRepo) ListCategories(_ context.Context, scope ConsoleScope, filter ConsoleListFilter, _ httpx.Page) ([]Category, int64, error) {
	r.lastScope = scope
	r.lastFilter = filter
	return r.categories, int64(len(r.categories)), nil
}

func (r *fakeConsoleRepo) GetCategory(_ context.Context, scope ConsoleScope, id int64) (Category, error) {
	r.lastScope = scope
	for _, c := range r.categories {
		if c.ID == id && (scope.StoreID == nil || c.StoreID != nil && *c.StoreID == *scope.StoreID) {
			return c, nil
		}
	}
	return Category{}, apperr.NotFound("catalog category not found")
}

func (r *fakeConsoleRepo) CreateCategory(_ context.Context, scope ConsoleScope, in CategoryInput) (Category, error) {
	r.lastScope = scope
	scopeType, storeID := scopeForWrite(scope, in.StoreID)
	c := Category{ID: 100, ScopeType: scopeType, StoreID: storeID, ParentID: in.ParentID,
		Name: in.Name, CategoryType: in.CategoryType, AssetID: in.AssetID, SortOrder: in.SortOrder, Status: in.Status}
	r.categories = append(r.categories, c)
	return c, nil
}
func (r *fakeConsoleRepo) UpdateCategory(_ context.Context, scope ConsoleScope, id int64, in CategoryInput) (Category, error) {
	r.lastScope = scope
	for i, c := range r.categories {
		if c.ID == id {
			r.categories[i].ScopeType = "store"
			r.categories[i].StoreID = in.StoreID
			r.categories[i].Name = in.Name
			r.categories[i].CategoryType = in.CategoryType
			r.categories[i].AssetID = in.AssetID
			r.categories[i].Status = in.Status
			return r.categories[i], nil
		}
	}
	return Category{}, apperr.NotFound("catalog category not found")
}
func (r *fakeConsoleRepo) CategoryHasIncompatibleItems(_ context.Context, _ ConsoleScope, _ int64, _ string) (bool, error) {
	return r.incompatibleCategory, nil
}
func (r *fakeConsoleRepo) CategoryHasItems(_ context.Context, _ ConsoleScope, id int64) (bool, error) {
	for _, item := range r.items {
		if item.CategoryID != nil && *item.CategoryID == id {
			return true, nil
		}
	}
	return false, nil
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

func (r *fakeConsoleRepo) ListItems(_ context.Context, scope ConsoleScope, filter ConsoleListFilter, _ httpx.Page) ([]Item, int64, error) {
	r.lastScope = scope
	r.lastFilter = filter
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
	scopeType, storeID := scopeForWrite(scope, in.StoreID)
	it := Item{ID: 200, ScopeType: scopeType, StoreID: storeID, CategoryID: in.CategoryID,
		Name: in.Name, Description: in.Description, AssetID: in.AssetID, ItemType: in.ItemType, PriceCent: in.PriceCent,
		StockQuantity: in.StockQuantity, PayChannels: in.PayChannels,
		CouponTemplateIDs: in.CouponTemplateIDs, PointsReward: in.PointsReward, Status: in.Status}
	it.GrantCouponTemplateID = in.GrantCouponTemplateID
	r.items = append(r.items, it)
	return it, nil
}
func (r *fakeConsoleRepo) UpdateItem(_ context.Context, scope ConsoleScope, id int64, in ItemInput) (Item, error) {
	r.lastScope = scope
	for i, it := range r.items {
		if it.ID == id {
			r.items[i].ScopeType = "store"
			r.items[i].StoreID = in.StoreID
			r.items[i].CategoryID = in.CategoryID
			r.items[i].AssetID = in.AssetID
			r.items[i].Name = in.Name
			r.items[i].CouponTemplateIDs = in.CouponTemplateIDs
			r.items[i].GrantCouponTemplateID = in.GrantCouponTemplateID
			r.items[i].Status = in.Status
			return r.items[i], nil
		}
	}
	return Item{}, apperr.NotFound("catalog item not found")
}

func (r *fakeConsoleRepo) CouponTemplatesExistForStore(_ context.Context, _ int64, _ []int64) (bool, error) {
	return !r.rejectCouponTemplates, nil
}
func (r *fakeConsoleRepo) CouponTemplateAvailableForSale(_ context.Context, _, _ int64) (bool, error) {
	return !r.rejectCouponTemplates, nil
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
	assetID := int64(18)
	repo := &fakeConsoleRepo{categories: []Category{
		{ID: 1, ScopeType: "store", Name: "Drinks", AssetID: &assetID, SortOrder: 1, Status: "active"},
	}}
	svc := NewConsoleService(repo, fakeCatalogAssets{})
	storeID := int64(42)
	scope := ConsoleScope{StoreID: &storeID}

	filter := ConsoleListFilter{Keyword: "Drink"}
	views, total, err := svc.ListCategories(context.Background(), scope, filter, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(views) != 1 {
		t.Fatalf("expected 1 category, got total=%d len=%d", total, len(views))
	}
	if views[0].Name != "Drinks" {
		t.Fatalf("unexpected view: %+v", views[0])
	}
	if views[0].ImageURL != "https://cdn.example.com/assets/18" {
		t.Fatalf("expected resolved category icon, got %+v", views[0])
	}
	if repo.lastScope.StoreID == nil || *repo.lastScope.StoreID != storeID {
		t.Fatalf("expected scope to propagate store id %d, got %+v", storeID, repo.lastScope)
	}
	if repo.lastFilter.Keyword != "Drink" {
		t.Fatalf("expected filter to propagate, got %+v", repo.lastFilter)
	}
}

func TestConsoleService_CreateCouponItem_RequiresAndPersistsGrantTemplate(t *testing.T) {
	storeID := int64(42)
	assetID := int64(18)
	templateID := int64(9)
	categoryID := int64(7)
	repo := &fakeConsoleRepo{categories: []Category{{
		ID: categoryID, StoreID: &storeID, CategoryType: CategoryTypeCoupon,
	}}}
	svc := NewConsoleService(repo)

	view, err := svc.CreateItem(context.Background(), ConsoleScope{StoreID: &storeID}, ItemInput{
		CategoryID: &categoryID, Name: "酒水券 3 张装", AssetID: &assetID,
		ItemType: ItemTypeFood, PriceCent: 9900, StockQuantity: 20,
		PayChannels: []string{"wechat", "coin"}, CouponTemplateIDs: []int64{88},
		GrantCouponTemplateID: &templateID, PointsReward: 100, Status: "draft",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ItemType != ItemTypeCoupon || view.GrantCouponTemplateID == nil || *view.GrantCouponTemplateID != templateID {
		t.Fatalf("expected coupon item bound to template %d, got %+v", templateID, view)
	}
	if len(view.CouponTemplateIDs) != 0 {
		t.Fatalf("coupon sale item must not also be coupon-redeemable, got %+v", view.CouponTemplateIDs)
	}
	if view.PointsReward != 100 {
		t.Fatalf("expected points reward to remain configurable, got %d", view.PointsReward)
	}
}

func TestConsoleService_CreateProductItem_RejectsGrantTemplate(t *testing.T) {
	storeID := int64(42)
	assetID := int64(18)
	templateID := int64(9)
	categoryID := int64(7)
	repo := &fakeConsoleRepo{categories: []Category{{
		ID: categoryID, StoreID: &storeID, CategoryType: CategoryTypeProduct,
	}}}
	svc := NewConsoleService(repo)

	_, err := svc.CreateItem(context.Background(), ConsoleScope{StoreID: &storeID}, ItemInput{
		CategoryID: &categoryID, Name: "啤酒", AssetID: &assetID,
		ItemType: ItemTypeFood, PayChannels: []string{"wechat"},
		GrantCouponTemplateID: &templateID, Status: "draft",
	})
	if err == nil || apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestConsoleService_UpdateCategory_RejectsIncompatibleItems(t *testing.T) {
	storeID := int64(42)
	repo := &fakeConsoleRepo{
		categories:           []Category{{ID: 7, StoreID: &storeID, CategoryType: CategoryTypeProduct}},
		incompatibleCategory: true,
	}
	svc := NewConsoleService(repo)

	_, err := svc.UpdateCategory(context.Background(), ConsoleScope{StoreID: &storeID}, 7, CategoryInput{
		Name: "售券", CategoryType: CategoryTypeCoupon, Status: "active",
	})
	if err == nil || apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestConsoleService_GetItem_AdminScopePropagatesNil(t *testing.T) {
	assetID := int64(19)
	repo := &fakeConsoleRepo{items: []Item{
		{ID: 7, Name: "Combo", AssetID: &assetID, Status: "draft", PayChannels: []string{"wechat"}},
	}}
	svc := NewConsoleService(repo, fakeCatalogAssets{})

	view, err := svc.GetItem(context.Background(), ConsoleScope{}, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Name != "Combo" {
		t.Fatalf("unexpected view: %+v", view)
	}
	if view.ImageURL != "https://cdn.example.com/assets/19" {
		t.Fatalf("expected resolved item image, got %+v", view)
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

func TestConsoleListWhereSupportsStoreAndEscapedFuzzyName(t *testing.T) {
	storeID := int64(7)
	where, args := consoleListWhere(ConsoleScope{}, ConsoleListFilter{
		StoreID: &storeID,
		Keyword: `Latte%_`,
		Status:  "published",
	}, "i")
	wantWhere := `1=1 AND i.store_id = ? AND i.name LIKE ? ESCAPE '\\' AND i.status = ?`
	if where != wantWhere {
		t.Fatalf("unexpected list filter: %q", where)
	}
	if len(args) != 3 || args[0] != storeID || args[1] != `%Latte\%\_%` || args[2] != "published" {
		t.Fatalf("unexpected list args: %#v", args)
	}
}

func TestEncodeChannels(t *testing.T) {
	if got := string(encodeChannels(nil)); got != "[]" {
		t.Fatalf("nil channels should encode to empty array, got %q", got)
	}
	if got := string(encodeChannels([]string{"wechat", "coin"})); got != `["wechat","coin"]` {
		t.Fatalf("unexpected channels encoding: %q", got)
	}
}

func TestNormalizePayChannels(t *testing.T) {
	got, err := normalizePayChannels([]string{"wechat", "balance", "coin"})
	if err != nil {
		t.Fatalf("normalize legacy balance: %v", err)
	}
	if len(got) != 2 || got[0] != "wechat" || got[1] != "coin" {
		t.Fatalf("unexpected normalized channels: %#v", got)
	}
	if _, err := normalizePayChannels([]string{"wechat", "points"}); err == nil {
		t.Fatal("unsupported channel should be rejected")
	}
}

func TestNormalizeCouponTemplateIDs(t *testing.T) {
	got, err := normalizeCouponTemplateIDs([]int64{3, 1, 3})
	if err != nil {
		t.Fatalf("normalize coupon template IDs: %v", err)
	}
	if len(got) != 2 || got[0] != 3 || got[1] != 1 {
		t.Fatalf("unexpected coupon template IDs: %#v", got)
	}
	if _, err := normalizeCouponTemplateIDs([]int64{0}); err == nil {
		t.Fatal("non-positive coupon template ID should be rejected")
	}
}

func TestConsoleService_CreateCategory_AdminRequiresAndBindsStore(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo, fakeCatalogAssets{})
	storeID := int64(6)
	assetID := int64(23)

	view, err := svc.CreateCategory(context.Background(), adminScope(), CategoryInput{
		StoreID: &storeID, Name: "Snacks", AssetID: &assetID, Status: "active",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ScopeType != "store" || view.StoreID == nil || *view.StoreID != storeID {
		t.Fatalf("admin create should bind store %d, got %+v", storeID, view)
	}
	if view.AssetID == nil || *view.AssetID != assetID || view.ImageURL != "https://cdn.example.com/assets/23" {
		t.Fatalf("create should return the uploaded category icon, got %+v", view)
	}
	if _, err := svc.CreateCategory(context.Background(), adminScope(), CategoryInput{
		Name: "Missing Store", Status: "active",
	}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected missing store to be rejected, got %v", err)
	}
}

func TestConsoleService_CreateItem_StoreWritesOwnStore(t *testing.T) {
	storeID := int64(3)
	categoryID := int64(8)
	assetID := int64(9)
	repo := &fakeConsoleRepo{categories: []Category{
		{ID: categoryID, ScopeType: "store", StoreID: &storeID, Name: "Drinks"},
	}}
	svc := NewConsoleService(repo)

	view, err := svc.CreateItem(context.Background(), ConsoleScope{StoreID: &storeID},
		ItemInput{CategoryID: &categoryID, AssetID: &assetID, Name: "Latte", Status: "draft"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ScopeType != "store" || view.StoreID == nil || *view.StoreID != storeID {
		t.Fatalf("store create should pin store id %d, got %+v", storeID, view)
	}
	if view.ItemType != ItemTypeFood {
		t.Fatalf("empty internal item type should default to food, got %+v", view)
	}
}

func TestConsoleService_CreateItem_AdminRejectsCategoryFromAnotherStore(t *testing.T) {
	storeID := int64(3)
	otherStoreID := int64(4)
	categoryID := int64(8)
	assetID := int64(9)
	repo := &fakeConsoleRepo{categories: []Category{
		{ID: categoryID, ScopeType: "store", StoreID: &otherStoreID, Name: "Other Store"},
	}}
	svc := NewConsoleService(repo)

	_, err := svc.CreateItem(context.Background(), adminScope(), ItemInput{
		StoreID: &storeID, CategoryID: &categoryID, AssetID: &assetID,
		Name: "Latte", Status: "draft",
	})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected cross-store category to be rejected, got %v", err)
	}
}

func TestConsoleService_CreateItem_RejectsNegativePointsReward(t *testing.T) {
	storeID := int64(3)
	categoryID := int64(8)
	assetID := int64(9)
	repo := &fakeConsoleRepo{categories: []Category{
		{ID: categoryID, ScopeType: "store", StoreID: &storeID, Name: "Drinks"},
	}}
	svc := NewConsoleService(repo)

	_, err := svc.CreateItem(context.Background(), adminScope(), ItemInput{
		StoreID: &storeID, CategoryID: &categoryID, AssetID: &assetID,
		Name: "Latte", PointsReward: -1, Status: "draft",
	})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected negative points reward to be rejected, got %v", err)
	}
}

func TestConsoleService_CreateItem_RejectsCouponTemplateOutsideStore(t *testing.T) {
	storeID := int64(3)
	categoryID := int64(8)
	assetID := int64(9)
	repo := &fakeConsoleRepo{
		rejectCouponTemplates: true,
		categories: []Category{
			{ID: categoryID, ScopeType: "store", StoreID: &storeID, Name: "Drinks"},
		},
	}
	svc := NewConsoleService(repo)

	_, err := svc.CreateItem(context.Background(), adminScope(), ItemInput{
		StoreID: &storeID, CategoryID: &categoryID, AssetID: &assetID,
		Name: "Latte", CouponTemplateIDs: []int64{99}, Status: "draft",
	})
	if apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected out-of-scope coupon template to be rejected, got %v", err)
	}
}

func TestConsoleService_UpdateAndDeleteCategory(t *testing.T) {
	repo := &fakeConsoleRepo{categories: []Category{{ID: 1, ScopeType: "store", Name: "Old", Status: "active"}}}
	svc := NewConsoleService(repo, fakeCatalogAssets{})
	storeID := int64(8)
	assetID := int64(29)
	scope := ConsoleScope{StoreID: &storeID}

	view, err := svc.UpdateCategory(context.Background(), scope, 1, CategoryInput{Name: "New", AssetID: &assetID, Status: "inactive"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Name != "New" || view.Status != "inactive" {
		t.Fatalf("update did not apply: %+v", view)
	}
	if view.AssetID == nil || *view.AssetID != assetID || view.ImageURL != "https://cdn.example.com/assets/29" {
		t.Fatalf("update should return the uploaded category icon, got %+v", view)
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

func TestConsoleService_DeleteCategory_RejectsCategoryWithItems(t *testing.T) {
	categoryID := int64(1)
	repo := &fakeConsoleRepo{
		categories: []Category{{ID: categoryID, ScopeType: "store", Name: "Drinks", Status: "active"}},
		items:      []Item{{ID: 10, CategoryID: &categoryID, Name: "Beer"}},
	}
	svc := NewConsoleService(repo)

	err := svc.DeleteCategory(context.Background(), adminScope(), categoryID)
	if apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected occupied category conflict, got %v", err)
	}
	if len(repo.categories) != 1 {
		t.Fatal("occupied category must not be deleted")
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
