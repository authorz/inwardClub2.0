package reservation

import (
	"context"
	"time"
)

// noShowGrace applies only to table-only bookings. Seat bookings remain occupied
// until manual cancellation or the daily 04:00 SeatResetService sweep.
const noShowGrace = 2 * time.Hour

// ExpiryService runs the reservation cleanup sweep. It deletes table-only
// no-shows after the grace period.
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
// removed.
func (s *ExpiryService) SweepExpired(ctx context.Context) (int64, error) {
	now := s.now().UTC()
	return s.repo.ExpireBookings(ctx, now.Add(-noShowGrace), now)
}
