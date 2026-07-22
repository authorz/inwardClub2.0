package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/modules/wallet"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
)

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

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/store/staff-accounts", withStoreScope(42), h.StoreCreateStaffAccount)

	body := `{"name":"Waiter Wang"}`
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

	repo := &fakeRepo{}
	svc := NewService(repo, fakeStores{}, nil)
	h := NewHandler(svc)

	router := gin.New()
	router.POST("/admin/staff-accounts", h.AdminCreateStaffAccount)

	body := `{"storeId":42,"name":"Waiter Wang"}`
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
