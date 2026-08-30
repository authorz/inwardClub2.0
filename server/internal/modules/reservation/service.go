package reservation

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	inputvalidation "github.com/inwardclub/server/internal/platform/validation"
)

// Service provides reservation, waitlist and arrival operations for both the
// mini program (member scope) and the store console (store scope).
type Service struct {
	repo     Repository
	assets   AssetResolver
	location *time.Location
	now      func() time.Time
}

const waitlistAvatarLimit = 16

// NewService builds the reservation service.
func NewService(repo Repository, assets AssetResolver, location *time.Location) *Service {
	if location == nil {
		location = time.UTC
	}
	return &Service{
		repo: repo, assets: assets, location: location, now: time.Now,
	}
}

// ListTables returns the store's tables for the availability view.
func (s *Service) ListTables(ctx context.Context, storeID int64) ([]TableView, error) {
	tables, err := s.repo.ListTables(ctx, storeID)
	if err != nil {
		return nil, err
	}
	views := make([]TableView, 0, len(tables))
	for _, t := range tables {
		views = append(views, tableView(t))
	}
	return views, nil
}

// ListSeats returns the store's seats for the availability view.
func (s *Service) ListSeats(ctx context.Context, storeID int64) ([]SeatView, error) {
	seats, err := s.repo.ListSeats(ctx, storeID, s.latestSeatReset())
	if err != nil {
		return nil, err
	}
	views := make([]SeatView, 0, len(seats))
	for _, seat := range seats {
		views = append(views, seatView(seat))
	}
	return views, nil
}

func (s *Service) latestSeatReset() time.Time {
	now := s.now().In(s.location)
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), seatResetHour, 0, 0, 0, s.location)
	if now.Before(cutoff) {
		cutoff = cutoff.AddDate(0, 0, -1)
	}
	return cutoff.UTC()
}

// ListReservations returns the member's reservations, most recent first.
func (s *Service) ListReservations(ctx context.Context, memberID int64, page httpx.Page) ([]ReservationView, int64, error) {
	items, total, err := s.repo.ListMemberReservations(ctx, memberID, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]ReservationView, 0, len(items))
	for _, r := range items {
		view := reservationView(r)
		if view.MemberAvatarURL == "" && r.MemberAvatarAssetID != nil && s.assets != nil {
			view.MemberAvatarURL, _ = s.assets.PublicURLByID(ctx, *r.MemberAvatarAssetID)
		}
		views = append(views, view)
	}
	return views, total, nil
}

// ListStoreReservations returns the acting store's reservations, most recent
// first (store console). storeID is the store scope pinned from the token.
func (s *Service) ListStoreReservations(ctx context.Context, storeID int64, filter StoreReservationFilter, page httpx.Page) ([]ReservationView, int64, error) {
	items, total, err := s.repo.ListStoreReservations(ctx, storeID, filter, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]ReservationView, 0, len(items))
	for _, r := range items {
		view := reservationView(r)
		if view.MemberAvatarURL == "" && r.MemberAvatarAssetID != nil && s.assets != nil {
			view.MemberAvatarURL, _ = s.assets.PublicURLByID(ctx, *r.MemberAvatarAssetID)
		}
		views = append(views, view)
	}
	return views, total, nil
}

// CreateReservation books a table/seat for the member.
func (s *Service) CreateReservation(ctx context.Context, memberID int64, req CreateReservationRequest) (ReservationView, error) {
	return s.CreateReservationForActor(ctx, memberID, false, req)
}

