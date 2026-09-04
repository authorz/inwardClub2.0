package authn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// fakeVersions is a canned TokenVersionChecker for exercising the stale-token
// gate without a datastore.
type fakeVersions struct {
	current    int64
	externalID string
	err        error
}

func (f fakeVersions) CurrentTokenVersion(context.Context, SubjectType, int64, int64) (int64, error) {
	return f.current, f.err
}

func (f fakeVersions) ExternalID(context.Context, SubjectType, int64) (string, error) {
	return f.externalID, f.err
}

func setupRouter(aud Audience, allowed ...SubjectType) (*gin.Engine, *Manager) {
	gin.SetMode(gin.TestMode)
	mgr := NewManager("k", "inwardclub", time.Hour, time.Hour)
	mw := NewMiddleware(mgr, aud)
	r := gin.New()
	r.GET("/protected", mw.RequireAuth(allowed...), func(c *gin.Context) {
		claims := MustFromContext(c)
		c.JSON(200, gin.H{"sub": claims.SubjectID()})
	})
	return r, mgr
}

func doGet(r *gin.Engine, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireAuth_MissingToken(t *testing.T) {
	r, _ := setupRouter(AudienceMini)
	if got := doGet(r, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", got)
	}
}

func TestRequireAuth_WrongAudienceRejected(t *testing.T) {
	r, _ := setupRouter(AudienceStore)
	other := NewManager("k", "inwardclub", time.Hour, time.Hour)
	pair, _ := other.Issue(Identity{SubjectID: 1, SubjectType: SubjectMember, Role: RoleMember, Audience: AudienceMini})
	if got := doGet(r, pair.AccessToken).Code; got != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong audience, got %d", got)
	}
}

func TestRequireAuth_SubjectTypeRestricted(t *testing.T) {
	r, mgr := setupRouter(AudienceStore, SubjectStoreAdmin)
	// cashier token should be forbidden when only store_admin is allowed.
	pair, _ := mgr.Issue(Identity{SubjectID: 9, SubjectType: SubjectCashier, Role: RoleCashier, Audience: AudienceStore, StoreID: 3})
	if got := doGet(r, pair.AccessToken).Code; got != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed subject, got %d", got)
	}
}

func TestRequireAuth_PreMemberRestrictedReturnsLoginPromptAndIdentityContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := NewManager("k", "inwardclub", time.Hour, time.Hour)
	mw := NewMiddleware(mgr, AudienceMini, WithTokenVersions(fakeVersions{externalID: "openid-27"}))
	var subjectType string
	var subjectID int64
	var loggedError string
	r := gin.New()
	r.Use(httpx.RequestID())
	r.Use(func(c *gin.Context) {
		c.Next()
		if value, ok := c.Get(httpx.CtxSubjectType); ok {
			subjectType, _ = value.(string)
		}
		if value, ok := c.Get(httpx.CtxSubjectID); ok {
			subjectID, _ = value.(int64)
		}
		if len(c.Errors) > 0 {
			loggedError = c.Errors.Last().Error()
		}
	})
	r.POST("/food-orders", mw.RequireAuth(SubjectMember, SubjectStaff), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	pair, err := mgr.Issue(Identity{
		SubjectID: 27, SubjectType: SubjectPreMember, Role: RolePreMember, Audience: AudienceMini,
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/food-orders", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("X-Request-ID", "request-12345678")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Message != "你当前还未登录，请先登录（错误 ID：12345678）" {
		t.Fatalf("unexpected message %q", body.Error.Message)
	}
	if subjectType != string(SubjectPreMember) || subjectID != 27 {
		t.Fatalf("unexpected identity context: type=%q id=%d", subjectType, subjectID)
	}
	if !strings.Contains(loggedError, "pre_member member_id=27 openid=openid-27") {
		t.Fatalf("internal error log missing member id: %q", loggedError)
	}
}

func TestRequireAuth_StoreTokenNeedsScope(t *testing.T) {
	r, mgr := setupRouter(AudienceStore, SubjectStoreAdmin)
	// store_admin token without store_id must be rejected.
	pair, _ := mgr.Issue(Identity{SubjectID: 9, SubjectType: SubjectStoreAdmin, Role: RoleStoreAdmin, Audience: AudienceStore})
	if got := doGet(r, pair.AccessToken).Code; got != http.StatusForbidden {
		t.Fatalf("expected 403 for missing store scope, got %d", got)
	}
}

func TestRequireAuth_HappyPath(t *testing.T) {
	r, mgr := setupRouter(AudienceStore, SubjectStoreAdmin)
	pair, _ := mgr.Issue(Identity{SubjectID: 9, SubjectType: SubjectStoreAdmin, Role: RoleStoreAdmin, Audience: AudienceStore, StoreID: 3})
	if got := doGet(r, pair.AccessToken).Code; got != http.StatusOK {
		t.Fatalf("expected 200, got %d", got)
	}
}

func TestOptionalAuth_AttachesValidMemberAndAllowsAnonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := NewManager("k", "inwardclub", time.Hour, time.Hour)
	mw := NewMiddleware(mgr, AudienceMini)
	r := gin.New()
	r.GET("/optional", mw.OptionalAuth(SubjectMember), func(c *gin.Context) {
		claims, ok := FromContext(c)
		if !ok {
			c.JSON(http.StatusOK, gin.H{"memberId": 0})
			return
		}
		c.JSON(http.StatusOK, gin.H{"memberId": claims.SubjectID()})
	})

	request := func(token string) string {
		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		return w.Body.String()
	}

	if got := request(""); got != `{"memberId":0}` {
		t.Fatalf("unexpected anonymous response: %s", got)
	}
	if got := request("invalid-token"); got != `{"memberId":0}` {
		t.Fatalf("invalid token should remain anonymous: %s", got)
	}
	pair, _ := mgr.Issue(Identity{SubjectID: 9, SubjectType: SubjectMember, Role: RoleMember, Audience: AudienceMini})
	if got := request(pair.AccessToken); got != `{"memberId":9}` {
		t.Fatalf("valid member was not attached: %s", got)
	}
}

