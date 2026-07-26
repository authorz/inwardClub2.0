package reservation

import (
	"context"
	"testing"
	"time"
)

func TestSeatResetUsesLatestBusinessFourAMBoundary(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	repo := &memRepo{}
	svc := NewSeatResetService(repo, location)
	svc.now = func() time.Time {
		return time.Date(2026, 7, 18, 5, 0, 0, 0, location)
	}
	seatID := int64(1)
	repo.reservations = []Reservation{
		{
			ID: 1, SeatID: &seatID, Status: StatusBooked,
			CreatedAt: time.Date(2026, 7, 18, 3, 59, 0, 0, location),
		},
		{
			ID: 2, SeatID: &seatID, Status: StatusBooked,
			CreatedAt: time.Date(2026, 7, 18, 4, 1, 0, 0, location),
		},
		{
			ID: 3, Status: StatusBooked,
			CreatedAt: time.Date(2026, 7, 18, 3, 0, 0, 0, location),
		},
	}

	n, err := svc.Sweep(context.Background())
	if err != nil {
		t.Fatalf("seat reset: %v", err)
	}
	if n != 1 {
		t.Fatalf("reset count: got %d, want 1", n)
	}
	if got := statusByID(repo, 1); got != StatusExpired {
		t.Fatalf("pre-boundary seat booking: got %q, want expired", got)
	}
	if got := statusByID(repo, 2); got != StatusBooked {
		t.Fatalf("post-boundary seat booking: got %q, want booked", got)
	}
	if got := statusByID(repo, 3); got != StatusBooked {
		t.Fatalf("table-only booking: got %q, want booked", got)
	}
}

func TestSeatResetBeforeFourAMUsesPreviousDayBoundary(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	repo := &memRepo{}
	svc := NewSeatResetService(repo, location)
	svc.now = func() time.Time {
		return time.Date(2026, 7, 18, 3, 0, 0, 0, location)
	}
	seatID := int64(1)
	repo.reservations = []Reservation{
		{
			ID: 1, SeatID: &seatID, Status: StatusBooked,
			CreatedAt: time.Date(2026, 7, 17, 3, 59, 0, 0, location),
		},
		{
			ID: 2, SeatID: &seatID, Status: StatusBooked,
			CreatedAt: time.Date(2026, 7, 17, 4, 1, 0, 0, location),
		},
	}

	n, err := svc.Sweep(context.Background())
	if err != nil {
		t.Fatalf("seat reset: %v", err)
	}
	if n != 1 || statusByID(repo, 1) != StatusExpired || statusByID(repo, 2) != StatusBooked {
		t.Fatalf("unexpected pre-04:00 reset result: count=%d statuses=%q/%q", n, statusByID(repo, 1), statusByID(repo, 2))
	}
}
