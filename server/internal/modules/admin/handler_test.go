package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/modules/wallet"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
)

func TestAdminLookupMemberReturnsCandidateArray(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{members: map[int64]Member{
		7: {ID: 7, Nickname: "Waiter Wang", Phone: "13800000007", AvatarURL: "https://cdn.test/staff.webp", Status: StatusActive},
	}}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/member-lookup", h.AdminLookupMember)

	req := httptest.NewRequest(http.MethodGet, "/admin/member-lookup?phone=0007", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data []MemberView `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Data) != 1 || envelope.Data[0].ID != 7 ||
		envelope.Data[0].Phone != "13800000007" ||
		envelope.Data[0].AvatarURL != "https://cdn.test/staff.webp" {
		t.Fatalf("unexpected candidates: %+v", envelope.Data)
	}
}

func TestAdminLookupMemberRejectsWildcard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(NewService(&fakeRepo{}, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/member-lookup", h.AdminLookupMember)

	req := httptest.NewRequest(http.MethodGet, "/admin/member-lookup?phone=%25%25%25", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMembersPassesSearchAndSortAndReturnsExpandedFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/members", h.Members)

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/members?keyword=Sam&sortBy=coinsBalance&sortOrder=asc",
		nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.lastFilter.Keyword != "Sam" || repo.lastFilter.SortBy != "coinsBalance" ||
		repo.lastFilter.SortOrder != "asc" {
		t.Fatalf("unexpected filter: %+v", repo.lastFilter)
	}
	body := rec.Body.String()
	for _, field := range []string{
		`"avatarUrl":"https://cdn.test/avatar.webp"`,
		`"gender":"male"`,
		`"pointsBalance":1200`,
		`"coinsBalance":588`,
		`"vipTierName":"白银会员"`,
		`"vipLevel":2`,
	} {
		if !strings.Contains(body, field) {
			t.Fatalf("expected %s in response: %s", field, body)
		}
	}
}

func TestOrdersPassesFuzzySearchAndStoreFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/orders", h.Orders)

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/orders?storeId=42&memberNickname=%E5%B0%8F%E6%98%8E&memberPhone=138&keyword=NO-2026&paymentStatus=paid&orderStatus=completed&payChannel=wechat",
		nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 ||
		repo.lastFilter.MemberNickname != "小明" ||
		repo.lastFilter.MemberPhone != "138" ||
		repo.lastFilter.Keyword != "NO-2026" ||
		repo.lastFilter.PaymentStatus != "paid" ||
		repo.lastFilter.Status != "completed" ||
		repo.lastFilter.PayChannel != "wechat" {
		t.Fatalf("unexpected order filters: %+v", repo.lastFilter)
	}
}

func TestOrdersRejectsInvalidStoreID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(NewService(&fakeRepo{}, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/orders", h.Orders)

	req := httptest.NewRequest(http.MethodGet, "/admin/orders?storeId=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestStaffAccountsHandlerReturnsStaffAccounts guards against the console
// endpoint GET /admin/staff-accounts regressing to admin_accounts data: the
// fakeRepo returns distinguishable rows for ListAdminAccounts ("hq_admin")
// and ListStaffAccounts ("Waiter Wang"), so the handler must call the latter.
func TestStaffAccountsHandlerReturnsStaffAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/admin/staff-accounts", h.StaffAccounts)

	req := httptest.NewRequest(http.MethodGet, "/admin/staff-accounts", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Waiter Wang") {
		t.Fatalf("expected staff account data in response, got: %s", body)
	}
	if strings.Contains(body, "hq_admin") {
		t.Fatalf("response leaked admin_accounts data: %s", body)
	}
}

func TestRefundsHandlerReturnsRefunds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/admin/refunds", h.Refunds)

	req := httptest.NewRequest(http.MethodGet, "/admin/refunds", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "RF-1") {
		t.Fatalf("expected refund data in response, got: %s", rec.Body.String())
	}
}

func TestRefundsHandlerRejectsInvalidStoreID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/admin/refunds", h.Refunds)

	req := httptest.NewRequest(http.MethodGet, "/admin/refunds?storeId=abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRefundsHandlerPassesSearchAndTimeFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/refunds", h.Refunds)

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/refunds?id=8&keyword=BO-20&memberNickname=%E5%B0%8F%E6%98%8E&memberPhone=138&storeId=42&status=succeeded&operatedFrom=2026-07-20T00%3A00%3A00Z&operatedTo=2026-07-21T00%3A00%3A00Z",
		nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.lastFilter.RefundID != "8" ||
		repo.lastFilter.Keyword != "BO-20" ||
		repo.lastFilter.MemberNickname != "小明" ||
		repo.lastFilter.MemberPhone != "138" ||
		repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 ||
		repo.lastFilter.Status != "succeeded" ||
		repo.lastFilter.OperatedFrom == nil ||
		repo.lastFilter.OperatedBefore == nil {
		t.Fatalf("unexpected refund filters: %+v", repo.lastFilter)
	}
	wantBefore := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	if !repo.lastFilter.OperatedBefore.Equal(wantBefore) {
		t.Fatalf("expected inclusive end date before %v, got %v", wantBefore, repo.lastFilter.OperatedBefore)
	}
}

func TestRefundsHandlerRejectsNonFinalStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(NewService(&fakeRepo{}, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/refunds", h.Refunds)

	req := httptest.NewRequest(http.MethodGet, "/admin/refunds?status=pending", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestRefundOrdersAdminAliasReturnsSameData guards GET /admin/refund-orders
// against drifting from GET /admin/refunds: both must hit the same handler.
func TestRefundOrdersAdminAliasReturnsSameData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/admin/refund-orders", h.Refunds)

	req := httptest.NewRequest(http.MethodGet, "/admin/refund-orders", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "RF-1") {
		t.Fatalf("expected refund data in response, got: %s", rec.Body.String())
	}
}

func TestWalletLedgerHandlerReturnsEntries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/admin/wallet-ledger", h.WalletLedger)

	req := httptest.NewRequest(http.MethodGet, "/admin/wallet-ledger?memberId=7&assetType=points&storeId=42", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"memberId":7`) {
		t.Fatalf("expected wallet ledger data in response, got: %s", rec.Body.String())
	}
	if repo.lastFilter.MemberID == nil || *repo.lastFilter.MemberID != 7 ||
		repo.lastFilter.AssetType != "points" ||
		!repo.lastFilter.IncludePointRequests ||
		repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 {
		t.Fatalf("expected filters propagated, got %+v", repo.lastFilter)
	}
}

