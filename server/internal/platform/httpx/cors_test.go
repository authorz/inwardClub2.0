package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/httpx"
)

func newCORSEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpx.DevCORS())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func TestDevCORS_PreflightFromAllowedOrigin(t *testing.T) {
	r := newCORSEngine()

	for _, origin := range []string{"http://127.0.0.1:5182", "http://127.0.0.1:5183"} {
		req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "GET")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("origin %s: preflight status = %d, want 204", origin, w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("origin %s: Allow-Origin = %q, want reflected origin", origin, got)
		}
		if got := w.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type, X-Request-ID, X-Admin-App, X-Store-App, Idempotency-Key" {
			t.Errorf("origin %s: Allow-Headers = %q", origin, got)
		}
		if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, PUT, PATCH, DELETE, OPTIONS" {
			t.Errorf("origin %s: Allow-Methods = %q", origin, got)
		}
	}
}

func TestDevCORS_ActualRequestReflectsOrigin(t *testing.T) {
	r := newCORSEngine()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5182")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK || w.Body.String() != "pong" {
		t.Fatalf("handler did not run: status=%d body=%q", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5182" {
		t.Errorf("Allow-Origin = %q, want reflected origin", got)
	}
}

func TestDevCORS_DisallowedOriginGetsNoCORS(t *testing.T) {
	r := newCORSEngine()

	cases := []string{"https://evil.example.com", "http://127.0.0.1:5999", ""}
	for _, origin := range cases {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("origin %q: status = %d, want 200 (pass-through)", origin, w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("origin %q: Allow-Origin = %q, want empty (no CORS)", origin, got)
		}
	}
}
