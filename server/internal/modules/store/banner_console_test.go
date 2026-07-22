package store

import (
	"context"
	"testing"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func isNotFound(err error) bool {
	return err != nil && apperr.From(err).Code == apperr.CodeNotFound
}

// bannerMemRepo is an in-memory Repository focused on banner CRUD; the store
// methods are unused stubs.
type bannerMemRepo struct {
	banners map[int64]Banner
	nextID  int64
}

func newBannerMemRepo() *bannerMemRepo {
	return &bannerMemRepo{banners: map[int64]Banner{}, nextID: 1}
}

func (r *bannerMemRepo) ListActiveStores(context.Context, int, int) ([]Store, int64, error) {
	return nil, 0, nil
}
func (r *bannerMemRepo) GetStore(context.Context, int64) (Store, error) { return Store{}, nil }
func (r *bannerMemRepo) ListStoreBanners(context.Context, int64) ([]Banner, error) {
	return nil, nil
}
func (r *bannerMemRepo) UpdateStoreProfile(context.Context, int64, UpdateProfileRequest) (Store, error) {
	return Store{}, nil
}
func (r *bannerMemRepo) UpdateStoreStatus(context.Context, int64, string) (Store, error) {
	return Store{}, nil
}
func (r *bannerMemRepo) GetStoreSettings(_ context.Context, storeID int64) (StoreSettings, error) {
	return StoreSettings{StoreID: storeID, SettingsJSON: []byte(`{}`)}, nil
}
func (r *bannerMemRepo) UpsertStoreSettings(_ context.Context, storeID int64, settingsJSON []byte) (StoreSettings, error) {
	return StoreSettings{StoreID: storeID, SettingsJSON: settingsJSON}, nil
}
func (r *bannerMemRepo) CreateStore(context.Context, StoreInput) (Store, error) { return Store{}, nil }
func (r *bannerMemRepo) UpdateStore(context.Context, int64, StoreInput) (Store, error) {
	return Store{}, nil
}

func (r *bannerMemRepo) ListBanners(_ context.Context, storeID *int64) ([]Banner, error) {
	var out []Banner
	for _, b := range r.banners {
		if storeID != nil && (b.StoreID == nil || *b.StoreID != *storeID) {
			continue
		}
		out = append(out, b)
	}
	return out, nil
}
func (r *bannerMemRepo) GetBanner(_ context.Context, id int64) (Banner, error) {
	b, ok := r.banners[id]
	if !ok {
		return Banner{}, apperr.NotFound("banner not found")
	}
	return b, nil
}
func (r *bannerMemRepo) CreateBanner(_ context.Context, b Banner) (Banner, error) {
	b.ID = r.nextID
	r.nextID++
	r.banners[b.ID] = b
	return b, nil
}
func (r *bannerMemRepo) UpdateBanner(_ context.Context, b Banner) (Banner, error) {
	if _, ok := r.banners[b.ID]; !ok {
		return Banner{}, apperr.NotFound("banner not found")
	}
	r.banners[b.ID] = b
	return b, nil
}
func (r *bannerMemRepo) DeleteBanner(_ context.Context, id int64) error {
	if _, ok := r.banners[id]; !ok {
		return apperr.NotFound("banner not found")
	}
	delete(r.banners, id)
	return nil
}

func newBannerSvc() (*BannerConsoleService, *bannerMemRepo) {
	repo := newBannerMemRepo()
	return NewBannerConsoleService(repo, fakeResolver{}), repo
}

func TestAdminCreateDefaultsToGlobal(t *testing.T) {
	svc, _ := newBannerSvc()
	v, err := svc.AdminCreate(context.Background(), BannerInput{AssetID: 5, Title: "Hi"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.ScopeType != BannerScopeGlobal || v.StoreID != nil {
		t.Fatalf("expected global banner, got %+v", v)
	}
	if v.Status != StatusActive {
		t.Fatalf("expected active status, got %q", v.Status)
	}
}

func TestAdminCreateStoreScopedRequiresStoreID(t *testing.T) {
	svc, _ := newBannerSvc()
	_, err := svc.AdminCreate(context.Background(), BannerInput{AssetID: 5, ScopeType: BannerScopeStore})
	if err == nil {
		t.Fatal("expected error for store scope without storeId")
	}
}

func TestStoreCreatePinsOwnScope(t *testing.T) {
	svc, _ := newBannerSvc()
	other := int64(999)
	// Client tries to set global/other store; both must be ignored.
	v, err := svc.StoreCreate(context.Background(), 7, BannerInput{
		AssetID: 5, ScopeType: BannerScopeGlobal, StoreID: &other,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.ScopeType != BannerScopeStore || v.StoreID == nil || *v.StoreID != 7 {
		t.Fatalf("expected pinned store scope 7, got %+v", v)
	}
}

func TestStoreCannotAccessOtherStoreBanner(t *testing.T) {
	svc, repo := newBannerSvc()
	other := int64(2)
	repo.CreateBanner(context.Background(), Banner{ScopeType: BannerScopeStore, StoreID: &other, AssetID: 1})

	ctx := context.Background()
	if _, err := svc.StoreGet(ctx, 1, 1); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := svc.StoreUpdate(ctx, 1, 1, BannerPatch{}); !isNotFound(err) {
		t.Fatalf("expected not found on update, got %v", err)
	}
	if err := svc.StoreDelete(ctx, 1, 1); !isNotFound(err) {
		t.Fatalf("expected not found on delete, got %v", err)
	}
}

func TestStoreListOnlyOwnBanners(t *testing.T) {
	svc, repo := newBannerSvc()
	mine, other := int64(1), int64(2)
	ctx := context.Background()
	repo.CreateBanner(ctx, Banner{ScopeType: BannerScopeStore, StoreID: &mine, AssetID: 1})
	repo.CreateBanner(ctx, Banner{ScopeType: BannerScopeStore, StoreID: &other, AssetID: 1})
	repo.CreateBanner(ctx, Banner{ScopeType: BannerScopeGlobal, AssetID: 1})

	list, err := svc.StoreList(ctx, mine)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].StoreID == nil || *list[0].StoreID != mine {
		t.Fatalf("expected only own store banner, got %+v", list)
	}
}

func TestAdminListSeesAll(t *testing.T) {
	svc, repo := newBannerSvc()
	s := int64(2)
	ctx := context.Background()
	repo.CreateBanner(ctx, Banner{ScopeType: BannerScopeStore, StoreID: &s, AssetID: 1})
	repo.CreateBanner(ctx, Banner{ScopeType: BannerScopeGlobal, AssetID: 1})

	list, err := svc.AdminList(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 banners, got %d", len(list))
	}
}

func TestStoreUpdateKeepsScope(t *testing.T) {
	svc, repo := newBannerSvc()
	mine := int64(1)
	ctx := context.Background()
	repo.CreateBanner(ctx, Banner{ScopeType: BannerScopeStore, StoreID: &mine, AssetID: 1, Title: "old"})

	global := BannerScopeGlobal
	title := "new"
	v, err := svc.StoreUpdate(ctx, mine, 1, BannerPatch{Title: &title, ScopeType: &global, StoreID: nil})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if v.Title != "new" {
		t.Fatalf("expected title updated, got %q", v.Title)
	}
	if v.ScopeType != BannerScopeStore || v.StoreID == nil || *v.StoreID != mine {
		t.Fatalf("scope must stay pinned, got %+v", v)
	}
}
