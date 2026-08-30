package diagnostics

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

var requestIDSearchPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Handler exposes the admin-console diagnostics read endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the diagnostics handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ListErrorEvents handles GET /api/v2/admin/error-events.
func (h *Handler) ListErrorEvents(c *gin.Context) {
	page := httpx.ParsePage(c)
	requestID := strings.TrimSpace(c.Query("requestId"))
	if requestID != "" && !requestIDSearchPattern.MatchString(requestID) {
		httpx.Fail(c, apperr.Invalid("错误 ID 格式不正确"))
		return
	}
	events, total, err := h.svc.List(c.Request.Context(), page, requestID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, events, httpx.MetaFor(page, total))
}
