package reservation

import (
	"context"
	"time"
)

// noShowGrace applies only to table-only bookings. Seat bookings remain occupied
// until manual cancellation or the daily 04:00 SeatResetService sweep.
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