func TestWalletLedgerHandlerRejectsInvalidMemberID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/admin/wallet-ledger", h.WalletLedger)

	req := httptest.NewRequest(http.MethodGet, "/admin/wallet-ledger?memberId=abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWalletLedgerHandlerPassesRichSearchFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/wallet-ledger", h.WalletLedger)

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/wallet-ledger?id=10&memberNickname=Sam&memberPhone=138&assetType=coins&direction=debit&sourceType=payment&status=completed&reason=order&createdFrom=2026-07-20T00%3A00%3A00Z&createdTo=2026-07-21T00%3A00%3A00Z",
		nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.lastFilter.LedgerID != "10" ||
		repo.lastFilter.MemberNickname != "Sam" ||
		repo.lastFilter.MemberPhone != "138" ||
		repo.lastFilter.AssetType != "coins" ||
		repo.lastFilter.Direction != "debit" ||
		repo.lastFilter.SourceType != "payment" ||
		repo.lastFilter.Status != "completed" ||
		repo.lastFilter.ReasonKeyword != "order" ||
		repo.lastFilter.CreatedFrom == nil ||
		repo.lastFilter.CreatedBefore == nil {
		t.Fatalf("unexpected wallet filters: %+v", repo.lastFilter)
	}
	wantBefore := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	if !repo.lastFilter.CreatedBefore.Equal(wantBefore) {
		t.Fatalf("expected inclusive end date before %v, got %v", wantBefore, repo.lastFilter.CreatedBefore)
	}
}

func TestWalletLedgerHandlerRejectsInvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(NewService(&fakeRepo{}, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/wallet-ledger", h.WalletLedger)

	req := httptest.NewRequest(http.MethodGet, "/admin/wallet-ledger?status=unknown", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStoreWalletLedgerHandlerScopesToStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/store/wallet-ledger", withStoreScope(42), h.StoreWalletLedger)

	// A client-supplied storeId must never override the token scope.
	req := httptest.NewRequest(http.MethodGet, "/store/wallet-ledger?storeId=99", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 {
		t.Fatalf("expected store scope pinned to 42 from token, got %v", repo.lastFilter.StoreID)
	}
}

func TestStoreWalletLedgerHandlerRequiresScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/store/wallet-ledger", h.StoreWalletLedger)

	req := httptest.NewRequest(http.MethodGet, "/store/wallet-ledger", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("expected failure without store scope, got 200: %s", rec.Body.String())
	}
}

// withStoreScope injects the given store id the way storescope.Inject would,
// without requiring a real JWT in tests.
func withStoreScope(storeID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(httpx.CtxStoreScope, storeID)
		c.Next()
	}
}

func TestStoreMembersHandlerListsPlatformWideMembers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.GET("/store/members", withStoreScope(42), h.StoreMembers)

	req := httptest.NewRequest(http.MethodGet, "/store/members", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.lastFilter.StoreID != nil {
		t.Fatalf("member list must not be store-scoped, got store id %v", *repo.lastFilter.StoreID)
	}
}

func TestStoreMemberDetailReadsPlatformWideMember(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{members: map[int64]Member{
		7: {ID: 7, Nickname: "Global Member", Status: StatusActive},
	}}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.GET("/store/members/:memberID", withStoreScope(42), h.StoreMemberDetail)

	req := httptest.NewRequest(http.MethodGet, "/store/members/7", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for member without store ownership, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Global Member") {
		t.Fatalf("expected platform-wide member detail, got: %s", rec.Body.String())
	}
}

func TestStoreRefundsHandlerScopesToStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/store/refunds", withStoreScope(42), h.StoreRefunds)

	req := httptest.NewRequest(http.MethodGet, "/store/refunds", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "RF-1") {
		t.Fatalf("expected refund data in response, got: %s", rec.Body.String())
	}
	if repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 {
		t.Fatalf("expected store scope 42, got %v", repo.lastFilter.StoreID)
	}
}

func TestStoreRefundsHandlerRequiresScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/store/refunds", h.StoreRefunds)

	req := httptest.NewRequest(http.MethodGet, "/store/refunds", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("expected failure without store scope, got 200: %s", rec.Body.String())
	}
}

// TestRefundOrdersStoreAliasReturnsSameData guards GET /store/refund-orders
// against drifting from GET /store/refunds: both must hit the same handler.
func TestRefundOrdersStoreAliasReturnsSameData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/store/refund-orders", withStoreScope(42), h.StoreRefunds)

	req := httptest.NewRequest(http.MethodGet, "/store/refund-orders", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "RF-1") {
		t.Fatalf("expected refund data in response, got: %s", rec.Body.String())
	}
}

func TestStoreCreateCashierHandlerScopesToStoreAndReturnsPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/store/cashiers", withStoreScope(42), h.StoreCreateCashier)

	body := `{"username":"cashier1","displayName":"Cashier One"}`
	req := httptest.NewRequest(http.MethodPost, "/store/cashiers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "initialPassword") {
		t.Fatalf("expected initialPassword in response, got: %s", rec.Body.String())
	}
	if len(repo.cashiers) != 1 {
		t.Fatalf("expected 1 cashier stored, got %d", len(repo.cashiers))
	}
	for _, a := range repo.cashiers {
		if a.StoreID == nil || *a.StoreID != 42 {
			t.Fatalf("expected cashier pinned to store 42, got %+v", a)
		}
	}
}

func TestStoreCreateCashierHandlerRequiresScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := NewService(&fakeRepo{}, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/store/cashiers", h.StoreCreateCashier)

	body := `{"username":"cashier1"}`
	req := httptest.NewRequest(http.MethodPost, "/store/cashiers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusCreated {
		t.Fatalf("expected failure without store scope, got 201: %s", rec.Body.String())
	}
}

