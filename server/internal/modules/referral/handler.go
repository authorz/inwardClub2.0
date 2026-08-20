package referral

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Config handles GET /mini/invitation-reward-config.
func (h *Handler) Config(c *gin.Context) {
	view, err := h.svc.Config(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}
