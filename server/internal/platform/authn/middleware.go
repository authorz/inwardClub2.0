package authn

import (
	"context"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// TokenVersionChecker reports the current token_version for a subject so
// RequireAuth can reject access tokens minted before a logout (or account
// disable / password reset) bumped it. The refresh path already performs this
// comparison; wiring a checker extends the same invalidation to access tokens.
// Implementations are backed by the member/account stores and are audience-
// specific (member-backed for mini, account-backed for admin/store).
type TokenVersionChecker interface {
	CurrentTokenVersion(ctx context.Context, subjectType SubjectType, subjectID int64) (int64, error)
}

// Middleware verifies bearer tokens for a fixed audience and stores the claims
// in the gin context. One Middleware instance is created per audience so the
// three consoles cannot accept each other's tokens.
type Middleware struct {
	manager  *Manager
	audience Audience
	versions TokenVersionChecker
}

// Option configures a Middleware at construction time.
type Option func(*Middleware)

// WithTokenVersions makes RequireAuth compare each access token's token_version
// against the subject's current stored value, rejecting tokens invalidated by a
// logout. Omitting it preserves the previous stateless behaviour.
func WithTokenVersions(checker TokenVersionChecker) Option {
	return func(m *Middleware) { m.versions = checker }
}

// NewMiddleware builds an auth middleware bound to a single audience.
func NewMiddleware(manager *Manager, audience Audience, opts ...Option) *Middleware {
	m := &Middleware{manager: manager, audience: audience}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// RequireAuth validates the access token and optionally restricts which subject
// types may pass. An empty allowed list accepts any subject valid for the audience.
func (m *Middleware) RequireAuth(allowed ...SubjectType) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c)
		if raw == "" {
			httpx.Fail(c, apperr.Unauthenticated("missing bearer token"))
			return
		}
		claims, err := m.manager.Parse(raw, m.audience)
		if err != nil {
			httpx.Fail(c, apperr.Unauthenticated("invalid or expired token"))
			return
		}
		if claims.Kind != TokenAccess {
			httpx.Fail(c, apperr.Unauthenticated("access token required"))
			return
		}
		if len(allowed) > 0 && !slices.Contains(allowed, claims.SubjectType) {
			httpx.Fail(c, apperr.Forbidden("subject type not permitted for this endpoint"))
			return
		}
		// Store-scoped audiences must carry a concrete store; admin must not.
		if m.audience == AudienceStore && !claims.HasStore() {
			httpx.Fail(c, apperr.New(apperr.CodeStoreScopeRequired, "store token missing store scope"))
			return
		}
		if m.audience == AudienceAdmin && claims.SubjectType != SubjectSuperAdmin {
			httpx.Fail(c, apperr.Forbidden("admin console requires super_admin"))
			return
		}
		// Stale-token check: an access token minted before a logout carries an
		// outdated token_version. This is the last gate because it is the only
		// one that touches the datastore; all cheaper structural checks run first.
		if m.versions != nil {
			current, err := m.versions.CurrentTokenVersion(
				c.Request.Context(), claims.SubjectType, claims.SubjectID(),
			)
			if err != nil {
				if apperr.From(err).Code == apperr.CodeNotFound {
					httpx.Fail(c, apperr.Unauthenticated("session expired"))
				} else {
					httpx.Fail(c, apperr.From(err))
				}
				return
			}
			if current != claims.TokenVersion {
				httpx.Fail(c, apperr.Unauthenticated("session expired"))
				return
			}
		}
		c.Set(httpx.CtxClaims, claims)
		c.Next()
	}
}

// OptionalAuth attaches valid access-token claims when the caller is logged in,
// while keeping the endpoint available to anonymous callers. Invalid, expired
// or stale tokens are treated as anonymous instead of turning a public request
// into a 401 response.
func (m *Middleware) OptionalAuth(allowed ...SubjectType) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c)
		if raw == "" {
			c.Next()
			return
		}
		claims, err := m.manager.Parse(raw, m.audience)
		if err != nil || claims.Kind != TokenAccess {
			c.Next()
			return
		}
		if len(allowed) > 0 && !slices.Contains(allowed, claims.SubjectType) {
			c.Next()
			return
		}
		if m.audience == AudienceStore && !claims.HasStore() {
			c.Next()
			return
		}
		if m.audience == AudienceAdmin && claims.SubjectType != SubjectSuperAdmin {
			c.Next()
			return
		}
		if m.versions != nil {
			current, err := m.versions.CurrentTokenVersion(
				c.Request.Context(), claims.SubjectType, claims.SubjectID(),
			)
			if err != nil || current != claims.TokenVersion {
				c.Next()
				return
			}
		}
		c.Set(httpx.CtxClaims, claims)
		c.Next()
	}
}

// FromContext returns the authenticated claims from the context.
func FromContext(c *gin.Context) (*Claims, bool) {
	v, ok := c.Get(httpx.CtxClaims)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*Claims)
	return claims, ok
}

// MustFromContext returns claims or panics; only call after RequireAuth.
func MustFromContext(c *gin.Context) *Claims {
	claims, ok := FromContext(c)
	if !ok {
		panic("authn: claims missing from context")
	}
	return claims
}

func bearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