func TestStoreDisableCashierHandlerCannotReachOtherStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	repo.cashiers = map[int64]AdminAccount{}
	storeID := int64(42)
	repo.cashiers[1] = AdminAccount{ID: 1, Username: "cashier1", Role: "cashier", StoreID: &storeID, Status: StatusActive}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/store/cashiers/:cashierID/disable", withStoreScope(99), h.StoreDisableCashier)

	req := httptest.NewRequest(http.MethodPost, "/store/cashiers/1/disable", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-store disable, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStoreCreateStaffAccountHandlerScopesToStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "Waiter Wang", Phone: "13800000007", Status: StatusActive}}}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/store/staff-accounts", withStoreScope(42), h.StoreCreateStaffAccount)

	body := `{"memberId":7}`
	req := httptest.NewRequest(http.MethodPost, "/store/staff-accounts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(repo.staffAccounts) != 1 {
		t.Fatalf("expected 1 staff account stored, got %d", len(repo.staffAccounts))
	}
	for _, a := range repo.staffAccounts {
		if a.StoreID != 42 {
			t.Fatalf("expected staff account pinned to store 42, got %+v", a)
		}
	}
}

func TestStoreDisableStaffAccountHandlerCannotReachOtherStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	repo.staffAccounts = map[int64]StaffAccount{1: {ID: 1, Name: "Waiter Wang", StoreID: 42, Status: StatusActive}}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/store/staff-accounts/:staffID/disable", withStoreScope(99), h.StoreDisableStaffAccount)

	req := httptest.NewRequest(http.MethodPost, "/store/staff-accounts/1/disable", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-store disable, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminCreateStaffAccountHandlerUsesBodyStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "Waiter Wang", Phone: "13800000007", Status: StatusActive}}}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/admin/staff-accounts", h.AdminCreateStaffAccount)

	body := `{"storeId":42,"memberId":7}`
	req := httptest.NewRequest(http.MethodPost, "/admin/staff-accounts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(repo.staffAccounts) != 1 {
		t.Fatalf("expected 1 staff account stored, got %d", len(repo.staffAccounts))
	}
	for _, a := range repo.staffAccounts {
		if a.StoreID != 42 {
			t.Fatalf("expected staff account pinned to store 42, got %+v", a)
		}
	}
}

func TestCreateStoreAdminAccountRequiresPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(NewService(&fakeRepo{}, fakeStores{}, nil))
	router := gin.New()
	router.POST("/admin/store-admin-accounts", h.CreateStoreAdminAccount)

	body := `{"storeId":42,"username":"store-admin","displayName":"Store Admin"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/store-admin-accounts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateAdminAccountAcceptsPasswordWithoutReturningIt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.POST("/admin/admin-accounts", h.CreateAdminAccount)

	body := `{"username":"ops-admin","password":"secret","displayName":"Ops Admin"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/admin-accounts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "password") {
		t.Fatalf("response leaked password data: %s", rec.Body.String())
	}
}

func TestUpdateSystemAdminAccountAcceptsNewPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "superadmin", Role: "super_admin", IsSystem: true, Status: StatusActive},
	}}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.PATCH("/admin/admin-accounts/:accountID", h.UpdateAdminAccount)

	body := `{"password":"new-secret"}`
	req := httptest.NewRequest(http.MethodPatch, "/admin/admin-accounts/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteAdminAccountProtectsSystemAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "superadmin", Role: "super_admin", IsSystem: true, Status: StatusActive},
	}}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.DELETE("/admin/admin-accounts/:accountID", h.DeleteAdminAccount)

	req := httptest.NewRequest(http.MethodDelete, "/admin/admin-accounts/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateStoreAdminAccountAcceptsNewPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	storeID := int64(42)
	repo := &fakeRepo{adminAccounts: map[int64]AdminAccount{
		1: {ID: 1, Username: "store-admin", DisplayName: "Store Admin", Role: "store_admin", StoreID: &storeID, Status: StatusActive},
	}}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.PATCH("/admin/store-admin-accounts/:accountID", h.UpdateStoreAdminAccount)

	body := `{"password":"new-secret"}`
	req := httptest.NewRequest(http.MethodPatch, "/admin/store-admin-accounts/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.lastPasswordHash == "" || repo.lastPasswordHash == "new-secret" {
		t.Fatal("expected a non-plaintext password hash")
	}
}

