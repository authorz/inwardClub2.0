package coupon

import (
	"context"
	"time"
)

// ExpiryService runs the coupon half of the ticket-coupon:expire scheduled sweep
// (spec §11): active entitlements past their validity are transitioned to
// expired. It only transitions DB state, so the worker builds it from just the
// repository.
type ExpiryService struct {
	repo Repository
	now  func() time.Time
}

// NewExpiryService builds the coupon expiry sweep service.
func NewExpiryService(repo Repository) *ExpiryService {
	return &ExpiryService{repo: repo, now: time.Now}
}

// SweepExpired expires every active entitlement whose validity window has ended.
// It returns the number of entitlements expired.
func (s *ExpiryService) SweepExpired(ctx context.Context) (int64, error) {
	return s.repo.ExpireEntitlements(ctx, s.now().UTC())
}
