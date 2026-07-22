package wallet

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
)

// Handler exposes the mini-program wallet endpoints: the read paths (wallet +
// ledger) and the points write paths (sign-in, savings, withdrawal). The member
// identity always comes from the token, never the request body.
type Handler struct {
	svc    *Service
	points *PointsService
}

// NewHandler builds the wallet handler. The points service is optional so the
// existing read-only wiring (NewHandler(svc)) keeps compiling; pass it to enable
// the sign-in and points-savings endpoints.
func NewHandler(svc *Service, points ...*PointsService) *Handler {
	h := &Handler{svc: svc}
	if len(points) > 0 {
		h.points = points[0]
	}
	return h
}

// Get handles GET /mini/wallet.
func (h *Handler) Get(c *gin.Context) {
	claims := authn.MustFromContext(c)
	accounts, err := h.svc.GetWallet(c.Request.Context(), claims.SubjectID())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, accounts)
}

// Ledger handles GET /mini/wallet/ledger.
func (h *Handler) Ledger(c *gin.Context) {
	claims := authn.MustFromContext(c)
	page := httpx.ParsePage(c)
	assetType := c.Query("assetType")
	if assetType == "" {
		assetType = c.Query("asset")
	}
	entries, total, err := h.svc.Ledger(c.Request.Context(), claims.SubjectID(), assetType, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, entries, httpx.MetaFor(page, total))
}

// SignIn handles POST /mini/sign-ins.
func (h *Handler) SignIn(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	res, err := h.points.SignIn(c.Request.Context(), memberID, idempotency.Key(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, res)
}

// SavePoints handles POST /mini/point-savings.
func (h *Handler) SavePoints(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	var req PointsAmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	res, err := h.points.SavePoints(c.Request.Context(), memberID, req, idempotency.Key(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, res)
}

// WithdrawPoints handles POST /mini/point-withdrawals.
func (h *Handler) WithdrawPoints(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	var req PointsAmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	res, err := h.points.WithdrawPoints(c.Request.Context(), memberID, req, idempotency.Key(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, res)
}
