package payment

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	"github.com/inwardclub/server/internal/platform/credentialcrypto"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// StoreHandler exposes the store-console payment endpoints. The store scope is
// always taken from the JWT via storescope; router wiring lives elsewhere.
type StoreHandler struct {
	svc         *StoreService
	credentials credentialcrypto.Decryptor
}

// NewStoreHandler builds the store payment handler.
func NewStoreHandler(svc *StoreService, credentials ...credentialcrypto.Decryptor) *StoreHandler {
	h := &StoreHandler{svc: svc}
	if len(credentials) > 0 {
		h.credentials = credentials[0]
	}
	return h
}

// CreateCollectionOrder handles POST /store/offline-collection-orders.
func (h *StoreHandler) CreateCollectionOrder(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	claims := authn.MustFromContext(c)
	var req CreateCollectionOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	view, err := h.svc.CreateCollectionOrder(c.Request.Context(), storeID,
		string(claims.SubjectType), claims.SubjectID(), idempotency.Key(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// GetCollectionOrder handles GET /store/offline-collection-orders/{collectionOrderID}.
func (h *StoreHandler) GetCollectionOrder(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "collectionOrderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetCollectionOrder(c.Request.Context(), storeID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// CancelCollectionOrder handles POST /store/offline-collection-orders/{collectionOrderID}/cancel.
func (h *StoreHandler) CancelCollectionOrder(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "collectionOrderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.CancelCollectionOrder(c.Request.Context(), storeID, id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}

// CreateRefund handles POST /store/refunds.
func (h *StoreHandler) CreateRefund(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
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
	view, err := h.svc.CreateRefund(c.Request.Context(), storeID,
		string(claims.SubjectType), claims.SubjectID(), idempotency.Key(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// ListPaymentOrders handles GET /store/payment-orders. Always scoped to the
// acting store; an optional ?status filters by payment order status.
func (h *StoreHandler) ListPaymentOrders(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListPaymentOrders(c.Request.Context(), storeID, c.Query("status"), page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// GetPaymentOrder handles GET /store/payment-orders/{paymentOrderID}. Always
// scoped to the acting store.
func (h *StoreHandler) GetPaymentOrder(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "paymentOrderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetPaymentOrder(c.Request.Context(), storeID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func pathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid " + name)
	}
	return id, nil
}
