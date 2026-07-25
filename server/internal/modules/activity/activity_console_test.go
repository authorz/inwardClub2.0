package activity

import (
	"context"
	"strconv"
	"testing"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// fakeConsoleRepo is an in-memory ConsoleRepository double. It records the
// scope/ids it was called with so tests can assert both mapping and scope
// propagation.
type fakeConsoleRepo struct {
	lastScope      ConsoleScope
	lastActivityID int64
	err            error
}

type consoleAssetResolver struct{}

func (consoleAssetResolver) PublicURLByID(_ context.Context, id int64) (string, error) {
	return "https://cdn.test/assets/" + strconv.FormatInt(id, 10), nil
}

func (f *fakeConsoleRepo) ListActivities(_ context.Context, scope ConsoleScope, _ httpx.Page) ([]Activity, int64, error) {
	f.lastScope = scope
	if f.err != nil {
		return nil, 0, f.err
	}
	return []Activity{{ID: 1, ScopeType: "store", Title: "Spring Fair", Status: "draft"}}, 1, nil
}

func (f *fakeConsoleRepo) GetActivity(_ context.Context, scope ConsoleScope, id int64) (Activity, error) {
	f.lastScope = scope
	f.lastActivityID = id
	if f.err != nil {
		return Activity{}, f.err
	}
	assetID := int64(9)
	return Activity{
		ID: id, ScopeType: "store", Title: "Spring Fair", AssetID: &assetID,
		PurchaseLimit: 3, Status: "draft",
	}, nil
}

func (f *fakeConsoleRepo) CreateActivity(context.Context, ConsoleScope, ActivityInput) (Activity, error) {
	return Activity{}, apperr.NotImplemented("fake: create activity")
}
func (f *fakeConsoleRepo) UpdateActivity(context.Context, ConsoleScope, int64, ActivityInput) (Activity, error) {
	return Activity{}, apperr.NotImplemented("fake: update activity")
}
func (f *fakeConsoleRepo) DeleteActivity(context.Context, ConsoleScope, int64) error {
	return apperr.NotImplemented("fake: delete activity")
}

func (f *fakeConsoleRepo) ListSessions(_ context.Context, activityID int64) ([]Session, error) {
	f.lastActivityID = activityID
	return []Session{{ID: 10, ActivityID: activityID, Name: "Morning slot", Status: "active"}}, nil
}
func (f *fakeConsoleRepo) GetSession(_ context.Context, activityID, sessionID int64) (Session, error) {
	return Session{ID: sessionID, ActivityID: activityID, Name: "Morning slot", Status: "active"}, nil
}
func (f *fakeConsoleRepo) CreateSession(context.Context, int64, SessionInput) (Session, error) {
	return Session{}, apperr.NotImplemented("fake: create session")
}
func (f *fakeConsoleRepo) UpdateSession(context.Context, int64, int64, SessionInput) (Session, error) {
	return Session{}, apperr.NotImplemented("fake: update session")
}
func (f *fakeConsoleRepo) DeleteSession(context.Context, int64, int64) error {
	return apperr.NotImplemented("fake: delete session")
}

func (f *fakeConsoleRepo) ListTicketTypes(_ context.Context, activityID int64, _ *int64) ([]TicketType, error) {
	f.lastActivityID = activityID
	return []TicketType{{ID: 100, ActivityID: activityID, Name: "VIP", PriceCent: 5000, Status: "active"}}, nil
}
func (f *fakeConsoleRepo) GetTicketType(_ context.Context, activityID, ticketTypeID int64) (TicketType, error) {
	return TicketType{ID: ticketTypeID, ActivityID: activityID, Name: "VIP", PriceCent: 5000, Status: "active"}, nil
}
func (f *fakeConsoleRepo) CreateTicketType(context.Context, int64, TicketTypeInput) (TicketType, error) {
	return TicketType{}, apperr.NotImplemented("fake: create ticket type")
}
func (f *fakeConsoleRepo) UpdateTicketType(context.Context, int64, int64, TicketTypeInput) (TicketType, error) {
	return TicketType{}, apperr.NotImplemented("fake: update ticket type")
}
func (f *fakeConsoleRepo) DeleteTicketType(context.Context, int64, int64) error {
	return apperr.NotImplemented("fake: delete ticket type")
}

func consolePage() httpx.Page { return httpx.Page{Page: 1, PageSize: 20} }

func TestConsoleListActivitiesMapsAndPassesTotal(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo)

	views, total, err := svc.ListActivities(context.Background(), ConsoleScope{}, consolePage())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(views) != 1 || views[0].Title != "Spring Fair" || views[0].Status != "draft" {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestConsoleListActivitiesPropagatesScope(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo)
	storeID := int64(42)

	if _, _, err := svc.ListActivities(context.Background(), ConsoleScope{StoreID: &storeID}, consolePage()); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.lastScope.StoreID == nil || *repo.lastScope.StoreID != storeID {
		t.Fatalf("expected store scope %d, got %v", storeID, repo.lastScope.StoreID)
	}
}

func TestConsoleGetActivityMaps(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo, consoleAssetResolver{})

	view, err := svc.GetActivity(context.Background(), ConsoleScope{}, 7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.ID != 7 || view.Title != "Spring Fair" || view.PurchaseLimitPerMember != 3 {
		t.Fatalf("unexpected view: %+v", view)
	}
	if view.ImageURL != "https://cdn.test/assets/9" {
		t.Fatalf("unexpected image URL: %q", view.ImageURL)
	}
}

func TestConsoleListSessionsVerifiesActivityScopeFirst(t *testing.T) {
	sentinel := apperr.NotFound("activity not found")
	repo := &fakeConsoleRepo{err: sentinel}
	svc := NewConsoleService(repo)

	storeID := int64(9)
	_, err := svc.ListSessions(context.Background(), ConsoleScope{StoreID: &storeID}, 1)
	if apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND when activity is out of scope, got %v", err)
	}
}

