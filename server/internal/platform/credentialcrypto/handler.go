package credentialcrypto

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/httpx"
)

// Handler exposes the current public key to authenticated consoles.
type Handler struct{ cipher *Cipher }

func NewHandler(cipher *Cipher) *Handler { return &Handler{cipher: cipher} }

func (h *Handler) PublicKey(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	httpx.OK(c, h.cipher.PublicKey())
}
