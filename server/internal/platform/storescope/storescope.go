// Package storescope enforces that a store console's data range comes only from
// the JWT, never from a client-supplied storeId. Store handlers read the scope
// through this package; repositories filter every query by it.
package storescope

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Inject reads the token's store_id and pins it as the request scope. It must
// run after RequireAuth on store-scoped route groups. Any storeId in the URL,
// query or body is deliberately ignored.
func Inject() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := authn.FromContext(c)
		if !ok {
			httpx.Fail(c, apperr.Unauthenticated("authentication required"))
			return
		}
		if !claims.HasStore() {
			httpx.Fail(c, apperr.New(apperr.CodeStoreScopeRequired, "store scope required"))
			return
		}
		c.Set(httpx.CtxStoreScope, claims.StoreID)
		c.Next()
	}
}

// FromContext returns the pinned store scope, or false if none was injected.
func FromContext(c *gin.Context) (int64, bool) {
	v, ok := c.Get(httpx.CtxStoreScope)
	if !ok {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

// MustFromContext returns the store scope or fails the request. Store handlers
// use this so a missing scope can never silently widen a query.
func MustFromContext(c *gin.Context) (int64, bool) {
	id, ok := FromContext(c)
	if !ok {
		httpx.Fail(c, apperr.New(apperr.CodeStoreScopeRequired, "store scope required"))
		return 0, false
	}
	return id, true
}
