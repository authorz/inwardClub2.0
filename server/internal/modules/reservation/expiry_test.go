package reservation

import (
	"context"
	"testing"
	"time"
)

// ExpireBookings on the in-memory repo mirrors the SQL sweep contract: only
// still-booked reservations reserved before the cutoff flip to expired.
func (r *memRepo) ExpireBookings(_ context.Context, reservedBefore, now time.Time) (int64, error) {
	var n int64
	for i := range r.reservations {
		res := &r.reservations[i]
		if res.Status == StatusBooked && res.SeatID == nil && res.ReservedAt.Before(reservedBefore) {
			res.Status = StatusExpired
			res.UpdatedAt = now
			n++
		}
	}
	return n, nil
}

func (r *memRepo) ClearSeatBookings(_ context.Context, createdBefore, now time.Time) (int64, error) {
	var n int64
	for i := range r.reservations {
		res := &r.reservations[i]
		if res.Status == StatusBooked && res.SeatID != nil && res.CreatedAt.Before(createdBefore) {
			res.Status = StatusExpired
			res.UpdatedAt = now
			n++
		}
	}
	return n, nil
}

func newExpiryService() (*ExpiryService, *memRepo) {
	repo := &memRepo{}
	svc := NewExpiryService(repo)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	return svc, repo
}

func statusByID(repo *memRepo, id int64) string {
	for _, r := range repo.reservations {
		if r.ID == id {
			return r.Status
		}
	}
	return ""
}

func TestSweepExpiredReleasesNoShowBookings(t *testing.T) {
	svc, repo := newExpiryService()
	// now = 12:00; grace = 2h, so the cutoff is 10:00.
	repo.reservations = []Reservation{
		{ID: 1, MemberID: 1, StoreID: 1, Status: StatusBooked, ReservedAt: time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC)},                      // past grace -> expire
		{ID: 2, MemberID: 1, StoreID: 1, Status: StatusBooked, ReservedAt: time.Date(2026, 7, 17, 11, 0, 0, 0, time.UTC)},                     // within grace -> keep
		{ID: 3, MemberID: 1, StoreID: 1, Status: StatusBooked, ReservedAt: time.Date(2026, 7, 18, 19, 0, 0, 0, time.UTC)},                     // future -> keep
		{ID: 4, MemberID: 1, StoreID: 1, Status: StatusArrived, ReservedAt: time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC)},                     // arrived -> keep
		{ID: 5, MemberID: 1, StoreID: 1, Status: StatusCancelled, ReservedAt: time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC)},                   // cancelled -> keep
		{ID: 6, MemberID: 1, StoreID: 1, SeatID: int64Ptr(8), Status: StatusBooked, ReservedAt: time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC)}, // seat booking -> daily reset
	}

	n, err := svc.SweepExpired(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired count: got %d, want 1", n)
	}
	if got := statusByID(repo, 1); got != StatusExpired {
		t.Fatalf("reservation 1 status: got %q, want expired", got)
	}
	for _, id := range []int64{2, 3} {
		if got := statusByID(repo, id); got != StatusBooked {
			t.Fatalf("reservation %d status: got %q, want booked", id, got)
		}
	}
	if got := statusByID(repo, 4); got != StatusArrived {
		t.Fatalf("reservation 4 status: got %q, want arrived", got)
	}
	if got := statusByID(repo, 5); got != StatusCancelled {
		t.Fatalf("reservation 5 status: got %q, want cancelled", got)
	}
	if got := statusByID(repo, 6); got != StatusBooked {
		t.Fatalf("seat reservation status: got %q, want booked", got)
	}

	// Idempotent: a second sweep transitions nothing.
	again, err := svc.SweepExpired(context.Background())
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if again != 0 {
		t.Fatalf("second sweep count: got %d, want 0", again)
	}
}

func int64Ptr(v int64) *int64 { return &v }
