package audit

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/httpx"
)

type requestRecorder interface {
	Record(ctx context.Context, entry Entry) error
}

// StoreWrites records every successful state-changing request made through the
// authenticated store console. It avoids duplicating requests already audited
// inside their own business transaction and never stores the request body, so
// credentials and other sensitive inputs cannot leak into the audit log.
func StoreWrites(recorder requestRecorder, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !stateChangingMethod(c.Request.Method) {
			c.Next()
			return
		}

		c.Next()
		statusCode := c.Writer.Status()
		if statusCode >= http.StatusBadRequest {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		requestID := httpx.RequestIDFromContext(c)
		if entryDeclared(c) {
			return
		}

		entry := FromContext(c, "store_console.write", "store", 0)
		entry.TargetID = entry.StoreID
		entry.Reason = c.Request.Method + " " + c.FullPath()
		entry.After = map[string]any{
			"method":     c.Request.Method,
			"route":      c.FullPath(),
			"statusCode": statusCode,
		}
		if err := recorder.Record(ctx, entry); err != nil {
			log.Error("failed to record store write audit", "request_id", requestID, "error", err)
		}
	}
}

func stateChangingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
