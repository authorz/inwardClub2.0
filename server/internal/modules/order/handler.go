package order

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
)

// Handler exposes the mini-program order endpoints. All routes run behind the
// mini auth middleware; the member identity comes from the token, never the body.
type Handler struct {
	svc *Service
}

// NewHandler builds the order handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ListFoodOrders handles GET /mini/food-orders.
func (h *Handler) ListFoodOrders(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListFoodOrders(c.Request.Context(), memberID, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// CreateFoodOrder handles POST /mini/food-orders.
func (h *Handler) CreateFoodOrder(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	var req CreateFoodOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.CreateFoodOrder(c.Request.Context(), memberID, idempotency.Key(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// GetFoodOrder handles GET /mini/food-orders/{orderID}.
func (h *Handler) GetFoodOrder(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	id, err := pathID(c, "orderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetFoodOrder(c.Request.Context(), memberID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// ListRechargeOrders handles GET /mini/recharge-orders.
func (h *Handler) ListRechargeOrders(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListRechargeOrders(c.Request.Context(), memberID, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// CreateRechargeOrder handles POST /mini/recharge-orders.
func (h *Handler) CreateRechargeOrder(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	var req CreateRechargeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.CreateRechargeOrder(c.Request.Context(), memberID, idempotency.Key(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

// GetRechargeOrder handles GET /mini/recharge-orders/{orderID}.
func (h *Handler) GetRechargeOrder(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	id, err := pathID(c, "orderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetRechargeOrder(c.Request.Context(), memberID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// ListActivityOrders handles GET /mini/activity-orders.
func (h *Handler) ListActivityOrders(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListActivityOrders(c.Request.Context(), memberID, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// CreateActivityOrder handles POST /mini/activity-orders.
func (h *Handler) CreateActivityOrder(c *gin.Context) {
	claims := authn.MustFromContext(c)
	var req CreateActivityOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	if err := validateActivityPayMethodForSubject(claims.SubjectType, req.PayMethod); err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.CreateActivityOrder(c.Request.Context(), claims.SubjectID(), idempotency.Key(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, view)
}

func validateActivityPayMethodForSubject(subject authn.SubjectType, payMethod string) error {
	if subject == authn.SubjectPreMember && payMethod != PayMethodWeChat {
		return apperr.Forbidden("完善会员资料后才可使用金币支付")
	}
	return nil
}

// GetActivityOrder handles GET /mini/activity-orders/{orderID}.
func (h *Handler) GetActivityOrder(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	id, err := pathID(c, "orderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetActivityOrder(c.Request.Context(), memberID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// WeChatJSAPI handles POST /mini/payment-orders/{paymentOrderID}/wechat-jsapi.
func (h *Handler) WeChatJSAPI(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	id, err := pathID(c, "paymentOrderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	resp, err := h.svc.CreateWeChatJSAPI(c.Request.Context(), memberID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, resp)
}

// PayByCoin handles POST /mini/payment-orders/{paymentOrderID}/pay-by-coin.
func (h *Handler) PayByCoin(c *gin.Context) {
	memberID := authn.MustFromContext(c).SubjectID()
	id, err := pathID(c, "paymentOrderID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.PayByCoin(c.Request.Context(), memberID, id, idempotency.Key(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}

func pathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid " + name)
	}
	return id, nil
}
