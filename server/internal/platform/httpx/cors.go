package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// devCORSOrigins is the fixed allowlist of local-development front-end origins
// permitted to make cross-origin browser requests: the admin console (Vite
// dev server on :5182) and the store console (Vite dev server on :5183). Only
// these exact loopback origins are ever reflected; every other Origin is left
// untouched, so this stays inert for real traffic.
var devCORSOrigins = map[string]bool{
	"http://127.0.0.1:5182": true,
	"http://127.0.0.1:5183": true,
}

// devCORSAllowHeaders is the exact set of request headers the two consoles send
// (see admin-console/store-console api/http.ts) that the browser must be told
// are allowed on a cross-origin request.
const devCORSAllowHeaders = "Authorization, Content-Type, X-Request-ID, X-Admin-App, X-Store-App, Idempotency-Key"

// devCORSAllowMethods lists the HTTP methods the consoles use.
const devCORSAllowMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"

// DevCORS is a minimal, conservative CORS middleware for local development. It
// only acts on requests whose Origin is in the fixed loopback allowlist; every
// other request passes through with no CORS headers added. It reflects the
// exact matched origin (never "*"), advertises only the fixed method/header
// sets, and does not enable credentials (the consoles authenticate with a
// Bearer token, not cookies). A browser preflight (OPTIONS) from an allowed
// origin is answered with 204 and short-circuits before routing.
func DevCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if !devCORSOrigins[origin] {
			c.Next()
			return
		}

		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", devCORSAllowMethods)
		c.Header("Access-Control-Allow-Headers", devCORSAllowHeaders)
		c.Header("Access-Control-Max-Age", "600")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
