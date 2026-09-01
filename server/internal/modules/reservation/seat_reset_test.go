package reservation

import (
	"context"
	"testing"
	"time"
)

func TestSeatResetUsesLatestMidnightBoundary(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	repo := &memRepo{}
	svc := NewSeatResetService(repo, location)
	svc.now = func() time.Time {
		return time.Date(2026, 7, 18, 0, 5, 0, 0, location)
	}
	seatID := int64(1)
	repo.reservations = []Reservation{
		{
			ID: 1, SeatID: &seatID, Status: StatusBooked,
			CreatedAt: time.Date(2026, 7, 17, 23, 59, 0, 0, location),
		},
		{
			ID: 2, SeatID: &seatID, Status: StatusBooked,
			CreatedAt: time.Date(2026, 7, 18, 0, 1, 0, 0, location),
		},
		{
			ID: 3, Status: StatusBooked,
			CreatedAt: time.Date(2026, 7, 17, 23, 0, 0, 0, location),
		},
	}

	n, err := svc.Sweep(context.Background())
	if err != nil {
		t.Fatalf("seat reset: %v", err)
	}
	if n != 1 {
		t.Fatalf("reset count: got %d, want 1", n)
	}
	if got := statusByID(repo, 1); got != "" {
		t.Fatalf("pre-midnight seat booking must be deleted, got %q", got)
	}
	if got := statusByID(repo, 2); got != StatusBooked {
		t.Fatalf("post-midnight seat booking: got %q, want booked", got)
	}
	if got := statusByID(repo, 3); got != StatusBooked {
		t.Fatalf("table-only booking: got %q, want booked", got)
	}
}
