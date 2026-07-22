package bootstrap

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	platdb "github.com/inwardclub/server/internal/platform/db"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// healthHandler serves liveness and readiness checks.
type healthHandler struct {
	db *platdb.DB
}

// Live always returns ok; it proves the process is up.
func (h *healthHandler) Live(c *gin.Context) {
	httpx.OK(c, gin.H{"status": "ok", "time": time.Now().UTC()})
}

// Ready pings the database so load balancers only route when dependencies are up.
func (h *healthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	status := "ok"
	if h.db != nil {
		if err := h.db.PingContext(ctx); err != nil {
			status = "degraded"
		}
	}
	httpx.OK(c, gin.H{"status": status})
}
