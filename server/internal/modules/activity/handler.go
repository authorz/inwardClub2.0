package activity

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Handler exposes the mini-program activity read endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the activity handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List handles GET /mini/activities.
func (h *Handler) List(c *gin.Context) {
	page := httpx.ParsePage(c)
	views, total, err := h.svc.List(c.Request.Context(), nil, page, c.Query("scope") == "history")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// ListForStore handles GET /mini/stores/{storeID}/activities.
func (h *Handler) ListForStore(c *gin.Context) {
	storeID, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	page := httpx.ParsePage(c)
	views, total, err := h.svc.List(c.Request.Context(), &storeID, page, c.Query("scope") == "history")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// TodayForStore handles GET /mini/stores/{storeID}/activities/today.
func (h *Handler) TodayForStore(c *gin.Context) {
	storeID, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, err := h.svc.ListToday(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// Detail handles GET /mini/activities/{activityID}.
func (h *Handler) Detail(c *gin.Context) {
	id, err := pathID(c, "activityID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.Get(c.Request.Context(), id)
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
