package httpx

import "github.com/gin-gonic/gin"

// Context keys shared across middleware. Typed accessors live in the packages
// that own the value (authn for claims, storescope for scope) but the key
// strings are centralised here to avoid collisions.
const (
	CtxRequestID   = "ctx.request_id"
	CtxClaims      = "ctx.claims"
	CtxSubjectID   = "ctx.subject_id"
	CtxSubjectType = "ctx.subject_type"
	CtxStoreScope  = "ctx.store_scope"
	CtxIdemKey     = "ctx.idempotency_key"
)

// RequestIDFromContext returns the request ID assigned by the RequestID middleware.
func RequestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get(CtxRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
