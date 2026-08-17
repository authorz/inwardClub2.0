package store

import (
	"context"
	"math"
	"testing"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

type memRepo struct {
	stores []Store
}

func (r *memRepo) ListActiveStores(_ context.Context, limit, offset int) ([]Store, int64, error) {
	end := offset + limit
	if end > len(r.stores) {
		end = len(r.stores)
	}
	if offset > len(r.stores) {
		offset = len(r.stores)
	}
	return r.stores[offset:end], int64(len(r.stores)), nil
}
func (r *memRepo) GetStore(_ context.Context, id int64) (Store, error) {
	for _, s := range r.stores {
		if s.ID == id {
			return s, nil
		}
	}
	return Store{}, nil
}
func (r *memRepo) UpdateStoreProfile(_ context.Context, _ int64, _ UpdateProfileRequest) (Store, error) {
	return Store{}, nil
}
func (r *memRepo) UpdateStoreStatus(_ context.Context, _ int64, _ string) (Store, error) {
	return Store{}, nil
}
func (r *memRepo) GetStoreSettings(_ context.Context, storeID int64) (StoreSettings, error) {
	return StoreSettings{StoreID: storeID, SettingsJSON: []byte(`{}`)}, nil
}
func (r *memRepo) UpsertStoreSettings(_ context.Context, storeID int64, settingsJSON []byte) (StoreSettings, error) {
	return StoreSettings{StoreID: storeID, SettingsJSON: settingsJSON}, nil
}
func (r *memRepo) CreateStore(_ context.Context, input StoreInput) (Store, error) {
	s := Store{ID: int64(len(r.stores) + 1), Name: input.Name, Phone: input.Phone,
		Address: input.Address, BusinessHours: input.BusinessHours, Status: StatusActive}
	r.stores = append(r.stores, s)
	return s, nil
}
func (r *memRepo) UpdateStore(_ context.Context, id int64, input StoreInput) (Store, error) {
	for i, s := range r.stores {
		if s.ID == id {
			r.stores[i].Name = input.Name
			return r.stores[i], nil
		}
	}
	return Store{}, apperr.NotFound("store not found")
}

type fakeResolver struct{}

func (fakeResolver) PublicURLByID(_ context.Context, id int64) (string, error) {
	return "https://cdn.test/asset", nil
}

func TestHaversineMeters(t *testing.T) {
	// Roughly 111km per degree of latitude near the equator.
	d := haversineMeters(0, 0, 1, 0)
	if math.Abs(d-111195) > 500 {
		t.Fatalf("unexpected distance: %f", d)
	}
}

func TestListStoresComputesDistance(t *testing.T) {
	lat, lng := 31.0, 121.0
	repo := &memRepo{stores: []Store{{ID: 1, Name: "A", Status: StatusActive, Latitude: &lat, Longitude: &lng}}}
	svc := NewService(repo, fakeResolver{})

	callerLat, callerLng := 31.01, 121.0
	views, total, err := svc.ListStores(context.Background(), httpx.Page{Page: 1, PageSize: 20}, Geo{Lat: &callerLat, Lng: &callerLng})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(views) != 1 {
		t.Fatalf("expected 1 store, got %d", len(views))
	}
	if views[0].DistanceMeters == nil {
		t.Fatal("expected distance to be computed when caller geo provided")
	}
}
