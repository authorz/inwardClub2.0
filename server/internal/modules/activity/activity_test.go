package activity

import (
	"context"
	"testing"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// publicMemRepo is an in-memory public Repository for service-level tests. It
// records whether the ticket-type read was invoked so tests can assert the list
// path stays light (no per-row ticket-type lookup).
type publicMemRepo struct {
	activities   []Activity
	ticketTypes  map[int64][]TicketType
	sellableCall int
}

func (r *publicMemRepo) ListPublished(_ context.Context, storeID *int64, limit, offset int) ([]Activity, int64, error) {
	var all []Activity
	for _, a := range r.activities {
		if storeID == nil || (a.ScopeType == "global" || (a.StoreID != nil && *a.StoreID == *storeID)) {
			all = append(all, a)
		}
	}
	total := int64(len(all))
	if offset > len(all) {
		offset = len(all)
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (r *publicMemRepo) GetByID(_ context.Context, id int64) (Activity, error) {
	for _, a := range r.activities {
		if a.ID == id {
			return a, nil
		}
	}
	return Activity{}, apperr.NotFound("activity not found")
}

func (r *publicMemRepo) ListSellableTicketTypes(_ context.Context, activityID int64) ([]TicketType, error) {
	r.sellableCall++
	return r.ticketTypes[activityID], nil
}

func newPublicTestService() (*Service, *publicMemRepo) {
	repo := &publicMemRepo{ticketTypes: map[int64][]TicketType{}}
	return NewService(repo, nopAssets{}), repo
}

func TestGetAttachesSellableTicketTypes(t *testing.T) {
	svc, repo := newPublicTestService()
	repo.activities = []Activity{{ID: 5, ScopeType: "global", Title: "Show", Status: "published"}}
	repo.ticketTypes[5] = []TicketType{
		{ID: 20, ActivityID: 5, Name: "VIP", PriceCent: 9900, StockQuantity: 100, SoldQuantity: 40, PayChannels: []string{"wechat"}},
		{ID: 21, ActivityID: 5, Name: "Free", PriceCent: 0, StockQuantity: 0, SoldQuantity: 3, PayChannels: []string{}},
	}

	view, err := svc.Get(context.Background(), 5)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(view.TicketTypes) != 2 {
		t.Fatalf("expected 2 ticket types, got %d", len(view.TicketTypes))
	}
	// Limited tier: remaining = stock - sold.
	if got := view.TicketTypes[0]; got.ID != 20 || got.Stock != 60 || got.PriceCent != 9900 {
		t.Fatalf("unexpected limited ticket view: %+v", got)
	}
	// Uncapped tier (stock_quantity 0): -1 sentinel regardless of sold count.
	if got := view.TicketTypes[1]; got.ID != 21 || got.Stock != -1 {
		t.Fatalf("unexpected uncapped ticket view: %+v", got)
	}
}

func TestListDoesNotLoadTicketTypes(t *testing.T) {
	svc, repo := newPublicTestService()
	repo.activities = []Activity{{ID: 5, ScopeType: "global", Title: "Show", Status: "published"}}

	views, _, err := svc.List(context.Background(), nil, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(views) != 1 || views[0].TicketTypes != nil {
		t.Fatalf("list view must not carry ticket types: %+v", views)
	}
	if repo.sellableCall != 0 {
		t.Fatalf("list path must not query ticket types, called %d times", repo.sellableCall)
	}
}

func TestRemainingStock(t *testing.T) {
	cases := []struct {
		stock, sold, want int64
	}{
		{0, 0, -1},    // uncapped
		{0, 12, -1},   // uncapped ignores sold
		{100, 40, 60}, // remaining
		{100, 100, 0}, // sold out
		{100, 130, 0}, // oversold clamps to 0
	}
	for _, c := range cases {
		if got := remainingStock(c.stock, c.sold); got != c.want {
			t.Fatalf("remainingStock(%d,%d)=%d, want %d", c.stock, c.sold, got, c.want)
		}
	}
}