// CreateReservationForActor records whether this booking was made from a
// reservation-only pre-registration session. Public seat reads use that flag
// to show the generic brand identity instead of exposing a member profile.
func (s *Service) CreateReservationForActor(ctx context.Context, memberID int64, bookedAsGuest bool, req CreateReservationRequest) (ReservationView, error) {
	if req.StoreID <= 0 {
		return ReservationView{}, apperr.Invalid("storeId is required")
	}
	if req.PartySize <= 0 {
		return ReservationView{}, apperr.Invalid("partySize must be positive")
	}
	if req.PartySize > 50 {
		return ReservationView{}, apperr.Invalid("预约人数不能超过50人")
	}
	remark, validationErr := inputvalidation.PlainText(req.Remark, inputvalidation.TextOptions{
		Label: "预约备注", MaxRunes: 200, AllowEmpty: true, AllowNewlines: true,
	})
	if validationErr != nil {
		return ReservationView{}, apperr.Invalid(validationErr.Error())
	}
	if req.TableID == nil || *req.TableID <= 0 {
		return ReservationView{}, apperr.Invalid("tableId is required")
	}
	dailyStart, dailyEnd := s.reservationDay()
	exists, err := s.repo.HasMemberReservation(ctx, memberID, dailyStart, dailyEnd)
	if err != nil {
		return ReservationView{}, err
	}
	if exists {
		return ReservationView{}, apperr.Conflict("你今天已经预约座位了")
	}
	now := s.now().UTC()
	res := Reservation{
		ReservationNo: s.newReservationNo(now),
		StoreID:       req.StoreID,
		MemberID:      memberID,
		BookedAsGuest: bookedAsGuest,
		TableID:       req.TableID,
		// Seat selection is no longer exposed to members. Keep the request field
		// backward-compatible, but always let the repository allocate a current
		// free seat atomically so stale clients cannot submit an occupied seat.
		SeatID:    nil,
		PartySize: req.PartySize,
		// reserved_at is retained for schema/API compatibility. A seat booking has
		// no member-selected arrival time; it mirrors the server creation time.
		ReservedAt: now,
		Status:     StatusBooked,
		Remark:     remark,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	id, err := s.repo.CreateReservation(ctx, res, dailyStart, dailyEnd)
	if err != nil {
		return ReservationView{}, err
	}
	persisted, err := s.repo.GetReservation(ctx, id)
	if err != nil {
		return ReservationView{}, err
	}
	return reservationView(persisted), nil
}

func (s *Service) reservationDay() (time.Time, time.Time) {
	start := s.latestSeatReset()
	end := start.In(s.location).AddDate(0, 0, 1)
	return start, end.UTC()
}

// GetReservation returns a single reservation owned by the member.
func (s *Service) GetReservation(ctx context.Context, memberID, id int64) (ReservationView, error) {
	res, err := s.repo.GetReservation(ctx, id)
	if err != nil {
		return ReservationView{}, err
	}
	if res.MemberID != memberID {
		// Do not disclose existence of another member's reservation.
		return ReservationView{}, apperr.NotFound("reservation not found")
	}
	return reservationView(res), nil
}

// GetStoreReservation returns a single reservation within the acting store's
// scope (store console). A booking outside the scope is reported as not found,
// mirroring the member-scoped GetReservation.
func (s *Service) GetStoreReservation(ctx context.Context, storeID, id int64) (ReservationView, error) {
	res, err := s.repo.GetReservation(ctx, id)
	if err != nil {
		return ReservationView{}, err
	}
	if res.StoreID != storeID {
		return ReservationView{}, apperr.NotFound("reservation not found")
	}
	return reservationView(res), nil
}

// CancelReservation cancels a member's still-booked reservation.
func (s *Service) CancelReservation(ctx context.Context, memberID, id int64) error {
	dailyStart, _ := s.reservationDay()
	return s.repo.CancelReservation(ctx, id, memberID, dailyStart)
}

// CancelStoreReservation releases an active booking owned by the acting store.
func (s *Service) CancelStoreReservation(ctx context.Context, storeID, id int64) error {
	dailyStart, _ := s.reservationDay()
	return s.repo.CancelStoreReservation(ctx, id, storeID, dailyStart)
}

// CreateWaitlistEntry queues a walk-in party for the member.
func (s *Service) CreateWaitlistEntry(ctx context.Context, memberID int64, req CreateWaitlistRequest) (WaitlistEntryView, error) {
	if req.StoreID <= 0 {
		return WaitlistEntryView{}, apperr.Invalid("storeId is required")
	}
	if req.PartySize <= 0 {
		return WaitlistEntryView{}, apperr.Invalid("partySize must be positive")
	}
	if req.PartySize > 50 {
		return WaitlistEntryView{}, apperr.Invalid("排队人数不能超过50人")
	}
	now := s.now().UTC()
	entry := WaitlistEntry{
		StoreID:   req.StoreID,
		MemberID:  memberID,
		PartySize: req.PartySize,
		Status:    WaitlistWaiting,
		QueuedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	dailyStart, dailyEnd := s.reservationDay()
	id, err := s.repo.CreateWaitlistEntry(ctx, entry, dailyStart, dailyEnd)
	if err != nil {
		return WaitlistEntryView{}, err
	}
	entry.ID = id
	return waitlistView(entry), nil
}

// ListWaitlistAvatars returns a compact, privacy-limited visual preview of the
// current reservation day's unique waiting members for one store.
func (s *Service) ListWaitlistAvatars(ctx context.Context, storeID int64) ([]WaitlistAvatarView, error) {
	if storeID <= 0 {
		return nil, apperr.Invalid("storeId is required")
	}
	entries, err := s.repo.ListWaitingMembers(ctx, storeID, s.latestSeatReset(), waitlistAvatarLimit)
	if err != nil {
		return nil, err
	}
	views := make([]WaitlistAvatarView, 0, len(entries))
	for _, entry := range entries {
		if entry.MemberAvatarURL == "" && entry.MemberAvatarAssetID != nil && s.assets != nil {
			entry.MemberAvatarURL, _ = s.assets.PublicURLByID(ctx, *entry.MemberAvatarAssetID)
		}
		views = append(views, waitlistAvatarView(entry))
	}
	return views, nil
}

// ArriveReservation records a member's arrival against a booking (store console).
// storeID is the acting store's scope; byType/byID identify the staff member.
func (s *Service) ArriveReservation(ctx context.Context, storeID, reservationID int64, byType string, byID int64) error {
	return s.repo.ArriveReservation(ctx, reservationID, storeID, byType, byID, s.now().UTC())
}

// newReservationNo builds a human-readable, collision-resistant booking number.
func (s *Service) newReservationNo(now time.Time) string {
	return fmt.Sprintf("R%s%04d", now.Format("20060102150405"), rand.Intn(10000))
}