// setupVersionedRouter mirrors setupRouter but wires a TokenVersionChecker so
// the stale-token gate is exercised.
func setupVersionedRouter(aud Audience, checker TokenVersionChecker, allowed ...SubjectType) (*gin.Engine, *Manager) {
	gin.SetMode(gin.TestMode)
	mgr := NewManager("k", "inwardclub", time.Hour, time.Hour)
	mw := NewMiddleware(mgr, aud, WithTokenVersions(checker))
	r := gin.New()
	r.GET("/protected", mw.RequireAuth(allowed...), func(c *gin.Context) {
		c.JSON(200, gin.H{"sub": MustFromContext(c).SubjectID()})
	})
	return r, mgr
}

// TestRequireAuth_RejectsStaleTokenVersion is the acceptance-checklist §4.4
// guarantee: an access token minted before a logout (which bumps token_version)
// stops working, on all three consoles.
func TestRequireAuth_RejectsStaleTokenVersion(t *testing.T) {
	cases := []struct {
		name    string
		aud     Audience
		subject SubjectType
		role    Role
		storeID int64
	}{
		{"mini", AudienceMini, SubjectMember, RoleMember, 0},
		{"admin", AudienceAdmin, SubjectSuperAdmin, RoleSuperAdmin, 0},
		{"store", AudienceStore, SubjectStoreAdmin, RoleStoreAdmin, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Token minted at version 0; logout has since bumped the stored value
			// to 1, so the stored version is ahead of the token's version.
			r, mgr := setupVersionedRouter(tc.aud, fakeVersions{current: 1}, tc.subject)
			pair, _ := mgr.Issue(Identity{SubjectID: 1, SubjectType: tc.subject, Role: tc.role, Audience: tc.aud, StoreID: tc.storeID, TokenVersion: 0})
			if got := doGet(r, pair.AccessToken).Code; got != http.StatusUnauthorized {
				t.Fatalf("%s: expected 401 for stale token_version, got %d", tc.name, got)
			}

			// Accept: a freshly minted token whose version matches the stored value.
			rOK, mgrOK := setupVersionedRouter(tc.aud, fakeVersions{current: 1}, tc.subject)
			fresh, _ := mgrOK.Issue(Identity{SubjectID: 1, SubjectType: tc.subject, Role: tc.role, Audience: tc.aud, StoreID: tc.storeID, TokenVersion: 1})
			if got := doGet(rOK, fresh.AccessToken).Code; got != http.StatusOK {
				t.Fatalf("%s: expected 200 for current token_version, got %d", tc.name, got)
			}
		})
	}
}

// TestRequireAuth_RejectsDeletedSubject: a NotFound from the checker (subject
// gone) is treated as unauthenticated, not surfaced as a 404.
func TestRequireAuth_RejectsDeletedSubject(t *testing.T) {
	r, mgr := setupVersionedRouter(AudienceMini, fakeVersions{err: apperr.NotFound("member not found")}, SubjectMember)
	pair, _ := mgr.Issue(Identity{SubjectID: 1, SubjectType: SubjectMember, Role: RoleMember, Audience: AudienceMini})
	if got := doGet(r, pair.AccessToken).Code; got != http.StatusUnauthorized {
		t.Fatalf("expected 401 for deleted subject, got %d", got)
	}
}

// TestRequireAuth_VersionLookupFailureIsServerError: a datastore failure fails
// closed (never authorises the request) and surfaces as 500, not 401.
func TestRequireAuth_VersionLookupFailureIsServerError(t *testing.T) {
	r, mgr := setupVersionedRouter(AudienceAdmin, fakeVersions{err: apperr.Internal(context.DeadlineExceeded)}, SubjectSuperAdmin)
	pair, _ := mgr.Issue(Identity{SubjectID: 1, SubjectType: SubjectSuperAdmin, Role: RoleSuperAdmin, Audience: AudienceAdmin})
	if got := doGet(r, pair.AccessToken).Code; got != http.StatusInternalServerError {
		t.Fatalf("expected 500 on lookup failure, got %d", got)
	}
}
