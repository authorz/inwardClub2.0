package asset

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// FakeObjectStore is the in-process ObjectStore used in development and tests so
// nothing touches the network. Callback signatures use HMAC-SHA256 over the body
// so forged/replayed callback tests behave like the real verifier.
type FakeObjectStore struct {
	bucket       string
	publicDomain string
	secret       []byte
}

// NewFakeObjectStore builds a fake store with a fixed callback secret.
func NewFakeObjectStore(bucket, publicDomain string) *FakeObjectStore {
	if bucket == "" {
		bucket = "inwardclub-dev"
	}
	if publicDomain == "" {
		publicDomain = "https://cdn.dev.example.com"
	}
	return &FakeObjectStore{
		bucket:       bucket,
		publicDomain: publicDomain,
		secret:       []byte("fake-callback-secret"),
	}
}

// Bucket returns the configured bucket name.
func (f *FakeObjectStore) Bucket() string { return f.bucket }

// CreateUploadCredential returns a deterministic dev token; no network call.
func (f *FakeObjectStore) CreateUploadCredential(_ context.Context, input CreateCredentialInput) (UploadCredential, error) {
	now := time.Now().UTC()
	return UploadCredential{
		AssetID:      input.AssetID,
		ObjectKey:    input.ObjectKey,
		UploadToken:  "fake-token:" + input.ObjectKey,
		UploadURL:    "https://up.fake.qiniup.com",
		ExpiresAt:    now.Add(10 * time.Minute),
		MaxSizeBytes: input.MaxSizeBytes,
	}, nil
}

// UploadPublicObject returns a deterministic result without any network call.
// The hash is derived from the object key so results are stable across runs.
func (f *FakeObjectStore) UploadPublicObject(_ context.Context, input PublicUploadInput) (UploadedObject, error) {
	sum := sha256.Sum256([]byte(input.ObjectKey))
	return UploadedObject{
		ObjectKey: input.ObjectKey,
		Hash:      "fake:" + hex.EncodeToString(sum[:8]),
		SizeBytes: input.SizeBytes,
		PublicURL: f.PublicURL(input.ObjectKey),
	}, nil
}

// Sign is a test helper that produces the signature header value for a body.
func (f *FakeObjectStore) Sign(body []byte) string {
	mac := hmac.New(sha256.New, f.secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyUploadCallback validates the fake signature header and parses the body.
func (f *FakeObjectStore) VerifyUploadCallback(req *http.Request) (CallbackPayload, error) {
	body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
	if err != nil {
		return CallbackPayload{}, apperr.Invalid("cannot read callback body")
	}
	expected := f.Sign(body)
	if !hmac.Equal([]byte(req.Header.Get("X-Fake-Signature")), []byte(expected)) {
		return CallbackPayload{}, apperr.Unauthenticated("invalid callback signature")
	}
	var payload CallbackPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return CallbackPayload{}, apperr.Invalid("invalid callback body")
	}
	return payload, nil
}

// PublicURL joins the public domain with the object key. If the configured
// public domain carries no scheme, https:// is assumed (mirrors the real store).
func (f *FakeObjectStore) PublicURL(objectKey string) string {
	domain := strings.TrimRight(f.publicDomain, "/")
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}
	return domain + "/" + objectKey
}

// PrivateURL appends a fake short-lived token.
func (f *FakeObjectStore) PrivateURL(_ context.Context, objectKey string, ttl time.Duration) (string, error) {
	exp := time.Now().Add(ttl).Unix()
	return fmt.Sprintf("%s/%s?e=%d&token=fake", f.publicDomain, objectKey, exp), nil
}

// Delete is a no-op for the fake store.
func (f *FakeObjectStore) Delete(_ context.Context, _ string) error { return nil }
