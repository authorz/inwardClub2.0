package reservation

import (
	"context"
	"fmt"
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

type fakeAssetResolver struct{}

func (fakeAssetResolver) PublicURLByID(_ context.Context, id int64) (string, error) {
	return fmt.Sprintf("https://cdn.example.com/assets/%d", id), nil
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

func (r *memRepo) ListSeats(_ context.Context, storeID int64, _ time.Time) ([]Seat, error) {
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

func (r *memRepo) HasMemberReservation(
	_ context.Context,
	memberID int64,
	createdFrom, createdBefore time.Time,
) (bool, error) {
	for _, res := range r.reservations {
		if res.MemberID == memberID &&
			!res.CreatedAt.Before(createdFrom) &&
			res.CreatedAt.Before(createdBefore) {
			return true, nil
		}
	}
	return false, nil
}

func (r *memRepo) CreateReservation(
	_ context.Context,
	res Reservation,
	dailyStart, dailyEnd time.Time,
) (int64, error) {
	if exists, _ := r.HasMemberReservation(context.Background(), res.MemberID, dailyStart, dailyEnd); exists {
		return 0, apperr.Conflict("你今天已经预约座位了")
	}
	if res.SeatID != nil {
		for _, existing := range r.reservations {
			if existing.SeatID != nil && *existing.SeatID == *res.SeatID && existing.Status == StatusBooked {
				return 0, apperr.Conflict("seat is already reserved")
			}
		}
	}
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
	svc := NewService(repo, fakeAssetResolver{}, time.UTC)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	return svc, repo
}

func TestLatestSeatResetUsesBusinessFourAM(t *testing.T) {
	svc, _ := newTestService()
	svc.location = time.FixedZone("CST", 8*60*60)
	svc.now = func() time.Time {
		return time.Date(2026, 7, 18, 3, 0, 0, 0, svc.location)
	}
	want := time.Date(2026, 7, 17, 4, 0, 0, 0, svc.location).UTC()
	if got := svc.latestSeatReset(); !got.Equal(want) {
		t.Fatalf("latest reset: got %s, want %s", got, want)
	}
}

func TestListTablesIncludesCustomLayoutURL(t *testing.T) {
	svc, repo := newTestService()
	layoutAssetID := int64(23)
	repo.tables = []Table{{
		ID: 1, StoreID: 8, Name: "A1", Capacity: 9,
		LayoutAssetID: &layoutAssetID, Status: AvailabilityAvailable,
	}}

	views, err := svc.ListTables(context.Background(), 8)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected one table, got %d", len(views))
	}
	if views[0].LayoutURL != "https://cdn.example.com/assets/23" {
		t.Fatalf("unexpected layout URL: %q", views[0].LayoutURL)
	}
}

func TestListSeatsIncludesReservedMemberProfile(t *testing.T) {
	svc, repo := newTestService()
	tableID := int64(3)
	repo.seats = []Seat{{
		ID: 7, StoreID: 8, TableID: &tableID, Name: "1号位",
		Status: AvailabilityReserved, MemberNickname: "会员甲",
		MemberAvatarURL: "https://cdn.example.com/avatar.jpg", MemberGender: "female",
	}}

	views, err := svc.ListSeats(context.Background(), 8)
	if err != nil {
		t.Fatalf("list seats: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected one seat, got %d", len(views))
	}
	if views[0].Status != AvailabilityReserved ||
		views[0].AvatarURL != "https://cdn.example.com/avatar.jpg" ||
		views[0].Gender != "female" ||
		views[0].Nickname != "会员甲" {
		t.Fatalf("unexpected reserved seat view: %+v", views[0])
	}
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

func TestCreateReservationRejectsOccupiedSeat(t *testing.T) {
	svc, _ := newTestService()
	seatID := int64(9)
	first := validCreateReq()
	first.SeatID = &seatID
	if _, err := svc.CreateReservation(context.Background(), 42, first); err != nil {
		t.Fatalf("first reservation: %v", err)
	}

	second := validCreateReq()
	second.SeatID = &seatID
	if _, err := svc.CreateReservation(context.Background(), 43, second); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected conflict for occupied seat, got %v", err)
	}
}

func TestCreateReservationReturnsDailyLimitBeforePastTime(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.CreateReservation(context.Background(), 42, validCreateReq()); err != nil {
		t.Fatalf("first reservation: %v", err)
	}

	second := validCreateReq()
	second.ReservedAt = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.CreateReservation(context.Background(), 42, second)
	appErr := apperr.From(err)
	if appErr.Code != apperr.CodeConflict || appErr.Message != "你今天已经预约座位了" {
		t.Fatalf("unexpected daily limit error: code=%s message=%q", appErr.Code, appErr.Message)
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
	svc, repo := newTestService()
	repo.reservations = []Reservation{
		{ID: 1, MemberID: 42, StoreID: 1, Status: StatusBooked},
		{ID: 2, MemberID: 42, StoreID: 1, Status: StatusExpired},
		{ID: 3, MemberID: 7, StoreID: 1, Status: StatusBooked},
	}

	views, total, err := svc.ListReservations(context.Background(), 42, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(views) != 2 {
		t.Fatalf("expected 2 reservations for member 42, got %d", total)
	}
}

func TestStoreReservationScope(t *testing.T) {
	svc, repo := newTestService()
	own := Reservation{ID: 1, MemberID: 42, StoreID: 1, Status: StatusBooked}
	other := Reservation{ID: 3, MemberID: 42, StoreID: 2, Status: StatusBooked}
	repo.reservations = []Reservation{
		own,
		{ID: 2, MemberID: 7, StoreID: 1, Status: StatusBooked},
		other,
	}

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
