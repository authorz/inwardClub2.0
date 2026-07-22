package asset

import (
	"context"
	"testing"
	"time"
)

// seed inserts an asset directly with a chosen status and created_at, bypassing
// CreatePending so the test controls the age precisely.
func (r *memRepo) seed(status string, createdAt time.Time) int64 {
	r.seq++
	id := r.seq
	r.assets[id] = &Asset{ID: id, Status: status, CreatedAt: createdAt}
	return id
}

func TestSweepPendingAbandonsStalePendingOnly(t *testing.T) {
	repo := newMemRepo()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)

	stale := repo.seed(StatusPending, now.Add(-pendingTTL-time.Hour)) // older than TTL: abandon
	fresh := repo.seed(StatusPending, now.Add(-time.Hour))            // within TTL: keep
	uploaded := repo.seed(StatusUploaded, now.Add(-48*time.Hour))     // terminal: never touched
	bound := repo.seed(StatusBound, now.Add(-48*time.Hour))           // terminal: never touched

	svc := NewCleanupService(repo)
	svc.now = func() time.Time { return now }

	n, err := svc.SweepPending(context.Background())
	if err != nil {
		t.Fatalf("SweepPending: %v", err)
	}
	if n != 1 {
		t.Fatalf("abandoned = %d, want 1", n)
	}
	if got := repo.assets[stale].Status; got != StatusFailed {
		t.Errorf("stale pending status = %q, want %q", got, StatusFailed)
	}
	if got := repo.assets[fresh].Status; got != StatusPending {
		t.Errorf("fresh pending status = %q, want %q", got, StatusPending)
	}
	if got := repo.assets[uploaded].Status; got != StatusUploaded {
		t.Errorf("uploaded status = %q, want unchanged %q", got, StatusUploaded)
	}
	if got := repo.assets[bound].Status; got != StatusBound {
		t.Errorf("bound status = %q, want unchanged %q", got, StatusBound)
	}
}

// TestSweepPendingIdempotent: a second sweep over the same window abandons
// nothing more, because the first run left no eligible pending rows.
func TestSweepPendingIdempotent(t *testing.T) {
	repo := newMemRepo()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	repo.seed(StatusPending, now.Add(-pendingTTL-time.Hour))

	svc := NewCleanupService(repo)
	svc.now = func() time.Time { return now }

	if n, err := svc.SweepPending(context.Background()); err != nil || n != 1 {
		t.Fatalf("first sweep = (%d, %v), want (1, nil)", n, err)
	}
	if n, err := svc.SweepPending(context.Background()); err != nil || n != 0 {
		t.Fatalf("second sweep = (%d, %v), want (0, nil)", n, err)
	}
}
