package asset

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// Handler exposes asset HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds the asset handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// UploadCredentials issues a scoped upload token for the authenticated caller.
// It is mounted under each console group; the caller identity comes from claims.
func (h *Handler) UploadCredentials(c *gin.Context) {
	claims := authn.MustFromContext(c)
	var req UploadCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, apperr.Invalid("invalid request body"))
		return
	}
	cred, err := h.svc.CreateUploadCredential(c.Request.Context(), Caller{
		SubjectType: claims.SubjectType,
		SubjectID:   claims.SubjectID(),
	}, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, cred)
}

// Callback handles the Qiniu (or fake) upload callback. No JWT; the signature is
// verified inside the object store implementation.
func (h *Handler) Callback(c *gin.Context) {
	payload, err := h.svc.HandleCallback(c.Request.Context(), c.Request)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(200, gin.H{"assetId": payload.AssetID, "status": StatusUploaded})
}
