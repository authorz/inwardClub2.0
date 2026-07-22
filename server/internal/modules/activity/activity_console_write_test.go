package activity

import (
	"context"
	"testing"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// statefulConsoleRepo is a functional in-memory ConsoleRepository double used to
// drive the console write paths (create/update/delete) through the service layer
// and observe the result. Scope filtering mirrors the SQL repo: the admin scope
// (nil store) sees every row, a store scope only its own 'store' rows.
type statefulConsoleRepo struct {
	activities  map[int64]Activity
	sessions    map[int64]Session
	ticketTypes map[int64]TicketType
	seq         int64
}

func newStatefulConsoleRepo() *statefulConsoleRepo {
	return &statefulConsoleRepo{
		activities:  map[int64]Activity{},
		sessions:    map[int64]Session{},
		ticketTypes: map[int64]TicketType{},
		seq:         100,
	}
}

func (r *statefulConsoleRepo) nextID() int64 { r.seq++; return r.seq }

func (r *statefulConsoleRepo) inScope(scope ConsoleScope, a Activity) bool {
	if scope.StoreID == nil {
		return true
	}
	return a.ScopeType == "store" && a.StoreID != nil && *a.StoreID == *scope.StoreID
}

func (r *statefulConsoleRepo) ListActivities(_ context.Context, scope ConsoleScope, _ httpx.Page) ([]Activity, int64, error) {
	out := []Activity{}
	for _, a := range r.activities {
		if r.inScope(scope, a) {
			out = append(out, a)
		}
	}
	return out, int64(len(out)), nil
}

func (r *statefulConsoleRepo) GetActivity(_ context.Context, scope ConsoleScope, id int64) (Activity, error) {
	a, ok := r.activities[id]
	if !ok || !r.inScope(scope, a) {
		return Activity{}, apperr.NotFound("activity not found")
	}
	return a, nil
}

func (r *statefulConsoleRepo) CreateActivity(_ context.Context, scope ConsoleScope, in ActivityInput) (Activity, error) {
	a := Activity{
		ID:          r.nextID(),
		StoreID:     scope.StoreID,
		Title:       in.Title,
		Description: in.Description,
		Content:     in.Content,
		AssetID:     in.AssetID,
		StartAt:     in.StartAt,
		EndAt:       in.EndAt,
		PayChannels: in.PayChannels,
		Status:      in.Status,
	}
	if scope.StoreID == nil {
		a.ScopeType = "global"
	} else {
		a.ScopeType = "store"
	}
	r.activities[a.ID] = a
	return a, nil
}

func (r *statefulConsoleRepo) UpdateActivity(_ context.Context, scope ConsoleScope, id int64, in ActivityInput) (Activity, error) {
	a, ok := r.activities[id]
	if !ok || !r.inScope(scope, a) {
		return Activity{}, apperr.NotFound("activity not found")
	}
	a.Title = in.Title
	a.Status = in.Status
	r.activities[id] = a
	return a, nil
}

func (r *statefulConsoleRepo) DeleteActivity(_ context.Context, scope ConsoleScope, id int64) error {
	a, ok := r.activities[id]
	if !ok || !r.inScope(scope, a) {
		return apperr.NotFound("activity not found")
	}
	delete(r.activities, id)
	return nil
}

func (r *statefulConsoleRepo) ListSessions(_ context.Context, activityID int64) ([]Session, error) {
	out := []Session{}
	for _, s := range r.sessions {
		if s.ActivityID == activityID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *statefulConsoleRepo) GetSession(_ context.Context, activityID, sessionID int64) (Session, error) {
	s, ok := r.sessions[sessionID]
	if !ok || s.ActivityID != activityID {
		return Session{}, apperr.NotFound("activity session not found")
	}
	return s, nil
}

func (r *statefulConsoleRepo) CreateSession(_ context.Context, activityID int64, in SessionInput) (Session, error) {
	// store_id is inherited from the owning activity, mirroring the SQL repo.
	s := Session{ID: r.nextID(), ActivityID: activityID, Name: in.Name, StartAt: in.StartAt, EndAt: in.EndAt, Status: in.Status}
	if a, ok := r.activities[activityID]; ok {
		s.StoreID = a.StoreID
	}
	r.sessions[s.ID] = s
	return s, nil
}

func (r *statefulConsoleRepo) UpdateSession(_ context.Context, activityID, sessionID int64, in SessionInput) (Session, error) {
	s, ok := r.sessions[sessionID]
	if !ok || s.ActivityID != activityID {
		return Session{}, apperr.NotFound("activity session not found")
	}
	s.Name = in.Name
	s.Status = in.Status
	r.sessions[sessionID] = s
	return s, nil
}

func (r *statefulConsoleRepo) DeleteSession(_ context.Context, activityID, sessionID int64) error {
	s, ok := r.sessions[sessionID]
	if !ok || s.ActivityID != activityID {
		return apperr.NotFound("activity session not found")
	}
	delete(r.sessions, sessionID)
	return nil
}

func (r *statefulConsoleRepo) ListTicketTypes(_ context.Context, activityID int64, sessionID *int64) ([]TicketType, error) {
	out := []TicketType{}
	for _, t := range r.ticketTypes {
		if t.ActivityID != activityID {
			continue
		}
		if sessionID != nil && (t.SessionID == nil || *t.SessionID != *sessionID) {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *statefulConsoleRepo) GetTicketType(_ context.Context, activityID, ticketTypeID int64) (TicketType, error) {
	t, ok := r.ticketTypes[ticketTypeID]
	if !ok || t.ActivityID != activityID {
		return TicketType{}, apperr.NotFound("ticket type not found")
	}
	return t, nil
}

func (r *statefulConsoleRepo) CreateTicketType(_ context.Context, activityID int64, in TicketTypeInput) (TicketType, error) {
	t := TicketType{
		ID: r.nextID(), ActivityID: activityID, SessionID: in.SessionID, Name: in.Name,
		PriceCent: in.PriceCent, StockQuantity: in.StockQuantity, SaleStartAt: in.SaleStartAt,
		SaleEndAt: in.SaleEndAt, PayChannels: in.PayChannels, MaxTicketsPerOrder: in.MaxTicketsPerOrder,
		Status: in.Status,
	}
	if a, ok := r.activities[activityID]; ok {
		t.StoreID = a.StoreID
	}
	r.ticketTypes[t.ID] = t
	return t, nil
}

func (r *statefulConsoleRepo) UpdateTicketType(_ context.Context, activityID, ticketTypeID int64, in TicketTypeInput) (TicketType, error) {
	t, ok := r.ticketTypes[ticketTypeID]
	if !ok || t.ActivityID != activityID {
		return TicketType{}, apperr.NotFound("ticket type not found")
	}
	t.Name = in.Name
	t.PriceCent = in.PriceCent
	t.Status = in.Status
	r.ticketTypes[ticketTypeID] = t
	return t, nil
}

func (r *statefulConsoleRepo) DeleteTicketType(_ context.Context, activityID, ticketTypeID int64) error {
	t, ok := r.ticketTypes[ticketTypeID]
	if !ok || t.ActivityID != activityID {
		return apperr.NotFound("ticket type not found")
	}
	delete(r.ticketTypes, ticketTypeID)
	return nil
}

// TestConsoleActivityWriteLifecycle drives the full create/update/delete path for
// an activity and its child session/ticket type under a store scope, asserting
// the store scope is pinned on create and enforced on cross-store access.
func TestConsoleActivityWriteLifecycle(t *testing.T) {
	repo := newStatefulConsoleRepo()
	svc := NewConsoleService(repo)
	ctx := context.Background()
	storeID := int64(7)
	scope := ConsoleScope{StoreID: &storeID}

	act, err := svc.CreateActivity(ctx, scope, ActivityInput{Title: "Fair", Status: "draft"})
	if err != nil {
		t.Fatalf("create activity: %v", err)
	}
	if act.ScopeType != "store" || act.StoreID == nil || *act.StoreID != storeID || act.Title != "Fair" {
		t.Fatalf("create should pin store scope, got %+v", act)
	}

	updated, err := svc.UpdateActivity(ctx, scope, act.ID, ActivityInput{Title: "Autumn Fair", Status: "published"})
	if err != nil {
		t.Fatalf("update activity: %v", err)
	}
	if updated.Title != "Autumn Fair" || updated.Status != "published" {
		t.Fatalf("update did not apply: %+v", updated)
	}

	// A different store scope can neither read nor delete the activity.
	otherStore := int64(8)
	otherScope := ConsoleScope{StoreID: &otherStore}
	if _, err := svc.GetActivity(ctx, otherScope, act.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND reading across store scope, got %v", err)
	}
	if err := svc.DeleteActivity(ctx, otherScope, act.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND deleting across store scope, got %v", err)
	}

	start := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	sess, err := svc.CreateSession(ctx, scope, act.ID, SessionInput{Name: "Morning", StartAt: start, EndAt: start.Add(2 * time.Hour), Status: "active"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if sess.ActivityID != act.ID || sess.Name != "Morning" {
		t.Fatalf("unexpected session: %+v", sess)
	}

	tt, err := svc.CreateTicketType(ctx, scope, act.ID, TicketTypeInput{Name: "VIP", PriceCent: 5000, StockQuantity: 100, Status: "active"})
	if err != nil {
		t.Fatalf("create ticket type: %v", err)
	}
	if tt.ActivityID != act.ID || tt.PriceCent != 5000 {
		t.Fatalf("unexpected ticket type: %+v", tt)
	}
	if tts, err := svc.ListTicketTypes(ctx, scope, act.ID, nil); err != nil || len(tts) != 1 {
		t.Fatalf("list ticket types: err=%v n=%d", err, len(tts))
	}

	if err := svc.DeleteTicketType(ctx, scope, act.ID, tt.ID); err != nil {
		t.Fatalf("delete ticket type: %v", err)
	}
	if err := svc.DeleteSession(ctx, scope, act.ID, sess.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if err := svc.DeleteActivity(ctx, scope, act.ID); err != nil {
		t.Fatalf("delete activity: %v", err)
	}
	if _, err := svc.GetActivity(ctx, scope, act.ID); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected activity to be gone, got %v", err)
	}
}

// TestConsoleChildWritesRequireActivityInScope proves the service checks the
// parent activity's scope before any session/ticket-type read or write. The
// fakeConsoleRepo (from activity_console_test.go) fails GetActivity with
// NOT_FOUND and would fail the underlying op with NOT_IMPLEMENTED, so a
// NOT_FOUND result can only come from the scope guard running first.
func TestConsoleChildWritesRequireActivityInScope(t *testing.T) {
	repo := &fakeConsoleRepo{err: apperr.NotFound("activity not found")}
	svc := NewConsoleService(repo)
	ctx := context.Background()
	storeID := int64(9)
	scope := ConsoleScope{StoreID: &storeID}

	ops := map[string]func() error{
		"GetSession":       func() error { _, err := svc.GetSession(ctx, scope, 1, 10); return err },
		"CreateSession":    func() error { _, err := svc.CreateSession(ctx, scope, 1, SessionInput{Name: "x"}); return err },
		"UpdateSession":    func() error { _, err := svc.UpdateSession(ctx, scope, 1, 10, SessionInput{Name: "x"}); return err },
		"DeleteSession":    func() error { return svc.DeleteSession(ctx, scope, 1, 10) },
		"ListTicketTypes":  func() error { _, err := svc.ListTicketTypes(ctx, scope, 1, nil); return err },
		"GetTicketType":    func() error { _, err := svc.GetTicketType(ctx, scope, 1, 100); return err },
		"CreateTicketType": func() error { _, err := svc.CreateTicketType(ctx, scope, 1, TicketTypeInput{Name: "x"}); return err },
		"UpdateTicketType": func() error { _, err := svc.UpdateTicketType(ctx, scope, 1, 100, TicketTypeInput{Name: "x"}); return err },
		"DeleteTicketType": func() error { return svc.DeleteTicketType(ctx, scope, 1, 100) },
	}
	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			if code := apperr.From(op()).Code; code != apperr.CodeNotFound {
				t.Fatalf("%s: expected NOT_FOUND from the activity scope guard, got %s", name, code)
			}
		})
	}
}
