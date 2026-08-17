package payment

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	"github.com/inwardclub/server/internal/platform/credentialcrypto"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
)

// AdminHandler exposes the admin-console payment endpoints. Unlike the store
// console, admin requests are not scoped by a token store_id.
type AdminHandler struct {
	svc         *AdminService
	credentials credentialcrypto.Decryptor
}

// NewAdminHandler builds the admin payment handler.
func NewAdminHandler(svc *AdminService, credentials ...credentialcrypto.Decryptor) *AdminHandler {
	h := &AdminHandler{svc: svc}
	if len(credentials) > 0 {
		h.credentials = credentials[0]
	}
	return h
}

// CreateRefund handles POST /api/v2/admin/refunds.
func (h *AdminHandler) CreateRefund(c *gin.Context) {
	claims := authn.MustFromContext(c)
	var req CreateRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	password, err := credentialcrypto.DecryptPassword(h.credentials, req.PasswordKeyID, req.PasswordCiphertext)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	req.Password = password
	view, err := h.svc.CreateRefund(c.Request.Context(),
		string(claims.SubjectType), claims.SubjectID(), idempotency.Key(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// ListPaymentOrders handles GET /admin/payment-orders. An optional ?storeId
// filter narrows the result to a single store; ?status filters by payment
// order status.
func (h *AdminHandler) ListPaymentOrders(c *gin.Context) {
	f := PaymentOrderFilter{Page: httpx.ParsePage(c), Status: c.Query("status")}
	if raw := c.Query("storeId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			httpx.Fail(c, apperr.Invalid("invalid storeId"))
			return
		}
		f.StoreID = &id
	}
	views, total, err := h.svc.ListPaymentOrders(c.Request.Context(), f)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(f.Page, total))
}

// GetPaymentOrder handles GET /admin/payment-orders/{paymentOrderID}. Not
// scoped to any store.
func (h *AdminHandler) GetPaymentOrder(c *gin.Context) {
	id, err := pathID(c, "paymentOrderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetPaymentOrder(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// PaymentChannelSettings handles GET /api/v2/admin/payment-channel-settings.
func (h *AdminHandler) PaymentChannelSettings(c *gin.Context) {
	httpx.OK(c, h.svc.ListChannelSettings(c.Request.Context()))
}

// UpdatePaymentChannelSettings handles PUT /api/v2/admin/payment-channel-settings.
func (h *AdminHandler) UpdatePaymentChannelSettings(c *gin.Context) {
	var req UpdateChannelSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	views, err := h.svc.UpdateChannelSettings(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}
