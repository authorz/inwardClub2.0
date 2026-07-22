package rbac

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Require blocks the request unless the authenticated role holds every listed
// permission. It must run after an authn RequireAuth middleware.
func Require(perms ...Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := authn.FromContext(c)
		if !ok {
			httpx.Fail(c, apperr.Unauthenticated("authentication required"))
			return
		}
		for _, p := range perms {
			if !Allowed(claims.Role, p) {
				httpx.Fail(c, apperr.Forbidden("missing permission: "+string(p)))
				return
			}
		}
		c.Next()
	}
}
