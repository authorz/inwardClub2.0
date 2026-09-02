package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
)

func TestMiniReservationRoutesRejectPreMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := authn.NewManager("test-key", "inwardclub", time.Hour, time.Hour)
	middleware := authn.NewMiddleware(manager, authn.AudienceMini)
	router := gin.New()
	(&App{}).registerMini(router, middleware)
	pair, err := manager.Issue(authn.Identity{
		SubjectID: 7, SubjectType: authn.SubjectPreMember,
		Role: authn.RolePreMember, Audience: authn.AudienceMini,
	})
	if err != nil {
		t.Fatalf("issue pre-member token: %v", err)
	}

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v2/mini/reservations", ""},
		{http.MethodGet, "/api/v2/mini/reservations/1", ""},
		{http.MethodPost, "/api/v2/mini/reservations", `{}`},
		{http.MethodPost, "/api/v2/mini/reservations/1/cancel", `{}`},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
			}
		})
	}
}