func TestConsoleListSessionsMaps(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo)

	views, err := svc.ListSessions(context.Background(), ConsoleScope{}, 5)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(views) != 1 || views[0].ActivityID != 5 || views[0].Name != "Morning slot" {
		t.Fatalf("unexpected views: %+v", views)
	}
}

func TestConsoleListTicketTypesMaps(t *testing.T) {
	repo := &fakeConsoleRepo{}
	svc := NewConsoleService(repo)

	views, err := svc.ListTicketTypes(context.Background(), ConsoleScope{}, 5, nil)
	if err != nil {
		t.Fatalf("list ticket types: %v", err)
	}
	if len(views) != 1 || views[0].ActivityID != 5 || views[0].Name != "VIP" || views[0].PriceCent != 5000 {
		t.Fatalf("unexpected views: %+v", views)
	}
}

// encodeChannels coerces a nil pay-channel slice to an empty JSON array so the
// NOT NULL column always receives a valid value.
func TestEncodeChannels(t *testing.T) {
	if got := string(encodeChannels(nil)); got != "[]" {
		t.Fatalf("nil channels: expected [], got %q", got)
	}
	if got := string(encodeChannels([]string{"wechat", "coin"})); got != `["wechat","coin"]` {
		t.Fatalf("unexpected encoding: %q", got)
	}
}

func TestNormalizePayChannels(t *testing.T) {
	got, err := normalizePayChannels([]string{"wechat", "balance", "coin"})
	if err != nil {
		t.Fatalf("normalize legacy balance: %v", err)
	}
	if len(got) != 2 || got[0] != "wechat" || got[1] != "coin" {
		t.Fatalf("unexpected normalized channels: %#v", got)
	}
	if _, err := normalizePayChannels([]string{"wechat", "alipay"}); err == nil {
		t.Fatal("unsupported channel should be rejected")
	}
}

// scopeWrite pins a store console to its own scope while allowing the admin
// console to choose a specific store or a global row.
func TestScopeWrite(t *testing.T) {
	if st, sid := scopeWrite(ConsoleScope{}, nil); st != "global" || sid != nil {
		t.Fatalf("admin scope: expected global/nil, got %q %v", st, sid)
	}
	storeID := int64(7)
	if st, sid := scopeWrite(ConsoleScope{}, &storeID); st != "store" || sid != int64(7) {
		t.Fatalf("admin store binding: expected store/7, got %q %v", st, sid)
	}
	otherStoreID := int64(8)
	if st, sid := scopeWrite(ConsoleScope{StoreID: &storeID}, &otherStoreID); st != "store" || sid != int64(7) {
		t.Fatalf("store scope: expected store/7, got %q %v", st, sid)
	}
}

func TestScopeWhereAdminHasNoRestriction(t *testing.T) {
	where, args := scopeWhere(ConsoleScope{})
	if where != "1 = 1" || len(args) != 0 {
		t.Fatalf("expected unrestricted admin scope, got %q %v", where, args)
	}
}

func TestScopeWherePinsStore(t *testing.T) {
	storeID := int64(3)
	where, args := scopeWhere(ConsoleScope{StoreID: &storeID})
	if where != "scope_type = 'store' AND store_id = ?" || len(args) != 1 || args[0] != storeID {
		t.Fatalf("expected store-scoped where, got %q %v", where, args)
	}
}
