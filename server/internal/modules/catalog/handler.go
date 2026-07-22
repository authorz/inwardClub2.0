package catalog

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Handler exposes the mini-program catalog read endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the catalog handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Categories handles GET /mini/stores/{storeID}/catalog/categories.
func (h *Handler) Categories(c *gin.Context) {
	storeID, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	views, err := h.svc.ListCategories(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// Items handles GET /mini/stores/{storeID}/catalog/items.
func (h *Handler) Items(c *gin.Context) {
	storeID, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	page := httpx.ParsePage(c)
	var categoryID *int64
	if v := c.Query("categoryId"); v != "" {
		if id, perr := strconv.ParseInt(v, 10, 64); perr == nil {
			categoryID = &id
		}
	}
	views, total, err := h.svc.ListItems(c.Request.Context(), storeID, categoryID, page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

func pathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Invalid("invalid " + name)
	}
	return id, nil
}
