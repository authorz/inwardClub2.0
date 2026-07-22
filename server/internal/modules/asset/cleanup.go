package asset

import (
	"context"
	"time"
)

// pendingTTL is how long an asset may sit in 'pending' before the
// asset:pending-cleanup sweep (task spec §11, Qiniu spec §8) abandons it. The
// upload token is only valid for minutes (QiniuConfig.TokenTTL defaults to 10m),
// so a row still pending after this window can never have its callback confirm
// it. The 24h window matches the documented cleanup policy and is generous enough
// that a slow or retried upload is never swept out from under an in-flight client.
const pendingTTL = 24 * time.Hour

// CleanupService runs the asset:pending-cleanup scheduled sweep: it abandons
// stale pending assets whose upload never confirmed. Like the other periodic
// sweeps it only transitions DB state, so the worker builds it from just the
// repository.
//
// Reaping the corresponding Qiniu objects (the abandoned pending object and any
// orphaned objects under the current env key prefix, Qiniu spec §8) is not done
// here: it needs the ObjectStore wired into the sweep plus a bucket-listing
// adapter method that does not exist yet, and the abandoned pending object may
// never have been uploaded at all. The DB transition alone bounds the unbounded
// growth of pending rows; object reaping stays a documented follow-up.
type CleanupService struct {
	repo Repository
	now  func() time.Time
}

// NewCleanupService builds the asset pending-cleanup sweep service.
func NewCleanupService(repo Repository) *CleanupService {
	return &CleanupService{repo: repo, now: time.Now}
}

// SweepPending marks every asset stuck in 'pending' past pendingTTL as failed and
// returns the number abandoned. Idempotent: the repository's status='pending'
// guard means a re-run touches zero rows.
func (s *CleanupService) SweepPending(ctx context.Context) (int64, error) {
	return s.repo.ExpirePending(ctx, s.now().UTC().Add(-pendingTTL))
}
