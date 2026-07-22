package reservation

import (
	"context"
	"time"
)

// noShowGrace is how long a booking lingers past its reserved time before the
// reservation:expire sweep releases it as a no-show. It is intentionally
// generous so a party that arrives late (and is then marked arrived) is never
// expired out from under the store; the exact window is a business input
// (spec §13) and is tuned here rather than in the SQL.
const noShowGrace = 2 * time.Hour

// ExpiryService runs the reservation:expire scheduled sweep (spec §11). It only
// transitions reservation state and needs no adapters, so the worker builds it
// from just the repository.
type ExpiryService struct {
	repo Repository
	now  func() time.Time
}

// NewExpiryService builds the reservation expiry sweep service.
func NewExpiryService(repo Repository) *ExpiryService {
	return &ExpiryService{repo: repo, now: time.Now}
}

// SweepExpired releases every still-booked reservation whose reserved time
// passed more than noShowGrace ago. It returns the number of reservations
// expired.
func (s *ExpiryService) SweepExpired(ctx context.Context) (int64, error) {
	now := s.now().UTC()
	return s.repo.ExpireBookings(ctx, now.Add(-noShowGrace), now)
}
