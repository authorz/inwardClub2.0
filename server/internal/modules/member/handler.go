package member

import (
	"mime"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Handler exposes the mini-program member endpoints. Authenticated endpoints
// read the member id from the request claims set by the auth middleware.
type Handler struct {
	svc *Service
}

// NewHandler builds the member handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// MembershipTiers handles GET /mini/membership-tiers.
func (h *Handler) MembershipTiers(c *gin.Context) {
	views, err := h.svc.ListMembershipTiers(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// AdminMembershipTiers handles GET /admin/membership-tiers.
func (h *Handler) AdminMembershipTiers(c *gin.Context) {
	views, err := h.svc.AdminListMembershipTiers(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// AdminGetMembershipTier handles GET /admin/membership-tiers/:tierID.
func (h *Handler) AdminGetMembershipTier(c *gin.Context) {
	tierID, err := strconv.ParseInt(c.Param("tierID"), 10, 64)
	if err != nil || tierID <= 0 {
		httpx.Fail(c, apperr.Invalid("member: invalid tierID"))
		return
	}
	view, err := h.svc.AdminGetMembershipTier(c.Request.Context(), tierID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// CreateMembershipTier handles POST /admin/membership-tiers.
func (h *Handler) CreateMembershipTier(c *gin.Context) {
	var req MembershipTierCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("member: invalid tier create body"))
		return
	}
	view, err := h.svc.CreateMembershipTier(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// UpdateMembershipTier handles PATCH /admin/membership-tiers/:tierID.
func (h *Handler) UpdateMembershipTier(c *gin.Context) {
	tierID, err := strconv.ParseInt(c.Param("tierID"), 10, 64)
	if err != nil || tierID <= 0 {
		httpx.Fail(c, apperr.Invalid("member: invalid tierID"))
		return
	}
	var req MembershipTierUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("member: invalid tier update body"))
		return
	}
	view, err := h.svc.UpdateMembershipTier(c.Request.Context(), tierID, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// DisableMembershipTier handles POST /admin/membership-tiers/:tierID/disable.
func (h *Handler) DisableMembershipTier(c *gin.Context) {
	tierID, err := strconv.ParseInt(c.Param("tierID"), 10, 64)
	if err != nil || tierID <= 0 {
		httpx.Fail(c, apperr.Invalid("member: invalid tierID"))
		return
	}
	view, err := h.svc.DisableMembershipTier(c.Request.Context(), tierID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// RechargeProducts handles GET /mini/recharge-products.
func (h *Handler) RechargeProducts(c *gin.Context) {
	views, err := h.svc.ListRechargeProducts(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	notice, err := h.svc.RechargeNotice(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OKWithMeta(c, views, gin.H{"rechargeNotice": notice})
}

// AdminRechargeProducts handles GET /admin/recharge-products.
func (h *Handler) AdminRechargeProducts(c *gin.Context) {
	views, err := h.svc.AdminListRechargeProducts(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// AdminGetRechargeProduct handles GET /admin/recharge-products/:productID.
func (h *Handler) AdminGetRechargeProduct(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("productID"), 10, 64)
	if err != nil || productID <= 0 {
		httpx.Fail(c, apperr.Invalid("member: invalid productID"))
		return
	}
	view, err := h.svc.AdminGetRechargeProduct(c.Request.Context(), productID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// CreateRechargeProduct handles POST /admin/recharge-products.
func (h *Handler) CreateRechargeProduct(c *gin.Context) {
	var req RechargeProductCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("member: invalid recharge product create body"))
		return
	}
	view, err := h.svc.CreateRechargeProduct(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// UpdateRechargeProduct handles PATCH /admin/recharge-products/:productID.
func (h *Handler) UpdateRechargeProduct(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("productID"), 10, 64)
	if err != nil || productID <= 0 {
		httpx.Fail(c, apperr.Invalid("member: invalid productID"))
		return
	}
	var req RechargeProductUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("member: invalid recharge product update body"))
		return
	}
	view, err := h.svc.UpdateRechargeProduct(c.Request.Context(), productID, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// DisableRechargeProduct handles POST /admin/recharge-products/:productID/disable.
func (h *Handler) DisableRechargeProduct(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("productID"), 10, 64)
	if err != nil || productID <= 0 {
		httpx.Fail(c, apperr.Invalid("member: invalid productID"))
		return
	}
	view, err := h.svc.DisableRechargeProduct(c.Request.Context(), productID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// Rankings handles GET /mini/rankings?period=.
func (h *Handler) Rankings(c *gin.Context) {
	views, err := h.svc.ListRankings(c.Request.Context(), c.Query("period"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, views)
}

// UpdateProfile handles PATCH /mini/me.
func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	view, err := h.svc.UpdateProfile(c.Request.Context(), memberID(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// UploadAvatar handles POST /mini/me/avatar. Authentication is enforced by
// the mini router before the image is streamed to object storage.
func (h *Handler) UploadAvatar(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		httpx.Fail(c, apperr.Invalid("file is required"))
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if _, ok := allowedProfileAvatarMimes[contentType]; !ok {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(header.Filename)))
	}
	url, err := h.svc.UploadAvatar(c.Request.Context(), file, header.Size, contentType)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, map[string]string{"avatarUrl": url})
}

var allowedProfileAvatarMimes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// BindPhone handles POST /mini/me/phone-bindings.
func (h *Handler) BindPhone(c *gin.Context) {
	var req BindPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	view, err := h.svc.BindPhone(c.Request.Context(), memberID(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, view)
}

// ListInvitations handles GET /mini/invitations.
func (h *Handler) ListInvitations(c *gin.Context) {
	page := httpx.ParsePage(c)
	views, total, err := h.svc.ListInvitations(c.Request.Context(), memberID(c), page)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.List(c, views, httpx.MetaFor(page, total))
}

// BindInvitation handles POST /mini/invitations/bind.
func (h *Handler) BindInvitation(c *gin.Context) {
	var req BindInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	if err := h.svc.BindInvitation(c.Request.Context(), memberID(c), req); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoData(c)
}

// memberID reads the authenticated member id from the request claims. Callers
// must run behind the auth middleware.
func memberID(c *gin.Context) int64 {
	return authn.MustFromContext(c).SubjectID()
}
