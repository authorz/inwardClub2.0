package reservation

import (
	"context"
	"time"
)

const seatResetHour = 0

// SeatResetService releases seat reservations at the latest local midnight.
// Using a cutoff instead of clearing every booked row makes the
// sweep safe to run both on schedule and once after a worker restart.
type SeatResetService struct {
	repo     Repository
	location *time.Location
	now      func() time.Time
}

// NewSeatResetService builds the daily seat reset sweep.
func NewSeatResetService(repo Repository, location *time.Location) *SeatResetService {
	if location == nil {
		location = time.UTC
	}
	return &SeatResetService{repo: repo, location: location, now: time.Now}
}

// Sweep deletes bookings created before the latest local midnight.
func (s *SeatResetService) Sweep(ctx context.Context) (int64, error) {
	now := s.now().In(s.location)
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), seatResetHour, 0, 0, 0, s.location)
	return s.repo.ClearSeatBookings(ctx, cutoff.UTC(), now.UTC())
}
