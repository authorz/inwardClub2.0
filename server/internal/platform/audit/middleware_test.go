package audit

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/inwardclub/server/internal/platform/authn"
	"github.com/inwardclub/server/internal/platform/httpx"
)

type fakeRequestRecorder struct {
	entries []Entry
}

func (f *fakeRequestRecorder) Record(_ context.Context, entry Entry) error {
	f.entries = append(f.entries, entry)
	return nil
}

func TestStoreWritesAuditsOnlySuccessfulMutations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name       string
		method     string
		statusCode int
		want       int
	}{
		{"successful patch", http.MethodPatch, http.StatusOK, 1},
		{"failed patch", http.MethodPatch, http.StatusBadRequest, 0},
		{"read", http.MethodGet, http.StatusOK, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &fakeRequestRecorder{}
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(httpx.CtxRequestID, "req-1")
				c.Set(httpx.CtxClaims, &authn.Claims{
					SubjectType:      authn.SubjectStoreAdmin,
					Role:             authn.RoleStoreAdmin,
					StoreID:          42,
					RegisteredClaims: jwt.RegisteredClaims{Subject: "7"},
				})
				c.Next()
			}, StoreWrites(recorder, slog.New(slog.NewTextHandler(io.Discard, nil))))
			router.Handle(tc.method, "/api/v2/store/profile", func(c *gin.Context) { c.Status(tc.statusCode) })
			request := httptest.NewRequest(tc.method, "/api/v2/store/profile", nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if len(recorder.entries) != tc.want {
				t.Fatalf("entries=%d, want %d", len(recorder.entries), tc.want)
			}
			if tc.want == 1 {
				entry := recorder.entries[0]
				if entry.ActorID != 7 || entry.StoreID != 42 || entry.TargetID != 42 || entry.Action != "store_console.write" {
					t.Fatalf("unexpected audit entry: %+v", entry)
				}
			}
		})
	}
}

func TestStoreWritesSkipsRequestAlreadyAuditedInTransaction(t *testing.T) {
	recorder := &fakeRequestRecorder{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(httpx.CtxRequestID, "req-1")
		c.Next()
	}, StoreWrites(recorder, slog.New(slog.NewTextHandler(io.Discard, nil))))
	router.POST("/write", func(c *gin.Context) {
		_ = FromContext(c, "specific.write", "store", 42)
		c.Status(http.StatusNoContent)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/write", nil))
	if len(recorder.entries) != 0 {
		t.Fatalf("expected no duplicate audit, got %+v", recorder.entries)
	}
}
