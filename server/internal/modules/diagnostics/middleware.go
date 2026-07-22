package diagnostics

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/httpx"
)

// Capture returns middleware that records a 5xx response (or a handler-attached
// gin error) as an error event once the request has finished, including any
// panic already turned into a 500 by httpx.Recovery further down the chain.
func (s *Service) Capture() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		status := c.Writer.Status()
		if status < http.StatusInternalServerError && len(c.Errors) == 0 {
			return
		}
		message := ""
		if len(c.Errors) > 0 {
			message = c.Errors.Last().Error()
		}
		// The request has already completed and its response is written, so the
		// event is persisted on a fresh, bounded context: a client disconnect
		// (which cancels the request context) must not drop the diagnostic, and a
		// wedged write must not hang the handler goroutine.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.Record(ctx, httpx.RequestIDFromContext(c), c.Request.Method, c.FullPath(), status, message)
	}
}
