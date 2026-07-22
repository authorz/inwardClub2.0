package reporting

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/httpx"
)

// withStoreScope injects the given store id the way storescope.Inject would,
// without requiring a real JWT in tests.
func withStoreScope(storeID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(httpx.CtxStoreScope, storeID)
		c.Next()
	}
}

func newHandler(repo *fakeRepo) *Handler { return NewHandler(NewService(repo)) }

// TestOverviewAdminAggregatesAllStores: without a store scope in context the
// admin dashboard must query unscoped (nil StoreID).
func TestOverviewAdminAggregatesAllStores(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{out: Overview{StoreCount: 5}}
	router := gin.New()
	router.GET("/admin/reports/overview", newHandler(repo).Overview)

	req := httptest.NewRequest(http.MethodGet, "/admin/reports/overview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.last.StoreID != nil {
		t.Fatalf("expected unscoped admin overview, got store %v", *repo.last.StoreID)
	}
}

// TestOverviewStoreScopesToToken: with a store scope in context the overview
// must be pinned to that store, and a client-supplied ?storeId is ignored.
func TestOverviewStoreScopesToToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{out: Overview{StoreCount: 1}}
	router := gin.New()
	router.GET("/store/reports/overview", withStoreScope(42), newHandler(repo).Overview)

	req := httptest.NewRequest(http.MethodGet, "/store/reports/overview?storeId=99", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.last.StoreID == nil || *repo.last.StoreID != 42 {
		t.Fatalf("expected overview pinned to store 42 from token, got %v", repo.last.StoreID)
	}
}

// TestRevenueStoreScopesToTokenAndParsesWindow: the list reports read the scope
// from the token (never ?storeId) and parse ?from/?to into the filter window.
func TestRevenueStoreScopesToTokenAndParsesWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{revenue: []RevenueRow{{OrderCount: 3, GrossCent: 1500}}, total: 1}
	router := gin.New()
	router.GET("/store/reports/revenue", withStoreScope(42), newHandler(repo).Revenue)

	req := httptest.NewRequest(http.MethodGet, "/store/reports/revenue?storeId=99&from=2026-07-01&to=2026-07-18", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"grossCent":1500`) {
		t.Fatalf("expected revenue data in response, got: %s", rec.Body.String())
	}
	if repo.lastReport.StoreID == nil || *repo.lastReport.StoreID != 42 {
		t.Fatalf("expected revenue pinned to store 42 from token, got %v", repo.lastReport.StoreID)
	}
	if repo.lastReport.From == nil || repo.lastReport.To == nil {
		t.Fatalf("expected from/to window parsed, got from=%v to=%v", repo.lastReport.From, repo.lastReport.To)
	}
}

// TestCouponsStoreScopesToToken guards the store coupons report wiring.
func TestCouponsStoreScopesToToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{coupons: []CouponStat{{TemplateID: 9, Name: "welcome", Issued: 10, Redeemed: 4}}, total: 1}
	router := gin.New()
	router.GET("/store/reports/coupons", withStoreScope(42), newHandler(repo).Coupons)

	req := httptest.NewRequest(http.MethodGet, "/store/reports/coupons", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "welcome") {
		t.Fatalf("expected coupon data in response, got: %s", rec.Body.String())
	}
	if repo.lastReport.StoreID == nil || *repo.lastReport.StoreID != 42 {
		t.Fatalf("expected coupons pinned to store 42 from token, got %v", repo.lastReport.StoreID)
	}
}
