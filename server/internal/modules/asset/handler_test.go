package asset

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// newTestEngine wires the shared asset Handler exactly like the real consoles do:
// one upload-credentials route whose caller identity comes from injected claims,
// plus the JWT-less callback route. mw stands in for the authn middleware.
func newTestEngine(h *Handler, claims *authn.Claims) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	inject := func(c *gin.Context) { c.Set(httpx.CtxClaims, claims) }
	r.POST("/assets/upload-credentials", inject, h.UploadCredentials)
	r.POST("/qiniu/upload-callback", h.Callback)
	return r
}

func claimsFor(st authn.SubjectType, id string) *authn.Claims {
	c := &authn.Claims{SubjectType: st}
	c.Subject = id
	return c
}

// TestHandler_UploadCredentials_ReusableAcrossConsoles proves the single shared
// handler issues credentials for a mini caller (member avatar), an admin caller
// (super_admin product) and a store caller (store_admin banner) — the three
// mount points in the router — without any per-console handler code.
func TestHandler_UploadCredentials_ReusableAcrossConsoles(t *testing.T) {
	cases := []struct {
		name    string
		subject authn.SubjectType
		req     UploadCredentialRequest
		prefix  string
	}{
		{"mini-member-avatar", authn.SubjectMember,
			UploadCredentialRequest{Purpose: "avatar", Filename: "me.png", ContentType: "image/png", SizeBytes: 2048},
			"inwardclub/test/avatar/"},
		{"admin-superadmin-product", authn.SubjectSuperAdmin,
			UploadCredentialRequest{Purpose: "product", Filename: "p.jpg", ContentType: "image/jpeg", SizeBytes: 4096},
			"inwardclub/test/product/"},
		{"store-storeadmin-banner", authn.SubjectStoreAdmin,
			UploadCredentialRequest{Purpose: "banner", Filename: "b.webp", ContentType: "image/webp", SizeBytes: 8192},
			"inwardclub/test/banner/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newService()
			r := newTestEngine(NewHandler(svc), claimsFor(tc.subject, "1"))

			body, _ := json.Marshal(tc.req)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/assets/upload-credentials", bytes.NewReader(body)))

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
			var env struct {
				Data UploadCredential `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if env.Data.AssetID == 0 {
				t.Errorf("expected an asset id, got 0")
			}
			if got := env.Data.ObjectKey; len(got) < len(tc.prefix) || got[:len(tc.prefix)] != tc.prefix {
				t.Errorf("objectKey = %q, want prefix %q", got, tc.prefix)
			}
			if env.Data.UploadToken == "" || env.Data.UploadURL == "" {
				t.Errorf("credential missing token/url: %+v", env.Data)
			}
		})
	}
}

// TestHandler_UploadCredentials_ForbiddenPurpose confirms the handler surfaces
// the service's permission error as an error envelope (member cannot upload a
// product image) rather than 200.
func TestHandler_UploadCredentials_ForbiddenPurpose(t *testing.T) {
	svc, _, _ := newService()
	r := newTestEngine(NewHandler(svc), claimsFor(authn.SubjectMember, "5"))

	body, _ := json.Marshal(UploadCredentialRequest{
		Purpose: "product", Filename: "p.png", ContentType: "image/png", SizeBytes: 2048,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/assets/upload-credentials", bytes.NewReader(body)))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", w.Code, w.Body.String())
	}
}

// TestHandler_Callback_HappyPathAndForged drives the full credential→callback
// round trip through HTTP: a signed callback marks the asset uploaded, while an
// unsigned (forged) callback is rejected. No network is touched (fake store).
func TestHandler_Callback_HappyPathAndForged(t *testing.T) {
	svc, _, store := newService()
	r := newTestEngine(NewHandler(svc), claimsFor(authn.SubjectSuperAdmin, "1"))

	// 1. Issue a credential over HTTP.
	credBody, _ := json.Marshal(UploadCredentialRequest{
		Purpose: "product", Filename: "a.jpg", ContentType: "image/jpeg", SizeBytes: 1024,
	})
	cw := httptest.NewRecorder()
	r.ServeHTTP(cw, httptest.NewRequest(http.MethodPost, "/assets/upload-credentials", bytes.NewReader(credBody)))
	if cw.Code != http.StatusOK {
		t.Fatalf("credential status = %d, body = %s", cw.Code, cw.Body.String())
	}
	var env struct {
		Data UploadCredential `json:"data"`
	}
	if err := json.Unmarshal(cw.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode credential: %v", err)
	}

	cbBody, _ := json.Marshal(CallbackPayload{
		AssetID: env.Data.AssetID, Key: env.Data.ObjectKey, Etag: "etag123",
		Fsize: 2048, MimeType: "image/jpeg", Bucket: "bucket-test",
	})

	// 2. Forged callback (no signature) is rejected.
	fw := httptest.NewRecorder()
	r.ServeHTTP(fw, httptest.NewRequest(http.MethodPost, "/qiniu/upload-callback", bytes.NewReader(cbBody)))
	if fw.Code != http.StatusUnauthorized {
		t.Fatalf("forged callback status = %d, want 401; body = %s", fw.Code, fw.Body.String())
	}

	// 3. Signed callback succeeds and reports the asset uploaded.
	sw := httptest.NewRecorder()
	signed := httptest.NewRequest(http.MethodPost, "/qiniu/upload-callback", bytes.NewReader(cbBody))
	signed.Header.Set("X-Fake-Signature", store.Sign(cbBody))
	r.ServeHTTP(sw, signed)
	if sw.Code != http.StatusOK {
		t.Fatalf("signed callback status = %d, body = %s", sw.Code, sw.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(sw.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode callback response: %v", err)
	}
	if got["status"] != StatusUploaded {
		t.Errorf("callback status = %v, want %q", got["status"], StatusUploaded)
	}
}
