package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// fakeVersions is a canned TokenVersionChecker for exercising the stale-token
// gate without a datastore.
type fakeVersions struct {
	current int64
	err     error
}

func (f fakeVersions) CurrentTokenVersion(context.Context, SubjectType, int64) (int64, error) {
	return f.current, f.err
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
