package activity

import (
	"context"
	"strings"
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
	operations    StaffTodayOperationsData

	verifyCalledStore int64
	verifyCalledCode  string
	reviewDecision    string
	previewCalled     bool
	operationsStart   time.Time
	operationsEnd     time.Time
}

func (r *storeMemRepo) VerifyTicket(_ context.Context, storeID int64, code string, byID int64, now time.Time) (TicketVerification, error) {
	r.verifyCalledStore = storeID
	r.verifyCalledCode = code
	return TicketVerification{ID: 1, StoreID: storeID, TicketNo: code, Status: "verified", VerifiedBy: byID, VerifiedAt: now}, nil
}

func (r *storeMemRepo) ListPointSavings(_ context.Context, storeID int64, status, phone string, limit, offset int) ([]PointSaving, int64, error) {
	var all []PointSaving
	for _, p := range r.pointSavings {
		if p.StoreID == storeID && (status == "" || p.Status == status) &&
			(phone == "" || strings.Contains(p.Phone, phone)) {
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

func (r *storeMemRepo) PreviewPointSaving(_ context.Context, saving PointSaving, _ time.Time) (PointSaving, error) {
	r.previewCalled = true
	saving.PointsDivisor = 5
	saving.AwardedPoints = saving.Points / saving.PointsDivisor
	saving.CalculationDescription = "实际获得积分 = 存入积分 ÷ 5（向下取整）"
	return saving, nil
}

func (r *storeMemRepo) ReviewPointSaving(_ context.Context, storeID, requestID int64, decision, remark, reviewerType string, byID int64, now time.Time) (PointSaving, error) {
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
			p.ReviewedByType = reviewerType
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

func (r *storeMemRepo) StaffTodayOperations(_ context.Context, _ int64, dayStart, dayEnd time.Time, _ int) (StaffTodayOperationsData, error) {
	r.operationsStart = dayStart
	r.operationsEnd = dayEnd
	return r.operations, nil
}

type nopAssets struct{}

func (nopAssets) PublicURLByID(context.Context, int64) (string, error) { return "", nil }

type fixedAssets struct{ url string }

func (a fixedAssets) PublicURLByID(context.Context, int64) (string, error) { return a.url, nil }

func newStoreTestService() (*StoreService, *storeMemRepo) {
	repo := &storeMemRepo{}
	svc := NewStoreService(repo, nopAssets{}, time.FixedZone("Asia/Shanghai", 8*60*60))
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	return svc, repo
}

func TestVerifyTicketValidation(t *testing.T) {
	svc, repo := newStoreTestService()
	for _, code := range []string{"", "12345", "TCK-1", "1234567"} {
		_, err := svc.VerifyTicket(context.Background(), 1, code, 5)
		if err == nil || apperr.From(err).Code != apperr.CodeInvalidArgument {
			t.Fatalf("expected INVALID_ARGUMENT for code %q, got %v", code, err)
		}
	}
	if repo.verifyCalledCode != "" {
		t.Fatal("repo must not be called with an invalid code")
	}
}

func TestVerifyTicketScopeAndStaff(t *testing.T) {
	svc, repo := newStoreTestService()
	view, err := svc.VerifyTicket(context.Background(), 7, "012345", 5)
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
		{ID: 1, StoreID: 7, MemberID: 1, MemberName: "会员甲", MemberAvatarURL: "https://cdn.test/member.jpg",
			Points: 100, Status: PointSavingApproved, ReviewedByType: "store_admin",
			ReviewerSnapshotJSON: []byte(`{"type":"store_admin","id":2,"username":"storeadmin"}`)},
		{ID: 2, StoreID: 7, MemberID: 2, Points: 50, Status: PointSavingPending},
		{ID: 3, StoreID: 99, MemberID: 3, Points: 10, Status: PointSavingPending},
	}
	views, total, err := svc.ListPointSavings(context.Background(), 7, httpx.Page{Page: 1, PageSize: 20}, "", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(views) != 2 {
		t.Fatalf("expected 2 requests for store 7, got %d", total)
	}
	if views[0].MemberID != 1 || views[0].MemberAvatarURL == "" ||
		views[0].ReviewedByType != "store_admin" || string(views[0].Reviewer) != `{"type":"store_admin","id":2,"username":"storeadmin"}` {
		t.Fatalf("expected member and reviewer identity fields, got %+v", views[0])
	}
}

func TestListPointSavingsFiltersPendingByPhone(t *testing.T) {
	svc, repo := newStoreTestService()
	repo.pointSavings = []PointSaving{
		{ID: 1, StoreID: 7, Phone: "13800001111", Status: PointSavingPending},
		{ID: 2, StoreID: 7, Phone: "13800002222", Status: PointSavingApproved},
		{ID: 3, StoreID: 7, Phone: "13900001111", Status: PointSavingPending},
		{ID: 4, StoreID: 99, Phone: "13800001111", Status: PointSavingPending},
	}
	views, total, err := svc.ListPointSavings(
		context.Background(), 7, httpx.Page{Page: 1, PageSize: 20}, PointSavingPending, "1111",
	)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 || len(views) != 2 || views[0].Status != PointSavingPending {
		t.Fatalf("expected two pending phone matches, got %+v (total %d)", views, total)
	}
	for _, phone := range []string{"123", "12A4", "123456789012"} {
		if _, _, err := svc.ListPointSavings(context.Background(), 7, httpx.Page{}, PointSavingPending, phone); apperr.From(err).Code != apperr.CodeInvalidArgument {
			t.Fatalf("expected invalid phone filter %q, got %v", phone, err)
		}
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

func TestGetPointSavingPreviewsPendingAward(t *testing.T) {
	svc, repo := newStoreTestService()
	repo.pointSavings = []PointSaving{{ID: 1, StoreID: 7, MemberID: 3, Points: 100, Status: PointSavingPending}}
	view, err := svc.GetPointSaving(context.Background(), 7, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !repo.previewCalled || view.AwardedPoints != 20 || view.CalculationDescription == "" {
		t.Fatalf("expected pending calculation preview, got %+v", view)
	}
}

func TestListPointSavingsResolvesAvatarAsset(t *testing.T) {
	repo := &storeMemRepo{}
	svc := NewStoreService(repo, fixedAssets{url: "https://cdn.test/resolved-avatar.webp"}, time.UTC)
	assetID := int64(88)
	repo.pointSavings = []PointSaving{{
		ID: 1, StoreID: 7, MemberID: 3, MemberAvatarAssetID: &assetID,
		Points: 100, Status: PointSavingPending,
	}}
	views, _, err := svc.ListPointSavings(context.Background(), 7, httpx.Page{Page: 1, PageSize: 20}, "", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(views) != 1 || views[0].MemberAvatarURL != "https://cdn.test/resolved-avatar.webp" {
		t.Fatalf("expected resolved avatar, got %+v", views)
	}
}

func TestReviewPointSavingValidation(t *testing.T) {
	svc, _ := newStoreTestService()
	req := ReviewPointSavingRequest{Decision: "maybe"}
	if _, err := svc.ReviewPointSaving(context.Background(), 7, 1, req, "staff", 5); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for bad decision, got %v", err)
	}
}

func TestReviewPointSavingApprove(t *testing.T) {
	svc, repo := newStoreTestService()
	repo.pointSavings = []PointSaving{{ID: 1, StoreID: 7, MemberID: 1, Points: 100, Status: PointSavingPending}}
	view, err := svc.ReviewPointSaving(context.Background(), 7, 1, ReviewPointSavingRequest{Decision: ReviewApprove, Remark: "ok"}, "staff", 5)
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if view.Status != PointSavingApproved || view.ReviewedBy == nil || *view.ReviewedBy != 5 || view.ReviewedByType != "staff" {
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

func TestStaffTodayOperationsUsesBusinessDayAndSeparateAssetTotals(t *testing.T) {
	svc, repo := newStoreTestService()
	repo.operations = StaffTodayOperationsData{
		CoinConsumptionAmount: 18,
		CoinConsumptionCount:  2,
		PointDepositAmount:    300,
		PointDepositCount:     1,
		PointWithdrawalAmount: 120,
		PointWithdrawalCount:  1,
		Entries: []StaffOperation{{
			RecordKey: "coin:1", Type: "coin_consumption", MemberID: 8,
			MemberName: "会员甲", Amount: 18, Status: "completed",
		}},
	}
	view, err := svc.StaffTodayOperations(context.Background(), 7)
	if err != nil {
		t.Fatalf("today operations: %v", err)
	}
	if view.Date != "2026-07-17" || view.Summary.CoinConsumptionAmount != 18 ||
		view.Summary.PointDepositAmount != 300 || view.Summary.PointWithdrawalAmount != 120 {
		t.Fatalf("unexpected daily view: %+v", view)
	}
	wantStart := time.Date(2026, 7, 16, 16, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 7, 17, 16, 0, 0, 0, time.UTC)
	if !repo.operationsStart.Equal(wantStart) || !repo.operationsEnd.Equal(wantEnd) {
		t.Fatalf("unexpected business-day bounds: %v - %v", repo.operationsStart, repo.operationsEnd)
	}
	if len(view.Entries) != 1 || view.Entries[0].Type != "coin_consumption" {
		t.Fatalf("unexpected entries: %+v", view.Entries)
	}
}
