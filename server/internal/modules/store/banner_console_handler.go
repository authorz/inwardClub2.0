package store

import (
	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// BannerConsoleHandler exposes banner CRUD for both the admin (global + any
// store) and store (own-store scoped) consoles. Route wiring decides which
// group mounts which method.
type BannerConsoleHandler struct {
	svc *BannerConsoleService
}

// NewBannerConsoleHandler builds the banner console handler.
func NewBannerConsoleHandler(svc *BannerConsoleService) *BannerConsoleHandler {
	return &BannerConsoleHandler{svc: svc}
}

// --- Admin ---

// AdminList handles GET /admin/banners.
func (h *BannerConsoleHandler) AdminList(c *gin.Context) {
	views, err := h.svc.AdminList(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// AdminGet handles GET /admin/banners/:id.
func (h *BannerConsoleHandler) AdminGet(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.AdminGet(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminCreate handles POST /admin/banners.
func (h *BannerConsoleHandler) AdminCreate(c *gin.Context) {
	var in BannerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.AdminCreate(c.Request.Context(), in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminUpdate handles PATCH /admin/banners/:id.
func (h *BannerConsoleHandler) AdminUpdate(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var patch BannerPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.AdminUpdate(c.Request.Context(), id, patch)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminDelete handles DELETE /admin/banners/:id.
func (h *BannerConsoleHandler) AdminDelete(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.AdminDelete(c.Request.Context(), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"id": id})
}

// --- Store ---

// StoreList handles GET /store/banners.
func (h *BannerConsoleHandler) StoreList(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	views, err := h.svc.StoreList(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// StoreGet handles GET /store/banners/:id.
func (h *BannerConsoleHandler) StoreGet(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.StoreGet(c.Request.Context(), storeID, id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreCreate handles POST /store/banners. The banner is pinned to the
// caller's own store scope.
func (h *BannerConsoleHandler) StoreCreate(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	var in BannerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.StoreCreate(c.Request.Context(), storeID, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreUpdate handles PATCH /store/banners/:id.
func (h *BannerConsoleHandler) StoreUpdate(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var patch BannerPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.StoreUpdate(c.Request.Context(), storeID, id, patch)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// StoreDelete handles DELETE /store/banners/:id.
func (h *BannerConsoleHandler) StoreDelete(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	id, err := pathID(c, "id")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.StoreDelete(c.Request.Context(), storeID, id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"id": id})
}
