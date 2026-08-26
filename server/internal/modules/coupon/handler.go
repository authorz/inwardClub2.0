package coupon

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
)

// Handler exposes the mini-program coupon endpoints. The member identity comes
// from the token, never the request body.
type Handler struct {
	svc *Service
}

// NewHandler builds the coupon handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ListCategories handles GET /mini/coupon-categories.
func (h *Handler) ListCategories(c *gin.Context) {
	views, err := h.svc.ListCouponCategories(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// List handles GET /mini/coupons.
func (h *Handler) List(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	page := httpx.ParsePage(c)
	var (
		views []MemberCouponView
		total int64
		err   error
	)
	if c.Query("activityId") != "" {
		var activityID int64
		activityID, err = positiveQueryID(c, "activityId")
		if err == nil {
			views, total, err = h.svc.ListActivityUsableCoupons(c.Request.Context(), memberID, activityID, page)
		}
	} else {
		views, total, err = h.svc.ListCoupons(c.Request.Context(), memberID, c.Query("status"), page)
	}
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// EligibleItems handles GET /mini/coupon-redemptions/eligible-items.
func (h *Handler) EligibleItems(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	entitlementID, err := positiveQueryID(c, "entitlementId")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	storeID, err := positiveQueryID(c, "storeId")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.ListEligibleItems(c.Request.Context(), memberID, entitlementID, storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func positiveQueryID(c *gin.Context, key string) (int64, error) {
	id, err := strconv.ParseInt(c.Query(key), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid " + key)
	}
	return id, nil
}

// Redeem handles POST /mini/coupon-redemptions.
func (h *Handler) Redeem(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	var req RedeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.Redeem(c.Request.Context(), memberID, idempotency.Key(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// UseEventCoupon handles POST /mini/event-coupon-redemptions.
func (h *Handler) UseEventCoupon(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	var req UseEventCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UseEventCoupon(c.Request.Context(), memberID, idempotency.Key(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// ListRedemptions handles GET /mini/coupon-redemptions.
func (h *Handler) ListRedemptions(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListRedemptions(c.Request.Context(), memberID, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// GetRedemption handles GET /mini/coupon-redemptions/{id}.
func (h *Handler) GetRedemption(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Fail(c, apperr.Invalid("invalid id"))
		return
	}
	view, err := h.svc.GetRedemption(c.Request.Context(), memberID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}
