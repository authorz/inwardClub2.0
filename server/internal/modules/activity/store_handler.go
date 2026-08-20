package activity

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// StoreHandler exposes the store-console operational activity endpoints. The
// store scope is always taken from the token via storescope; the acting staff
// identity from the auth claims. Router wiring is owned elsewhere.
type StoreHandler struct {
	svc *StoreService
}

// NewStoreHandler builds the store-console activity handler.
func NewStoreHandler(svc *StoreService) *StoreHandler { return &StoreHandler{svc: svc} }

// VerifyTicket handles POST /api/v2/store/tickets/verify.
func (h *StoreHandler) VerifyTicket(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	claims := authn.MustFromContext(c)
	var req VerifyTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	view, err := h.svc.VerifyTicket(c.Request.Context(), storeID, req.Code, claims.SubjectID())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// ListPointSavings handles GET /api/v2/store/point-savings.
func (h *StoreHandler) ListPointSavings(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListPointSavings(
		c.Request.Context(), storeID, page, c.Query("status"), c.Query("phone"),
	)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// StaffTodayOperations handles GET /mini/staff/operations/today.
func (h *StoreHandler) StaffTodayOperations(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	page := httpx.ParsePage(c)
	view, err := h.svc.StaffTodayOperations(c.Request.Context(), storeID, page, c.Query("type"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// GetPointSaving handles GET /api/v2/store/point-savings/{requestID}.
func (h *StoreHandler) GetPointSaving(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	requestID, err := pathID(c, "requestID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetPointSaving(c.Request.Context(), storeID, requestID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// ReviewPointSaving handles POST /api/v2/store/point-savings/{requestID}/review.
func (h *StoreHandler) ReviewPointSaving(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	claims := authn.MustFromContext(c)
	requestID, err := pathID(c, "requestID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req ReviewPointSavingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	view, err := h.svc.ReviewPointSaving(
		c.Request.Context(), storeID, requestID, req, string(claims.SubjectType), claims.SubjectID(),
	)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// TodayActivities handles GET /api/v2/store/activities/today.
func (h *StoreHandler) TodayActivities(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	views, err := h.svc.TodayActivities(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// StaffHome handles GET /api/v2/store/staff/home.
func (h *StoreHandler) StaffHome(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	view, err := h.svc.StaffHome(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// ListVerifications handles GET /api/v2/store/verifications.
func (h *StoreHandler) ListVerifications(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListVerifications(
		c.Request.Context(), storeID, page, c.Query("kind"), c.Query("keyword"),
	)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}
