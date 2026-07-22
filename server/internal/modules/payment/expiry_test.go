package payment

import (
	"context"
	"testing"
	"time"
)

// ExpireCollections on the in-memory repo mirrors the SQL sweep contract: only
// pending collection orders past their expiry flip to expired.
func (r *memStoreRepo) ExpireCollections(_ context.Context, now time.Time) (int64, error) {
	var n int64
	for i := range r.orders {
		o := &r.orders[i]
		if o.Status == CollectionPending && o.ExpiresAt.Before(now) {
			o.Status = CollectionExpired
			n++
		}
	}
	return n, nil
}

func TestSweepExpiredCollections(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute)
	future := now.Add(time.Hour)
	repo := &memStoreRepo{orders: []CollectionOrder{
		{ID: 1, StoreID: 1, Status: CollectionPending, ExpiresAt: past},   // expire
		{ID: 2, StoreID: 1, Status: CollectionPending, ExpiresAt: future}, // still valid -> keep
		{ID: 3, StoreID: 1, Status: CollectionPaid, ExpiresAt: past},      // paid -> keep
		{ID: 4, StoreID: 1, Status: CollectionCancelled, ExpiresAt: past}, // cancelled -> keep
	}}
	svc := NewCollectionExpiryService(repo)
	svc.now = func() time.Time { return now }

	n, err := svc.SweepExpired(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired count: got %d, want 1", n)
	}
	if repo.orders[0].Status != CollectionExpired {
		t.Fatalf("collection 1 should be expired, got %q", repo.orders[0].Status)
	}
	if repo.orders[1].Status != CollectionPending {
		t.Fatalf("collection 2 should stay pending, got %q", repo.orders[1].Status)
	}
	if repo.orders[2].Status != CollectionPaid || repo.orders[3].Status != CollectionCancelled {
		t.Fatalf("paid/cancelled collections should be untouched")
	}

	again, err := svc.SweepExpired(context.Background())
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if again != 0 {
		t.Fatalf("second sweep count: got %d, want 0", again)
	}
}
