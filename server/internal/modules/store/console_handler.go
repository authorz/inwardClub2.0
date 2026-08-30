package store

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/audit"
	"github.com/inwardclub/server/internal/platform/authn"
	"github.com/inwardclub/server/internal/platform/credentialcrypto"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// ConsoleHandler exposes the store console's own-store profile endpoints and
// the admin-side store create/update/delete contract. Route wiring decides which
// group (store-console vs admin) mounts which method.
type ConsoleHandler struct {
	svc         *ConsoleService
	credentials credentialcrypto.Decryptor
}

// NewConsoleHandler builds the store console handler.
func NewConsoleHandler(svc *ConsoleService, credentials ...credentialcrypto.Decryptor) *ConsoleHandler {
	h := &ConsoleHandler{svc: svc}
	if len(credentials) > 0 {
		h.credentials = credentials[0]
	}
	return h
}

// GetOwnProfile handles GET /store/profile-console. The store id comes only
// from the pinned request scope, never from the client.
func (h *ConsoleHandler) GetOwnProfile(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	view, err := h.svc.GetProfile(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// UpdateOwnProfile handles PUT /store/profile-console. Any storeId in the
// request body is ignored; the store id comes only from the pinned scope.
func (h *ConsoleHandler) UpdateOwnProfile(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateProfile(c.Request.Context(), storeID, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// UpdateOwnStatus handles PATCH /store/profile/status. Only the status field
// is applied; the store id comes only from the pinned scope.
func (h *ConsoleHandler) UpdateOwnStatus(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	var req StoreStatusPatch
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateStatus(c.Request.Context(), storeID, req.Status)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// GetOwnSettings handles GET /store/settings.
func (h *ConsoleHandler) GetOwnSettings(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	view, err := h.svc.GetSettings(c.Request.Context(), storeID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// UpdateOwnSettings handles PUT /store/settings. Any storeId in the request
// body is ignored; the store id comes only from the pinned scope.
func (h *ConsoleHandler) UpdateOwnSettings(c *gin.Context) {
	storeID, ok := storescope.MustFromContext(c)
	if !ok {
		return
	}
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateSettings(c.Request.Context(), storeID, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminGetStore handles GET /admin/stores/:storeID. Admin reads are not
// scoped by the caller's own store; any store id may be requested.
func (h *ConsoleHandler) AdminGetStore(c *gin.Context) {
	id, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetProfile(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminGetSettings handles GET /admin/stores/:storeID/settings.
func (h *ConsoleHandler) AdminGetSettings(c *gin.Context) {
	id, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.GetSettings(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminUpdateSettings handles PUT /admin/stores/:storeID/settings. Full
// replace of the store's settings blob, same contract as the store-console
// path but targeting an admin-chosen store id.
func (h *ConsoleHandler) AdminUpdateSettings(c *gin.Context) {
	id, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	view, err := h.svc.UpdateSettings(c.Request.Context(), id, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminCreateStore handles the admin-side store creation contract.
func (h *ConsoleHandler) AdminCreateStore(c *gin.Context) {
	var in StoreInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	st, err := h.svc.CreateStore(c.Request.Context(), in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.ProfileView(c.Request.Context(), st)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminUpdateStore handles the admin-side store update contract.
func (h *ConsoleHandler) AdminUpdateStore(c *gin.Context) {
	id, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var in StoreInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, apperr.Invalid(err.Error()))
		return
	}
	st, err := h.svc.UpdateStore(c.Request.Context(), id, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	view, err := h.svc.ProfileView(c.Request.Context(), st)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// AdminDeleteStore handles DELETE /admin/stores/:storeID. The current
// headquarters administrator's password is required in the request body.
func (h *ConsoleHandler) AdminDeleteStore(c *gin.Context) {
	id, err := pathID(c, "storeID")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	var req DeleteStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("请输入管理员登录密码"))
		return
	}
	password, err := credentialcrypto.DecryptPassword(h.credentials, req.PasswordKeyID, req.PasswordCiphertext)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	req.Password = password
	claims := authn.MustFromContext(c)
	auditEntry := audit.FromContext(c, "store.delete", "store", id)
	if err := h.svc.DeleteStore(c.Request.Context(), id, claims.SubjectID(), req.Password, auditEntry); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}
