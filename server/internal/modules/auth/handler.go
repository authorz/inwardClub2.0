package auth

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Handler exposes authentication endpoints for all three consoles.
type Handler struct {
	svc *Service
}

// NewHandler builds the auth handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// MiniLogin handles POST /mini/auth/wechat/login.
func (h *Handler) MiniLogin(c *gin.Context) {
	var req WeChatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	resp, err := h.svc.MiniLogin(c.Request.Context(), req.Code)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, resp)
}

// AdminLogin handles POST /admin/auth/login.
func (h *Handler) AdminLogin(c *gin.Context) {
	var req PasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	resp, err := h.svc.AdminLogin(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, resp)
}

// StoreLogin handles POST /store/auth/login.
func (h *Handler) StoreLogin(c *gin.Context) {
	var req PasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	resp, err := h.svc.StoreLogin(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, resp)
}

// MiniRefresh handles POST /mini/auth/refresh.
func (h *Handler) MiniRefresh(c *gin.Context) { h.refresh(c, authn.AudienceMini) }

// AdminRefresh handles POST /admin/auth/refresh.
func (h *Handler) AdminRefresh(c *gin.Context) { h.refresh(c, authn.AudienceAdmin) }

// StoreRefresh handles POST /store/auth/refresh.
func (h *Handler) StoreRefresh(c *gin.Context) { h.refresh(c, authn.AudienceStore) }

func (h *Handler) refresh(c *gin.Context, audience authn.Audience) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken, audience)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, pair)
}

// MiniMe handles GET /mini/me.
func (h *Handler) MiniMe(c *gin.Context) {
	claims := authn.MustFromContext(c)
	profile, err := h.svc.MemberProfile(c.Request.Context(), claims.SubjectID())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, profile)
}

// AdminMe handles GET /admin/auth/me.
func (h *Handler) AdminMe(c *gin.Context) { h.accountMe(c) }

// StoreMe handles GET /store/auth/me.
func (h *Handler) StoreMe(c *gin.Context) { h.accountMe(c) }

func (h *Handler) accountMe(c *gin.Context) {
	claims := authn.MustFromContext(c)
	profile, err := h.svc.AccountProfile(c.Request.Context(), claims.SubjectID())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, profile)
}

// MiniLogout handles POST /mini/auth/logout.
func (h *Handler) MiniLogout(c *gin.Context) {
	claims := authn.MustFromContext(c)
	if err := h.svc.LogoutMember(c.Request.Context(), claims.SubjectID()); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}

// AccountLogout handles POST /{admin,store}/auth/logout.
func (h *Handler) AccountLogout(c *gin.Context) {
	claims := authn.MustFromContext(c)
	if err := h.svc.LogoutAccount(c.Request.Context(), claims.SubjectID()); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}
