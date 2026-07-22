package activity

import (
	"context"
	"testing"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// storeMemRepo is an in-memory StoreRepository for service-level tests. It
// records the store scope it was called with so tests can assert isolation.
type storeMemRepo struct {
	pointSavings  []PointSaving
	verifications []Verification
	activities    []Activity

	verifyCalledStore int64
	verifyCalledCode  string
	reviewDecision    string
}

func (r *storeMemRepo) VerifyTicket(_ context.Context, storeID int64, code string, byID int64, now time.Time) (TicketVerification, error) {
	r.verifyCalledStore = storeID
	r.verifyCalledCode = code
	return TicketVerification{ID: 1, StoreID: storeID, TicketNo: code, Status: "verified", VerifiedBy: byID, VerifiedAt: now}, nil
}

func (r *storeMemRepo) ListPointSavings(_ context.Context, storeID int64, limit, offset int) ([]PointSaving, int64, error) {
	var all []PointSaving
	for _, p := range r.pointSavings {
		if p.StoreID == storeID {
			all = append(all, p)
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

func (r *storeMemRepo) GetPointSaving(_ context.Context, storeID, requestID int64) (PointSaving, error) {
	for _, p := range r.pointSavings {
		if p.ID == requestID && p.StoreID == storeID {
			return p, nil
		}
	}
	return PointSaving{}, apperr.NotFound("point-saving request not found")
}

func (r *storeMemRepo) ReviewPointSaving(_ context.Context, storeID, requestID int64, decision, remark string, byID int64, now time.Time) (PointSaving, error) {
	r.reviewDecision = decision
	for i := range r.pointSavings {
		p := &r.pointSavings[i]
		if p.ID == requestID && p.StoreID == storeID && p.Status == PointSavingPending {
			if decision == ReviewApprove {
				p.Status = PointSavingApproved
			} else {
				p.Status = PointSavingRejected
			}
			p.Remark = remark
			p.ReviewedBy = &byID
			p.ReviewedAt = &now
			return *p, nil
		}
	}
	return PointSaving{}, apperr.Conflict("point-saving request cannot be reviewed")
}

func (r *storeMemRepo) ListTodayActivities(_ context.Context, storeID int64, _, _ time.Time) ([]Activity, error) {
	var out []Activity
	for _, a := range r.activities {
		if a.StoreID != nil && *a.StoreID == storeID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *storeMemRepo) ListTodayActivitySummaries(_ context.Context, storeID int64, _, _ time.Time) ([]TodayActivity, error) {
	var out []TodayActivity
	for _, a := range r.activities {
		if a.StoreID != nil && *a.StoreID == storeID {
			out = append(out, TodayActivity{ID: a.ID, Title: a.Title, StartAt: a.StartAt, EndAt: a.EndAt})
		}
	}
	return out, nil
}

func (r *storeMemRepo) ListVerifications(_ context.Context, storeID int64, limit, offset int) ([]Verification, int64, error) {
	var all []Verification
	for _, v := range r.verifications {
		if v.StoreID == storeID {
			all = append(all, v)
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

func (r *storeMemRepo) StaffHome(_ context.Context, storeID int64, dayStart, dayEnd time.Time) (StaffHomeData, error) {
	data := StaffHomeData{StoreName: "Test Store"}
	for _, p := range r.pointSavings {
		if p.StoreID == storeID && p.Status == PointSavingPending {
			data.PendingReview++
		}
	}
	for _, v := range r.verifications {
		if v.StoreID == storeID && !v.VerifiedAt.Before(dayStart) && v.VerifiedAt.Before(dayEnd) {
			data.TodayVerifications++
		}
	}
	for i := range r.activities {
		a := r.activities[i]
		if a.StoreID != nil && *a.StoreID == storeID {
			data.TodayActivity = &r.activities[i]
			break
		}
	}
	return data, nil
}

type nopAssets struct{}

func (nopAssets) PublicURLByID(context.Context, int64) (string, error) { return "", nil }

func newStoreTestService() (*StoreService, *storeMemRepo) {
	repo := &storeMemRepo{}
	svc := NewStoreService(repo, nopAssets{})
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	return svc, repo
}

func TestVerifyTicketValidation(t *testing.T) {
	svc, repo := newStoreTestService()
	if _, err := svc.VerifyTicket(context.Background(), 1, "", 5); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for empty code, got %v", err)
	}
	if repo.verifyCalledCode != "" {
		t.Fatal("repo must not be called with an invalid code")
	}
}

func TestVerifyTicketScopeAndStaff(t *testing.T) {
	svc, repo := newStoreTestService()
	view, err := svc.VerifyTicket(context.Background(), 7, "TCK-1", 5)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if repo.verifyCalledStore != 7 {
		t.Fatalf("expected store scope 7 passed to repo, got %d", repo.verifyCalledStore)
	}
	if view.VerifiedBy != 5 || view.Status != "verified" {
		t.Fatalf("unexpected verification view: %+v", view)
	}
}

func TestListPointSavingsScoped(t *testing.T) {
	svc, repo := newStoreTestService()
	repo.pointSavings = []PointSaving{
		{ID: 1, StoreID: 7, MemberID: 1, Points: 100, Status: PointSavingPending},
		{ID: 2, StoreID: 7, MemberID: 2, Points: 50, Status: PointSavingPending},
		{ID: 3, StoreID: 99, MemberID: 3, Points: 10, Status: PointSavingPending},
	}
	views, total, err := svc.ListPointSavings(context.Background(), 7, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(views) != 2 {
		t.Fatalf("expected 2 requests for store 7, got %d", total)
	}
}

func TestGetPointSavingCrossStoreHidden(t *testing.T) {
	svc, repo := newStoreTestService()
	repo.pointSavings = []PointSaving{{ID: 1, StoreID: 99, MemberID: 3, Points: 10, Status: PointSavingPending}}
	// A store must not read another store's request.
	if _, err := svc.GetPointSaving(context.Background(), 7, 1); apperr.From(err).Code != apperr.CodeNotFound {
		t.Fatalf("expected NOT_FOUND for cross-store lookup, got %v", err)
	}
}

func TestReviewPointSavingValidation(t *testing.T) {
	svc, _ := newStoreTestService()
	req := ReviewPointSavingRequest{Decision: "maybe"}
	if _, err := svc.ReviewPointSaving(context.Background(), 7, 1, req, 5); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for bad decision, got %v", err)
	}
}

func TestReviewPointSavingApprove(t *testing.T) {
	svc, repo := newStoreTestService()
	repo.pointSavings = []PointSaving{{ID: 1, StoreID: 7, MemberID: 1, Points: 100, Status: PointSavingPending}}
	view, err := svc.ReviewPointSaving(context.Background(), 7, 1, ReviewPointSavingRequest{Decision: ReviewApprove, Remark: "ok"}, 5)
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if view.Status != PointSavingApproved || view.ReviewedBy == nil || *view.ReviewedBy != 5 {
		t.Fatalf("unexpected reviewed view: %+v", view)
	}
}

func TestTodayActivitiesScoped(t *testing.T) {
	svc, repo := newStoreTestService()
	s7 := int64(7)
	s99 := int64(99)
	repo.activities = []Activity{
		{ID: 1, StoreID: &s7, Title: "A", Status: "published"},
		{ID: 2, StoreID: &s99, Title: "B", Status: "published"},
	}
	views, err := svc.TodayActivities(context.Background(), 7)
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	if len(views) != 1 || views[0].Title != "A" {
		t.Fatalf("expected 1 activity for store 7, got %+v", views)
	}
}

func TestListVerificationsScoped(t *testing.T) {
	svc, repo := newStoreTestService()
	repo.verifications = []Verification{
		{ID: 1, StoreID: 7, Kind: "ticket", RefID: 10},
		{ID: 2, StoreID: 99, Kind: "ticket", RefID: 11},
	}
	views, total, err := svc.ListVerifications(context.Background(), 7, httpx.Page{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(views) != 1 || views[0].RefID != 10 {
		t.Fatalf("expected 1 verification for store 7, got %+v (total %d)", views, total)
	}
}

func TestStaffHomeAggregates(t *testing.T) {
	svc, repo := newStoreTestService()
	s7 := int64(7)
	repo.pointSavings = []PointSaving{
		{ID: 1, StoreID: 7, Status: PointSavingPending},
		{ID: 2, StoreID: 7, Status: PointSavingApproved},
		{ID: 3, StoreID: 99, Status: PointSavingPending},
	}
	repo.verifications = []Verification{
		{ID: 1, StoreID: 7, VerifiedAt: time.Date(2026, 7, 17, 9, 0, 0, 0, time.UTC)},
		{ID: 2, StoreID: 7, VerifiedAt: time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC)},
	}
	repo.activities = []Activity{{ID: 10, StoreID: &s7, Title: "Today", Status: "published"}}

	view, err := svc.StaffHome(context.Background(), 7)
	if err != nil {
		t.Fatalf("staff home: %v", err)
	}
	if view.Store.Name != "Test Store" {
		t.Fatalf("expected store name, got %q", view.Store.Name)
	}
	if view.PendingReview != 1 {
		t.Fatalf("expected 1 pending review, got %d", view.PendingReview)
	}
	if view.TodayVerifications != 1 {
		t.Fatalf("expected 1 today verification, got %d", view.TodayVerifications)
	}
	if view.TodayActivity == nil || view.TodayActivity.Title != "Today" {
		t.Fatalf("expected today activity, got %+v", view.TodayActivity)
	}
}

func TestStaffHomeNoActivity(t *testing.T) {
	svc, _ := newStoreTestService()
	view, err := svc.StaffHome(context.Background(), 7)
	if err != nil {
		t.Fatalf("staff home: %v", err)
	}
	if view.TodayActivity != nil {
		t.Fatalf("expected nil today activity, got %+v", view.TodayActivity)
	}
}
