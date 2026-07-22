package diagnostics

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/httpx"
)

// Handler exposes the admin-console diagnostics read endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the diagnostics handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ListErrorEvents handles GET /api/v2/admin/error-events.
func (h *Handler) ListErrorEvents(c *gin.Context) {
	page := httpx.ParsePage(c)
	events, total, err := h.svc.List(c.Request.Context(), page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, events, httpx.MetaFor(page, total))
}
