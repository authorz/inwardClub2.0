package reservation

import (
	"context"
	"testing"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// memRepo is an in-memory Repository for service-level tests.
type memRepo struct {
	tables       []Table
	seats        []Seat
	reservations []Reservation
	waitlist     []WaitlistEntry
	arrivals     int
	nextID       int64
}

func (r *memRepo) ListTables(_ context.Context, storeID int64) ([]Table, error) {
	var out []Table
	for _, t := range r.tables {
		if t.StoreID == storeID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *memRepo) ListSeats(_ context.Context, storeID int64) ([]Seat, error) {
	var out []Seat
	for _, s := range r.seats {
		if s.StoreID == storeID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *memRepo) ListMemberReservations(_ context.Context, memberID int64, limit, offset int) ([]Reservation, int64, error) {
	var all []Reservation
	for _, res := range r.reservations {
		if res.MemberID == memberID {
			all = append(all, res)
		}
	}
	total := int64(len(all))
	if offset > len(all) {
		offset = len(all)
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (r *memRepo) ListStoreReservations(_ context.Context, storeID int64, limit, offset int) ([]Reservation, int64, error) {
	var all []Reservation
	for _, res := range r.reservations {
		if res.StoreID == storeID {
			all = append(all, res)
		}
	}
	total := int64(len(all))
	if offset > len(all) {
		offset = len(all)
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (r *memRepo) CreateReservation(_ context.Context, res Reservation) (int64, error) {
	r.nextID++
	res.ID = r.nextID
	r.reservations = append(r.reservations, res)
	return res.ID, nil
}

func (r *memRepo) GetReservation(_ context.Context, id int64) (Reservation, error) {
	for _, res := range r.reservations {
		if res.ID == id {
			return res, nil
		}
	}
	return Reservation{}, apperr.NotFound("reservation not found")
}

func (r *memRepo) CancelReservation(_ context.Context, id, memberID int64, now time.Time) error {
	for i := range r.reservations {
		res := &r.reservations[i]
		if res.ID == id && res.MemberID == memberID && res.Status == StatusBooked {
			res.Status = StatusCancelled
			res.UpdatedAt = now
			return nil
		}
	}
	return apperr.Conflict("reservation cannot be cancelled")
}

func (r *memRepo) CreateWaitlistEntry(_ context.Context, w WaitlistEntry) (int64, error) {
	r.nextID++
	w.ID = r.nextID
	r.waitlist = append(r.waitlist, w)
	return w.ID, nil
}

func (r *memRepo) ArriveReservation(_ context.Context, reservationID, storeID int64, _ string, _ int64, now time.Time) error {
	for i := range r.reservations {
		res := &r.reservations[i]
		if res.ID == reservationID && res.StoreID == storeID && res.Status == StatusBooked {
			res.Status = StatusArrived
			res.UpdatedAt = now
			r.arrivals++
			return nil
		}
	}
	return apperr.Conflict("reservation cannot be marked arrived")
}

func newTestService() (*Service, *memRepo) {
	repo := &memRepo{}
	svc := NewService(repo)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	return svc, repo
}

func validCreateReq() CreateReservationRequest {
	return CreateReservationRequest{
		StoreID:    1,
		PartySize:  2,
		ReservedAt: time.Date(2026, 7, 18, 19, 0, 0, 0, time.UTC),
		Remark:     "window seat",
	}
}

func TestCreateReservationBooks(t *testing.T) {
	svc, repo := newTestService()
	view, err := svc.CreateReservation(context.Background(), 42, validCreateReq())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Status != StatusBooked {
		t.Fatalf("expected booked, got %s", view.Status)
	}
	if view.ReservationNo == "" {
		t.Fatal("expected a reservation number")
	}
	if len(repo.reservations) != 1 || repo.reservations[0].MemberID != 42 {
		t.Fatalf("reservation not persisted for member")
	}
}

func TestCreateReservationValidation(t *testing.T) {
	svc, _ := newTestService()
	cases := map[string]CreateReservationRequest{
		"missing store": {PartySize: 2, ReservedAt: time.Date(2026, 7, 18, 19, 0, 0, 0, time.UTC)},
		"zero party":    {StoreID: 1, ReservedAt: time.Date(2026, 7, 18, 19, 0, 0, 0, time.UTC)},
		"missing time":  {StoreID: 1, PartySize: 2},
		"past reserved": {StoreID: 1, PartySize: 2, ReservedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	for name, req := range cases {
		if _, err := svc.CreateReservation(context.Background(), 1, req); apperr.From(err).Code != apperr.CodeInvalidArgument {
			t.Fatalf("%s: expected INVALID_ARGUMENT, got %v", name, err)
		}
	}
}

func TestGetReservationOwnership(t *testing.T) {
	svc, _ := newTestService()
	view, _ := svc.CreateReservation(context.Background(), 42, validCreateReq())

	if _, err := svc.GetReservation(context.Background(), 42, view.ID); err != nil {
		t.Fatalf("owner get: %v", err)
	}
	// A different member must not see the reservation.
	if _, err := svc.GetReservation(context.Background(), 99, view.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for non-owner, got %v", err)
	}
}

func TestCancelReservation(t *testing.T) {
	svc, _ := newTestService()
	view, _ := svc.CreateReservation(context.Background(), 42, validCreateReq())

	if err := svc.CancelReservation(context.Background(), 42, view.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	// Second cancel must conflict (already cancelled).
	if err := svc.CancelReservation(context.Background(), 42, view.ID); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected CONFLICT on double cancel, got %v", err)
	}
}

func TestListReservations(t *testing.T) {
	svc, _ := newTestService()
	_, _ = svc.CreateReservation(context.Background(), 42, validCreateReq())
	_, _ = svc.CreateReservation(context.Background(), 42, validCreateReq())
	_, _ = svc.CreateReservation(context.Background(), 7, validCreateReq())

	views, total, err := svc.ListReservations(context.Background(), 42, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(views) != 2 {
		t.Fatalf("expected 2 reservations for member 42, got %d", total)
	}
}

func TestStoreReservationScope(t *testing.T) {
	svc, _ := newTestService()
	store2Req := validCreateReq()
	store2Req.StoreID = 2

	own, _ := svc.CreateReservation(context.Background(), 42, validCreateReq()) // store 1
	_, _ = svc.CreateReservation(context.Background(), 7, validCreateReq())     // store 1
	other, _ := svc.CreateReservation(context.Background(), 42, store2Req)      // store 2

	// A store sees only its own bookings, regardless of the owning member.
	views, total, err := svc.ListStoreReservations(context.Background(), 1, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(views) != 2 {
		t.Fatalf("expected 2 reservations for store 1, got %d", total)
	}

	// Detail is scoped to the store: own booking visible, another store's hidden.
	if _, err := svc.GetStoreReservation(context.Background(), 1, own.ID); err != nil {
		t.Fatalf("store get own: %v", err)
	}
	if _, err := svc.GetStoreReservation(context.Background(), 1, other.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for another store's booking, got %v", err)
	}
}

func TestCreateWaitlistEntry(t *testing.T) {
	svc, repo := newTestService()
	view, err := svc.CreateWaitlistEntry(context.Background(), 42, CreateWaitlistRequest{StoreID: 1, PartySize: 3})
	if err != nil {
		t.Fatalf("waitlist: %v", err)
	}
	if view.Status != WaitlistWaiting || len(repo.waitlist) != 1 {
		t.Fatalf("waitlist entry not queued: %+v", view)
	}
	if _, err := svc.CreateWaitlistEntry(context.Background(), 42, CreateWaitlistRequest{StoreID: 1, PartySize: 0}); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for zero party, got %v", err)
	}
}

func TestArriveReservation(t *testing.T) {
	svc, repo := newTestService()
	view, _ := svc.CreateReservation(context.Background(), 42, validCreateReq())

	if err := svc.ArriveReservation(context.Background(), 1, view.ID, "cashier", 5); err != nil {
		t.Fatalf("arrive: %v", err)
	}
	if repo.arrivals != 1 || repo.reservations[0].Status != StatusArrived {
		t.Fatalf("arrival not recorded")
	}
	// Wrong store scope must not be able to arrive the booking.
	other, _ := svc.CreateReservation(context.Background(), 42, validCreateReq())
	if err := svc.ArriveReservation(context.Background(), 999, other.ID, "cashier", 5); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected CONFLICT for wrong store scope, got %v", err)
	}
}

func TestListTablesAndSeats(t *testing.T) {
	svc, repo := newTestService()
	repo.tables = []Table{{ID: 1, StoreID: 1, Name: "T1", Capacity: 4, Status: AvailabilityAvailable}, {ID: 2, StoreID: 2}}
	tableID := int64(1)
	repo.seats = []Seat{{ID: 1, StoreID: 1, TableID: &tableID, Name: "S1", Status: AvailabilityAvailable}, {ID: 2, StoreID: 2}}

	tables, err := svc.ListTables(context.Background(), 1)
	if err != nil || len(tables) != 1 || tables[0].Name != "T1" {
		t.Fatalf("expected 1 table for store 1, got %+v (%v)", tables, err)
	}
	seats, err := svc.ListSeats(context.Background(), 1)
	if err != nil || len(seats) != 1 || seats[0].TableID == nil {
		t.Fatalf("expected 1 seat for store 1, got %+v (%v)", seats, err)
	}
}
