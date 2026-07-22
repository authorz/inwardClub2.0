package reporting

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRollupRepo struct {
	got RollupRequest
	res RollupResult
	err error
}

func (f *fakeRollupRepo) RollupDaily(_ context.Context, req RollupRequest) (RollupResult, error) {
	f.got = req
	return f.res, f.err
}

// TestRollupForwardsRequestAndResult: the service is a thin seam over the write
// repository — it must pass the bounds through untouched and return the result.
func TestRollupForwardsRequestAndResult(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	store := int64(42)
	repo := &fakeRollupRepo{res: RollupResult{RevenueRows: 7, ReservationRows: 3}}
	svc := NewRollupService(repo)

	res, err := svc.Rollup(context.Background(), RollupRequest{From: &from, To: &to, StoreID: &store})
	if err != nil {
		t.Fatalf("rollup: %v", err)
	}
	if res.RevenueRows != 7 || res.ReservationRows != 3 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if repo.got.From == nil || !repo.got.From.Equal(from) ||
		repo.got.To == nil || !repo.got.To.Equal(to) ||
		repo.got.StoreID == nil || *repo.got.StoreID != store {
		t.Fatalf("bounds not forwarded: %+v", repo.got)
	}
}

// TestRollupFullRecompute: a zero-value request (the scheduled/startup case)
// forwards unbounded, i.e. recompute every date and store.
func TestRollupFullRecompute(t *testing.T) {
	repo := &fakeRollupRepo{}
	svc := NewRollupService(repo)

	if _, err := svc.Rollup(context.Background(), RollupRequest{}); err != nil {
		t.Fatalf("rollup: %v", err)
	}
	if repo.got.From != nil || repo.got.To != nil || repo.got.StoreID != nil {
		t.Fatalf("expected unbounded full recompute, got %+v", repo.got)
	}
}

// TestRollupPropagatesError: a pipeline failure surfaces so the worker can retry.
func TestRollupPropagatesError(t *testing.T) {
	sentinel := errors.New("boom")
	svc := NewRollupService(&fakeRollupRepo{err: sentinel})

	if _, err := svc.Rollup(context.Background(), RollupRequest{}); !errors.Is(err, sentinel) {
		t.Fatalf("expected propagated error, got %v", err)
	}
}
