package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestActivitiesHandlerAcceptsStoreFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &fakeRepo{}
	h := NewHandler(NewService(repo, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/activities", h.Activities)

	req := httptest.NewRequest(http.MethodGet, "/admin/activities?storeId=42", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.lastFilter.StoreID == nil || *repo.lastFilter.StoreID != 42 {
		t.Fatalf("expected store filter 42, got %+v", repo.lastFilter)
	}
}

func TestActivitiesHandlerRejectsInvalidStoreFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(NewService(&fakeRepo{}, fakeStores{}, nil))
	router := gin.New()
	router.GET("/admin/activities", h.Activities)

	req := httptest.NewRequest(http.MethodGet, "/admin/activities?storeId=invalid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
