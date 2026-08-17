package store

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Handler exposes the mini-program store read endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the store handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// List handles GET /mini/stores?lat=&lng=.
func (h *Handler) List(c *gin.Context) {
	page := httpx.ParsePage(c)
	geo := parseGeo(c)
	views, total, err := h.svc.ListStores(c.Request.Context(), page, geo)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// Detail handles GET /mini/stores/{storeID}.
func (h *Handler) Detail(c *gin.Context) {
	id, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetStore(c.Request.Context(), id, parseGeo(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

func parseGeo(c *gin.Context) Geo {
	var geo Geo
	if v, err := strconv.ParseFloat(c.Query("lat"), 64); err == nil {
		geo.Lat = &v
	}
	if v, err := strconv.ParseFloat(c.Query("lng"), 64); err == nil {
		geo.Lng = &v
	}
	return geo
}

func pathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid " + name)
	}
	return id, nil
}
