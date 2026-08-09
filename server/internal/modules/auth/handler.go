package auth

import (
	"mime"
	"path/filepath"
	"strings"

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

// MiniPreRegister silently exchanges wx.login's code. Registered members
// recover a full login session; new users receive an OpenID-backed,
// reservation-only identity.
func (h *Handler) MiniPreRegister(c *gin.Context) {
	var req WeChatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("请求内容格式不正确"))
		return
	}
	resp, err := h.svc.MiniPreRegister(c.Request.Context(), req.Code)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, resp)
}

// MiniRegister handles POST /mini/auth/wechat/register — completing a first-time
// member's profile form. It creates a new member or upgrades an OpenID-only
// pre-registration in place.
func (h *Handler) MiniRegister(c *gin.Context) {
	var req WeChatRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	resp, err := h.svc.MiniRegister(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, resp)
}

// GetPhoneMask handles POST /mini/auth/wechat/phone-mask — decrypts the WeChat
// phone code once during registration and returns the masked phone number for
// display plus a fresh register ticket carrying the authorized phone. The client
// submits the returned ticket to /register. No session required (the register
// ticket authorizes it).
func (h *Handler) GetPhoneMask(c *gin.Context) {
	var req struct {
		RegisterTicket string `json:"registerTicket" binding:"required"`
		PhoneCode      string `json:"phoneCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	masked, ticket, err := h.svc.GetPhoneMask(c.Request.Context(), req.RegisterTicket, req.PhoneCode)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, map[string]string{"phoneMasked": masked, "registerTicket": ticket})
}

// RegisterAvatar handles POST /mini/auth/wechat/register-avatar — a multipart
// upload of the first-time user's chosen avatar during registration. Authorized
// by the register ticket (form field), it uploads the file to object storage and
// returns its public https URL for the client to submit to /register.
func (h *Handler) RegisterAvatar(c *gin.Context) {
	ticket := c.PostForm("registerTicket")
	if ticket == "" {
		httpx.Fail(c, apperr.Invalid("registerTicket is required"))
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		httpx.Fail(c, apperr.Invalid("file is required"))
		return
	}
	defer file.Close()

	// Resolve the content type from the multipart part, falling back to the
	// filename extension when the client omits or mis-sets it (some mini-program
	// clients send application/octet-stream).
	contentType := header.Header.Get("Content-Type")
	if _, ok := allowedAvatarMimes[contentType]; !ok {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(header.Filename)))
	}

	url, err := h.svc.RegisterAvatar(c.Request.Context(), ticket, file, header.Size, contentType)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, map[string]string{"avatarUrl": url})
}

// allowedAvatarMimes mirrors the asset module's image MIME allowlist so the
// handler can decide whether to trust the multipart part's declared type or fall
// back to the filename extension.
var allowedAvatarMimes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
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
	if err := h.svc.LogoutMini(c.Request.Context(), claims.SubjectType, claims.SubjectID()); err != nil {
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