func TestAdminUpdateStaffAccountHandlerReachesAnyStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	repo.staffAccounts = map[int64]StaffAccount{1: {ID: 1, Name: "Waiter Wang", StoreID: 42, Status: StatusActive}}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.PATCH("/admin/staff-accounts/:staffID", h.AdminUpdateStaffAccount)

	body := `{"storeId":99,"name":"Waiter Wang Jr"}`
	req := httptest.NewRequest(http.MethodPatch, "/admin/staff-accounts/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.staffAccounts[1].StoreID != 99 || repo.staffAccounts[1].Name != "Waiter Wang Jr" {
		t.Fatalf("expected staff account reassigned to store 99, got %+v", repo.staffAccounts[1])
	}
}

func TestMemberDetailHandlerNotScopedToStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "n", Status: StatusActive}}}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/admin/members/:memberID", h.MemberDetail)

	req := httptest.NewRequest(http.MethodGet, "/admin/members/7", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":7`) {
		t.Fatalf("expected member data in response, got: %s", rec.Body.String())
	}
}

func TestMemberDetailHandlerReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{members: map[int64]Member{}}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/admin/members/:memberID", h.MemberDetail)

	req := httptest.NewRequest(http.MethodGet, "/admin/members/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateWalletAdjustmentHandlerRequiresIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "n", Status: StatusActive}}}
	wallets := &fakeWallets{adjusted: wallet.Account{AssetType: wallet.AssetPoints, AvailableAmount: 150}}
	svc := NewService(repo, fakeStores{}, wallets)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/admin/members/:memberID/wallet-adjustments", idempotency.Require(), h.CreateWalletAdjustment)

	body := `{"assetType":"points","direction":"credit","amount":50,"reason":"goodwill"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/members/7/wallet-adjustments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without Idempotency-Key, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateWalletAdjustmentHandlerDelegatesUnscoped(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{members: map[int64]Member{7: {ID: 7, Nickname: "n", Status: StatusActive}}}
	wallets := &fakeWallets{adjusted: wallet.Account{AssetType: wallet.AssetPoints, AvailableAmount: 150}}
	svc := NewService(repo, fakeStores{}, wallets)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/admin/members/:memberID/wallet-adjustments", idempotency.Require(), h.CreateWalletAdjustment)

	body := `{"assetType":"points","direction":"credit","amount":50,"reason":"goodwill"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/members/7/wallet-adjustments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if wallets.lastMemberID != 7 || wallets.lastIdemKey != "idem-1" {
		t.Fatalf("expected adjustment delegated with memberID 7 and idem key, got %+v", wallets)
	}
}

func TestAdminDisableStaffAccountHandlerReachesAnyStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	repo.staffAccounts = map[int64]StaffAccount{1: {ID: 1, Name: "Waiter Wang", StoreID: 42, Status: StatusActive}}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/admin/staff-accounts/:staffID/disable", h.AdminDisableStaffAccount)

	req := httptest.NewRequest(http.MethodPost, "/admin/staff-accounts/1/disable", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.staffAccounts[1].Status != "disabled" {
		t.Fatalf("expected staff account disabled, got %+v", repo.staffAccounts[1])
	}
}

func TestAdminDeleteStaffBindingRouteRemovesBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{
		staffAccounts: map[int64]StaffAccount{
			1: {ID: 1, Name: "Waiter Wang", StoreID: 42, Status: StatusActive},
		},
	}
	h := NewHandler(NewService(repo, fakeStores{}, nil))

	router := gin.New()
	router.DELETE("/admin/staff-accounts/:staffID/binding", h.AdminDeleteStaffAccount)

	req := httptest.NewRequest(http.MethodDelete, "/admin/staff-accounts/1/binding", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, ok := repo.staffAccounts[1]; ok {
		t.Fatal("expected staff binding to be removed")
	}
}
