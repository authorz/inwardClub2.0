package reservation

import (
	"context"
	"fmt"
	"strings"
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
	dailyClaims  map[string]bool
}

type fakeAssetResolver struct{}

type closedReservationWindow struct{}

func (closedReservationWindow) ValidateReservationTime(context.Context, int64) error {
	return apperr.Conflict("今日预约已截止")
}

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
		if res.MemberID == memberID && (res.Status == StatusBooked || res.Status == StatusArrived) {
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

func (r *memRepo) ListStoreReservations(_ context.Context, storeID int64, filter StoreReservationFilter, limit, offset int) ([]Reservation, int64, error) {
	matches := func(value, query string) bool {
		return query == "" || strings.Contains(strings.ToLower(value), strings.ToLower(query))
	}
	var all []Reservation
	for _, res := range r.reservations {
		if res.StoreID == storeID && (res.Status == StatusBooked || res.Status == StatusArrived) &&
			matches(res.TableNo, filter.TableNo) && matches(res.SeatNo, filter.SeatNo) &&
			matches(res.MemberNickname, filter.MemberNickname) && matches(res.MemberPhone, filter.MemberPhone) {
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
	if r.dailyClaims[fmt.Sprintf("%d/%s", memberID, createdFrom.UTC().Format(time.RFC3339))] {
		return true, nil
	}
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
	if res.SeatID == nil {
		for i := range r.seats {
			seat := r.seats[i]
			if seat.StoreID != res.StoreID || seat.TableID == nil || res.TableID == nil ||
				*seat.TableID != *res.TableID || seat.Status != AvailabilityAvailable {
				continue
			}
			occupied := false
			for _, existing := range r.reservations {
				if existing.SeatID != nil && *existing.SeatID == seat.ID &&
					(existing.Status == StatusBooked || existing.Status == StatusArrived) &&
					!existing.CreatedAt.Before(dailyStart) && existing.CreatedAt.Before(dailyEnd) {
					occupied = true
					break
				}
			}
			if !occupied {
				seatID := seat.ID
				res.SeatID = &seatID
				break
			}
		}
		if res.SeatID == nil {
			return 0, apperr.Conflict("该桌暂时没有空位")
		}
	}
	if res.SeatID != nil {
		for _, existing := range r.reservations {
			if existing.SeatID != nil && *existing.SeatID == *res.SeatID &&
				(existing.Status == StatusBooked || existing.Status == StatusArrived) &&
				!existing.CreatedAt.Before(dailyStart) && existing.CreatedAt.Before(dailyEnd) {
				return 0, apperr.Conflict("seat is already reserved")
			}
		}
	}
	if r.dailyClaims == nil {
		r.dailyClaims = make(map[string]bool)
	}
	r.dailyClaims[fmt.Sprintf("%d/%s", res.MemberID, dailyStart.UTC().Format(time.RFC3339))] = true
	r.nextID++
	res.ID = r.nextID
	r.reservations = append(r.reservations, res)
	for i := range r.waitlist {
		entry := &r.waitlist[i]
		if entry.StoreID == res.StoreID && entry.MemberID == res.MemberID &&
			(entry.Status == WaitlistWaiting || entry.Status == WaitlistCalled) &&
			!entry.QueuedAt.Before(dailyStart) && entry.QueuedAt.Before(dailyEnd) {
			entry.Status = WaitlistLeft
			entry.UpdatedAt = res.UpdatedAt
		}
	}
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

func (r *memRepo) CancelReservation(_ context.Context, id, memberID int64, dailyStart time.Time) error {
	for i := range r.reservations {
		res := r.reservations[i]
		if res.ID != id {
			continue
		}
		if res.MemberID != memberID {
			return apperr.NotFound("reservation not found")
		}
		if res.Status != StatusBooked {
			return apperr.Conflict("已到店的预约不能取消")
		}
		r.reservations = append(r.reservations[:i], r.reservations[i+1:]...)
		delete(r.dailyClaims, fmt.Sprintf("%d/%s", memberID, dailyStart.UTC().Format(time.RFC3339)))
		return nil
	}
	return nil
}

func (r *memRepo) CancelStoreReservation(_ context.Context, id, storeID int64, dailyStart time.Time) error {
	for i := range r.reservations {
		res := r.reservations[i]
		if res.ID == id && res.StoreID == storeID && res.Status == StatusBooked {
			r.reservations = append(r.reservations[:i], r.reservations[i+1:]...)
			delete(r.dailyClaims, fmt.Sprintf("%d/%s", res.MemberID, dailyStart.UTC().Format(time.RFC3339)))
			return nil
		}
	}
	return apperr.Conflict("reservation cannot be cancelled")
}

func (r *memRepo) ListWaitingMembers(_ context.Context, storeID int64, queuedFrom time.Time, limit int) ([]WaitlistEntry, error) {
	seen := make(map[int64]bool)
	var entries []WaitlistEntry
	for _, entry := range r.waitlist {
		if entry.StoreID != storeID || entry.Status != WaitlistWaiting || entry.QueuedAt.Before(queuedFrom) || seen[entry.MemberID] {
			continue
		}
		seen[entry.MemberID] = true
		entries = append(entries, entry)
		if len(entries) == limit {
			break
		}
	}
	return entries, nil
}

func (r *memRepo) CreateWaitlistEntry(_ context.Context, w WaitlistEntry, dailyStart, dailyEnd time.Time) (int64, error) {
	if exists, _ := r.HasMemberReservation(context.Background(), w.MemberID, dailyStart, dailyEnd); exists {
		return 0, apperr.Conflict("你已经预约座位了，如需排队请先取消预约")
	}
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
	tableID := int64(1)
	repo := &memRepo{
		dailyClaims: make(map[string]bool),
		seats: []Seat{{
			ID: 1, StoreID: 1, TableID: &tableID, Status: AvailabilityAvailable,
		}},
	}
	svc := NewService(repo, fakeAssetResolver{}, time.UTC, nil)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	return svc, repo
}

func TestLatestSeatResetUsesCurrentMidnight(t *testing.T) {
	svc, _ := newTestService()
	svc.location = time.FixedZone("CST", 8*60*60)
	svc.now = func() time.Time {
		return time.Date(2026, 7, 18, 3, 0, 0, 0, svc.location)
	}
	want := time.Date(2026, 7, 18, 0, 0, 0, 0, svc.location).UTC()
	if got := svc.latestSeatReset(); !got.Equal(want) {
		t.Fatalf("latest reset: got %s, want %s", got, want)
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
	tableID := int64(1)
	seatID := int64(1)
	return CreateReservationRequest{
		StoreID:   1,
		TableID:   &tableID,
		SeatID:    &seatID,
		PartySize: 2,
		Remark:    "window seat",
	}
}

func ptrInt64(value int64) *int64 { return &value }

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
	if !repo.reservations[0].ReservedAt.Equal(svc.now().UTC()) {
		t.Fatalf("reserved_at must mirror server creation time, got %s", repo.reservations[0].ReservedAt)
	}
}

func TestReservationWindowBlocksReservationAndWaitlist(t *testing.T) {
	svc, repo := newTestService()
	svc.policy = closedReservationWindow{}

	if _, err := svc.CreateReservation(context.Background(), 42, validCreateReq()); apperr.From(err).Message != "今日预约已截止" {
		t.Fatalf("reservation error = %v", err)
	}
	if _, err := svc.CreateWaitlistEntry(context.Background(), 42, CreateWaitlistRequest{StoreID: 1, PartySize: 1}); apperr.From(err).Message != "今日预约已截止" {
		t.Fatalf("waitlist error = %v", err)
	}
	if len(repo.reservations) != 0 || len(repo.waitlist) != 0 {
		t.Fatalf("closed window persisted data: reservations=%d waitlist=%d", len(repo.reservations), len(repo.waitlist))
	}
}

func TestCreateReservationLeavesCurrentWaitlistForSameStore(t *testing.T) {
	svc, repo := newTestService()
	dailyStart, _ := svc.reservationDay()
	repo.waitlist = []WaitlistEntry{
		{ID: 1, StoreID: 1, MemberID: 42, Status: WaitlistWaiting, QueuedAt: dailyStart.Add(time.Hour)},
		{ID: 2, StoreID: 1, MemberID: 42, Status: WaitlistCalled, QueuedAt: dailyStart.Add(2 * time.Hour)},
		{ID: 3, StoreID: 2, MemberID: 42, Status: WaitlistWaiting, QueuedAt: dailyStart.Add(time.Hour)},
		{ID: 4, StoreID: 1, MemberID: 7, Status: WaitlistWaiting, QueuedAt: dailyStart.Add(time.Hour)},
		{ID: 5, StoreID: 1, MemberID: 42, Status: WaitlistWaiting, QueuedAt: dailyStart.Add(-time.Hour)},
	}

	if _, err := svc.CreateReservation(context.Background(), 42, validCreateReq()); err != nil {
		t.Fatalf("create reservation: %v", err)
	}

	wantStatuses := []string{WaitlistLeft, WaitlistLeft, WaitlistWaiting, WaitlistWaiting, WaitlistWaiting}
	for i, want := range wantStatuses {
		if got := repo.waitlist[i].Status; got != want {
			t.Fatalf("waitlist entry %d status = %q, want %q", repo.waitlist[i].ID, got, want)
		}
	}
}

func TestHistoricalGuestReservationPreservesGenericSeatIdentity(t *testing.T) {
	svc, repo := newTestService()
	repo.seats = []Seat{{
		ID: 1, StoreID: 1, Status: AvailabilityReserved, BookedAsGuest: true,
		MemberNickname: "inward会员",
	}}
	seats, err := svc.ListSeats(context.Background(), 1)
	if err != nil {
		t.Fatalf("list guest seat: %v", err)
	}
	if len(seats) != 1 || !seats[0].IsGuest || seats[0].Nickname != "inward会员" {
		t.Fatalf("unexpected guest seat view: %+v", seats)
	}
}

func TestCreateReservationIgnoresStaleClientSeatSelection(t *testing.T) {
	svc, repo := newTestService()
	tableID := int64(1)
	repo.seats = append(repo.seats, Seat{
		ID: 2, StoreID: 1, TableID: &tableID, Status: AvailabilityAvailable,
	})
	repo.reservations = []Reservation{{
		ID: 1, StoreID: 1, MemberID: 7, TableID: &tableID, SeatID: ptrInt64(1),
		Status: StatusBooked, CreatedAt: svc.now().UTC(),
	}}
	repo.nextID = 1

	req := validCreateReq()
	req.SeatID = ptrInt64(1)
	view, err := svc.CreateReservation(context.Background(), 42, req)
	if err != nil {
		t.Fatalf("create with stale requested seat: %v", err)
	}
	if view.SeatID == nil || *view.SeatID != 2 {
		t.Fatalf("expected automatic assignment to seat 2, got %+v", view.SeatID)
	}
}

func TestCreateReservationAutomaticallyAssignsAvailableSeat(t *testing.T) {
	svc, repo := newTestService()
	tableID := int64(1)
	repo.seats = []Seat{
		{ID: 1, StoreID: 1, TableID: &tableID, Status: AvailabilityAvailable},
		{ID: 2, StoreID: 1, TableID: &tableID, Status: AvailabilityAvailable},
	}
	repo.reservations = []Reservation{{
		ID: 1, StoreID: 1, MemberID: 7, TableID: &tableID, SeatID: ptrInt64(1),
		Status: StatusBooked, CreatedAt: svc.now().UTC(),
	}}
	repo.nextID = 1

	req := validCreateReq()
	req.SeatID = nil
	view, err := svc.CreateReservation(context.Background(), 42, req)
	if err != nil {
		t.Fatalf("create with automatic seat: %v", err)
	}
	if view.SeatID == nil || *view.SeatID != 2 {
		t.Fatalf("expected seat 2, got %+v", view.SeatID)
	}
}

func TestCreateReservationIgnoresOldSeatBooking(t *testing.T) {
	svc, repo := newTestService()
	tableID := int64(1)
	repo.seats = []Seat{{ID: 1, StoreID: 1, TableID: &tableID, Status: AvailabilityAvailable}}
	repo.reservations = []Reservation{{
		ID: 1, StoreID: 1, MemberID: 7, TableID: &tableID, SeatID: ptrInt64(1),
		Status: StatusBooked, CreatedAt: svc.now().Add(-24 * time.Hour),
	}}
	repo.nextID = 1

	req := validCreateReq()
	req.SeatID = nil
	view, err := svc.CreateReservation(context.Background(), 42, req)
	if err != nil {
		t.Fatalf("create with historical booking: %v", err)
	}
	if view.SeatID == nil || *view.SeatID != 1 {
		t.Fatalf("expected historical booking to release seat 1, got %+v", view.SeatID)
	}
}

func TestCreateReservationAutomaticAssignmentReturnsFull(t *testing.T) {
	svc, repo := newTestService()
	tableID := int64(1)
	repo.seats = []Seat{{ID: 1, StoreID: 1, TableID: &tableID, Status: AvailabilityAvailable}}
	repo.reservations = []Reservation{{
		ID: 1, StoreID: 1, MemberID: 7, TableID: &tableID, SeatID: ptrInt64(1),
		Status: StatusBooked, CreatedAt: svc.now().UTC(),
	}}

	req := validCreateReq()
	req.SeatID = nil
	_, err := svc.CreateReservation(context.Background(), 42, req)
	appErr := apperr.From(err)
	if appErr.Code != apperr.CodeConflict || appErr.Message != "该桌暂时没有空位" {
		t.Fatalf("unexpected full-table error: code=%s message=%q", appErr.Code, appErr.Message)
	}
}

func TestCreateReservationReturnsDailyLimit(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.CreateReservation(context.Background(), 42, validCreateReq()); err != nil {
		t.Fatalf("first reservation: %v", err)
	}

	_, err := svc.CreateReservation(context.Background(), 42, validCreateReq())
	appErr := apperr.From(err)
	if appErr.Code != apperr.CodeConflict || appErr.Message != "你今天已经预约座位了" {
		t.Fatalf("unexpected daily limit error: code=%s message=%q", appErr.Code, appErr.Message)
	}
}

func TestCreateReservationValidation(t *testing.T) {
	svc, _ := newTestService()
	cases := map[string]CreateReservationRequest{
		"missing store": {PartySize: 2},
		"zero party":    {StoreID: 1},
		"missing table": {StoreID: 1, PartySize: 1},
	}
	for name, req := range cases {
		if _, err := svc.CreateReservation(context.Background(), 1, req); apperr.From(err).Code != apperr.CodeInvalidArgument {
			t.Fatalf("%s: expected INVALID_ARGUMENT, got %v", name, err)
		}
	}
}

func TestCreateReservationRejectsUnsafeRemark(t *testing.T) {
	svc, repo := newTestService()
	req := validCreateReq()
	req.Remark = `<svg onload=alert(1)>`
	if _, err := svc.CreateReservation(context.Background(), 1, req); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected unsafe remark rejection, got %v", err)
	}
	if len(repo.reservations) != 0 {
		t.Fatal("unsafe reservation reached repository")
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

	if err := svc.CancelReservation(context.Background(), 99, view.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("another member must not cancel the reservation, got %v", err)
	}
	if err := svc.CancelReservation(context.Background(), 42, view.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := svc.GetReservation(context.Background(), 42, view.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("cancelled reservation must be deleted, got %v", err)
	}
	if _, err := svc.CreateReservation(context.Background(), 42, validCreateReq()); err != nil {
		t.Fatalf("member should be able to reserve again after cancellation, got %v", err)
	}
	// A retry after the first request deleted the row must remain successful.
	if err := svc.CancelReservation(context.Background(), 42, view.ID); err != nil {
		t.Fatalf("repeated cancel must be idempotent, got %v", err)
	}
}

func TestCancelArrivedReservationIsRejected(t *testing.T) {
	svc, repo := newTestService()
	repo.reservations = []Reservation{{ID: 1, MemberID: 42, StoreID: 1, Status: StatusArrived}}
	if err := svc.CancelReservation(context.Background(), 42, 1); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("arrived reservation cancellation = %v", err)
	}
}

func TestListReservations(t *testing.T) {
	svc, repo := newTestService()
	repo.reservations = []Reservation{
		{ID: 1, MemberID: 42, StoreID: 1, Status: StatusBooked},
		{ID: 2, MemberID: 42, StoreID: 1, Status: StatusExpired},
		{ID: 3, MemberID: 7, StoreID: 1, Status: StatusBooked},
		{ID: 4, MemberID: 42, StoreID: 1, Status: StatusArrived},
	}

	views, total, err := svc.ListReservations(context.Background(), 42, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(views) != 2 {
		t.Fatalf("expected booked and arrived reservations for member 42, got %d", total)
	}
}

func TestStoreReservationScope(t *testing.T) {
	svc, repo := newTestService()
	own := Reservation{ID: 1, MemberID: 42, StoreID: 1, Status: StatusBooked}
	other := Reservation{ID: 3, MemberID: 42, StoreID: 2, Status: StatusBooked}
	repo.reservations = []Reservation{
		own,
		{ID: 2, MemberID: 7, StoreID: 1, Status: StatusBooked},
		{ID: 4, MemberID: 8, StoreID: 1, Status: StatusExpired},
		{ID: 5, MemberID: 9, StoreID: 1, Status: StatusArrived},
		other,
	}

	// A store sees only its own bookings, regardless of the owning member.
	views, total, err := svc.ListStoreReservations(context.Background(), 1, StoreReservationFilter{}, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 3 || len(views) != 3 {
		t.Fatalf("expected booked and arrived reservations for store 1, got %d", total)
	}

	repo.reservations[0].TableNo = "A-01"
	repo.reservations[0].SeatNo = "09"
	repo.reservations[0].MemberNickname = "测试会员"
	repo.reservations[0].MemberPhone = "19123450770"
	filtered, filteredTotal, err := svc.ListStoreReservations(context.Background(), 1, StoreReservationFilter{
		TableNo: "-0", SeatNo: "9", MemberNickname: "试会", MemberPhone: "3450",
	}, httpx.Page{Page: 1, PageSize: 20})
	if err != nil || filteredTotal != 1 || len(filtered) != 1 || filtered[0].ID != own.ID {
		t.Fatalf("fuzzy filters did not match the expected reservation: total=%d views=%+v err=%v", filteredTotal, filtered, err)
	}

	// Detail is scoped to the store: own booking visible, another store's hidden.
	if _, err := svc.GetStoreReservation(context.Background(), 1, own.ID); err != nil {
		t.Fatalf("store get own: %v", err)
	}
	if _, err := svc.GetStoreReservation(context.Background(), 1, other.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for another store's booking, got %v", err)
	}

	if err := svc.CancelStoreReservation(context.Background(), 2, own.ID); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("another store must not cancel the booking, got %v", err)
	}
	if err := svc.CancelStoreReservation(context.Background(), 1, own.ID); err != nil {
		t.Fatalf("store cancel own booking: %v", err)
	}
	if _, err := svc.GetStoreReservation(context.Background(), 1, own.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("store-cancelled booking must be deleted, got %v", err)
	}
}

func TestStoreReservationListResolvesMemberAvatarAsset(t *testing.T) {
	svc, repo := newTestService()
	assetID := int64(88)
	repo.reservations = []Reservation{{
		ID: 1, MemberID: 42, StoreID: 1, Status: StatusBooked,
		MemberAvatarAssetID: &assetID,
	}}
	views, _, err := svc.ListStoreReservations(context.Background(), 1, StoreReservationFilter{}, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].MemberAvatarURL != "https://cdn.example.com/assets/88" {
		t.Fatalf("unexpected resolved avatar: %+v", views)
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

func TestCreateWaitlistEntryRejectsMemberWithReservation(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.CreateReservation(context.Background(), 42, validCreateReq()); err != nil {
		t.Fatalf("create reservation: %v", err)
	}
	_, err := svc.CreateWaitlistEntry(context.Background(), 42, CreateWaitlistRequest{StoreID: 1, PartySize: 1})
	appErr := apperr.From(err)
	if appErr.Code != apperr.CodeConflict || appErr.Message != "你已经预约座位了，如需排队请先取消预约" {
		t.Fatalf("waitlist conflict = %#v", appErr)
	}
}

func TestListWaitlistAvatarsUsesCurrentWaitingMembers(t *testing.T) {
	svc, repo := newTestService()
	assetID := int64(88)
	queuedAt := svc.latestSeatReset().Add(time.Hour)
	repo.waitlist = []WaitlistEntry{
		{ID: 1, StoreID: 1, MemberID: 7, MemberAvatarAssetID: &assetID, Status: WaitlistWaiting, QueuedAt: queuedAt},
		{ID: 2, StoreID: 1, MemberID: 7, Status: WaitlistWaiting, QueuedAt: queuedAt.Add(time.Minute)},
		{ID: 3, StoreID: 1, MemberID: 8, MemberAvatarURL: "https://cdn.example.com/direct.png", Status: WaitlistWaiting, QueuedAt: queuedAt},
		{ID: 4, StoreID: 1, MemberID: 9, Status: WaitlistSeated, QueuedAt: queuedAt},
		{ID: 5, StoreID: 2, MemberID: 10, Status: WaitlistWaiting, QueuedAt: queuedAt},
	}
	views, err := svc.ListWaitlistAvatars(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 || views[0].AvatarURL != "https://cdn.example.com/assets/88" || views[1].AvatarURL != "https://cdn.example.com/direct.png" {
		t.Fatalf("unexpected waitlist avatars: %+v", views)
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
