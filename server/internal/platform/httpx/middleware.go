package httpx

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

const requestIDHeader = "X-Request-ID"

// RequestID assigns a request ID (honouring an inbound header) and echoes it on
// the response so every log line and audit row can be correlated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(CtxRequestID, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

// Recovery converts panics into a JSON 500 envelope instead of crashing the
// process, logging the panic value with the request ID.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					slog.String("request_id", RequestIDFromContext(c)),
					slog.String("path", c.FullPath()),
					slog.Any("panic", r),
				)
				Fail(c, apperr.New(apperr.CodeInternal, "internal error"))
			}
		}()
		c.Next()
	}
}

// AccessLog logs one structured line per request after it completes.
func AccessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		attrs := []any{
			slog.String("request_id", RequestIDFromContext(c)),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("error", c.Errors.Last().Error()))
		}
		log.Info("request", attrs...)
	}
}
