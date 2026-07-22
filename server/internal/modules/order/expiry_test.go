package order

import (
	"context"
	"testing"
	"time"
)

// fakeActivityOrder is the in-memory stand-in for the activity-order spine
// (activity order + business order + payment order + reserved stock) the SQL
// sweep transitions atomically.
type fakeActivityOrder struct {
	id           int64
	createdAt    time.Time
	aoStatus     string // created / expired / paid
	poStatus     string // pending / paid / expired
	payStatus    string // unpaid / paid
	ticketTypeID int64
	pendingTix   int
}

// fakeTicket is the in-memory stand-in for a ticket with its resolved event end.
type fakeTicket struct {
	id     int64
	status string
	endAt  *time.Time
}

// ExpireActivityOrders mirrors the SQL sweep contract branch-for-branch: only
// created+unpaid+pending orders older than the cutoff are closed, their stock is
// released (never below zero) and their pending tickets/payment order expire.
func (r *memRepo) ExpireActivityOrders(_ context.Context, createdBefore, now time.Time) (int64, error) {
	var n int64
	for i := range r.activityOrders {
		o := &r.activityOrders[i]
		if o.aoStatus == "created" && o.payStatus == "unpaid" && o.poStatus == PaymentStatusPending && o.createdAt.Before(createdBefore) {
			if r.sold != nil {
				if released := r.sold[o.ticketTypeID] - int64(o.pendingTix); released > 0 {
					r.sold[o.ticketTypeID] = released
				} else {
					r.sold[o.ticketTypeID] = 0
				}
			}
			o.aoStatus = "expired"
			o.poStatus = PaymentStatusExpired
			o.pendingTix = 0
			n++
		}
	}
	return n, nil
}

// ExpireTickets mirrors the SQL sweep: only active tickets whose (already
// resolved) event end has passed flip to expired.
func (r *memRepo) ExpireTickets(_ context.Context, now time.Time) (int64, error) {
	var n int64
	for i := range r.sweepTickets {
		t := &r.sweepTickets[i]
		if t.status == TicketStatusActive && t.endAt != nil && t.endAt.Before(now) {
			t.status = TicketStatusExpired
			n++
		}
	}
	return n, nil
}

func newOrderExpiryService(repo *memRepo) *ExpiryService {
	svc := NewExpiryService(repo)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	return svc
}

func TestSweepExpiredActivityOrders(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	// pay window = 15m, so the cutoff is 11:45.
	repo := &memRepo{
		sold: map[int64]int64{100: 5},
		activityOrders: []fakeActivityOrder{
			{id: 1, createdAt: now.Add(-30 * time.Minute), aoStatus: "created", poStatus: PaymentStatusPending, payStatus: "unpaid", ticketTypeID: 100, pendingTix: 2}, // expire
			{id: 2, createdAt: now.Add(-5 * time.Minute), aoStatus: "created", poStatus: PaymentStatusPending, payStatus: "unpaid", ticketTypeID: 100, pendingTix: 1},  // within window -> keep
			{id: 3, createdAt: now.Add(-time.Hour), aoStatus: "created", poStatus: PaymentStatusPaid, payStatus: "paid", ticketTypeID: 100, pendingTix: 0},             // paid -> keep
		},
	}
	svc := newOrderExpiryService(repo)

	n, err := svc.SweepExpiredActivityOrders(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired count: got %d, want 1", n)
	}
	if repo.activityOrders[0].aoStatus != "expired" || repo.activityOrders[0].poStatus != PaymentStatusExpired {
		t.Fatalf("order 1 not fully expired: %+v", repo.activityOrders[0])
	}
	if repo.sold[100] != 3 { // 5 reserved - 2 released
		t.Fatalf("stock release: got sold=%d, want 3", repo.sold[100])
	}
	if repo.activityOrders[1].aoStatus != "created" {
		t.Fatalf("order 2 (within window) should be untouched, got %q", repo.activityOrders[1].aoStatus)
	}
	if repo.activityOrders[2].aoStatus != "created" && repo.activityOrders[2].aoStatus != "paid" {
		t.Fatalf("order 3 (paid) should be untouched")
	}

	// Idempotent: a second sweep closes nothing more and releases no more stock.
	again, err := svc.SweepExpiredActivityOrders(context.Background())
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if again != 0 {
		t.Fatalf("second sweep count: got %d, want 0", again)
	}
	if repo.sold[100] != 3 {
		t.Fatalf("stock double-released: got sold=%d, want 3", repo.sold[100])
	}
}

func TestSweepExpiredTickets(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	repo := &memRepo{sweepTickets: []fakeTicket{
		{id: 1, status: TicketStatusActive, endAt: &past},   // event ended -> expire
		{id: 2, status: TicketStatusActive, endAt: &future}, // upcoming -> keep
		{id: 3, status: TicketStatusActive, endAt: nil},     // no deadline -> keep
		{id: 4, status: TicketStatusPending, endAt: &past},  // unpaid -> keep (owned by activity-order:expire)
		{id: 5, status: "used", endAt: &past},               // terminal -> keep
	}}
	svc := newOrderExpiryService(repo)

	n, err := svc.SweepExpiredTickets(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Fatalf("expired count: got %d, want 1", n)
	}
	if repo.sweepTickets[0].status != TicketStatusExpired {
		t.Fatalf("ticket 1 should be expired, got %q", repo.sweepTickets[0].status)
	}
	if repo.sweepTickets[1].status != TicketStatusActive || repo.sweepTickets[2].status != TicketStatusActive {
		t.Fatalf("tickets 2/3 should stay active")
	}
	if repo.sweepTickets[3].status != TicketStatusPending {
		t.Fatalf("ticket 4 should stay pending")
	}

	again, err := svc.SweepExpiredTickets(context.Background())
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if again != 0 {
		t.Fatalf("second sweep count: got %d, want 0", again)
	}
}
