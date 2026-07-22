package payment

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/httpx"
)

// Handler exposes the internal payment callbacks. It verifies the callback
// signature via the gateway, then applies the verified notification through the
// idempotent settlement service.
type Handler struct {
	wechat     WeChatPayGateway
	offline    OfflineAcquirer
	settlement *SettlementService
}

// NewHandler builds the payment handler.
func NewHandler(wechat WeChatPayGateway, offline OfflineAcquirer, settlement *SettlementService) *Handler {
	return &Handler{wechat: wechat, offline: offline, settlement: settlement}
}

// WeChatNotify handles POST /internal/payments/wechat/notify.
func (h *Handler) WeChatNotify(c *gin.Context) {
	n, err := h.wechat.VerifyNotification(c.Request)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.settlement.SettleWeChat(c.Request.Context(), n); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}

// OfflineNotify handles POST /internal/payments/offline-acquirer/notify.
func (h *Handler) OfflineNotify(c *gin.Context) {
	n, err := h.offline.VerifyNotification(c.Request)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.settlement.SettleOffline(c.Request.Context(), n); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}
