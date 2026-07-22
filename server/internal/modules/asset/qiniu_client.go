package asset

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/downloader"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"

	"github.com/inwardclub/server/internal/platform/config"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// QiniuObjectStore is the single wrapper around the Qiniu v7 SDK. It is used
// only when USE_FAKE_ADAPTERS=false; tests always use FakeObjectStore.
type QiniuObjectStore struct {
	cfg       config.QiniuConfig
	mac       *auth.Credentials
	v2creds   *credentials.Credentials
	uploadURL string
}

// NewQiniuObjectStore builds the real object store from config.
func NewQiniuObjectStore(cfg config.QiniuConfig) *QiniuObjectStore {
	uploadURL := "https://up.qiniup.com"
	if cfg.Region != "" {
		uploadURL = fmt.Sprintf("https://up-%s.qiniup.com", cfg.Region)
	}
	return &QiniuObjectStore{
		cfg:       cfg,
		mac:       auth.New(cfg.AccessKey, cfg.SecretKey),
		v2creds:   credentials.NewCredentials(cfg.AccessKey, cfg.SecretKey),
		uploadURL: uploadURL,
	}
}

// storageConfig builds a storage.Config pinned to the configured region so
// server-side uploads target the same hosts as uploadURL. Without an explicit
// region the SDK would query the default zone and could pick the wrong one.
func (q *QiniuObjectStore) storageConfig() *storage.Config {
	cfg := &storage.Config{}
	if q.cfg.Region != "" {
		cfg.Region = &storage.Region{
			SrcUpHosts: []string{fmt.Sprintf("up-%s.qiniup.com", q.cfg.Region)},
			CdnUpHosts: []string{fmt.Sprintf("upload-%s.qiniup.com", q.cfg.Region)},
		}
	}
	return cfg
}

// Bucket returns the configured bucket.
func (q *QiniuObjectStore) Bucket() string { return q.cfg.Bucket }

// CreateUploadCredential signs a scoped, size-limited, callback-bound put token.
func (q *QiniuObjectStore) CreateUploadCredential(ctx context.Context, input CreateCredentialInput) (UploadCredential, error) {
	ttl := q.cfg.TokenTTL
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	expiry := time.Now().Add(ttl)

	policy, err := uptoken.NewPutPolicyWithKey(q.cfg.Bucket, input.ObjectKey, expiry)
	if err != nil {
		return UploadCredential{}, apperr.Internal(err)
	}
	// Lock the token to this exact object and size, and bind the callback so the
	// server records the asset only after Qiniu confirms the upload.
	callbackBody := `{"assetId":$(x:assetId),"key":"$(key)","etag":"$(etag)","fsize":$(fsize),"mimeType":"$(mimeType)","bucket":"$(bucket)"}`
	policy = policy.
		SetInsertOnly(1).
		SetFsizeLimit(input.MaxSizeBytes).
		SetReturnBody(callbackBody).
		SetCallbackUrl(q.cfg.UploadCallbackURL).
		SetCallbackBody(callbackBody).
		SetCallbackBodyType("application/json")

	token, err := uptoken.NewSigner(policy, q.v2creds).GetUpToken(ctx)
	if err != nil {
		return UploadCredential{}, apperr.Internal(err)
	}
	return UploadCredential{
		AssetID:      input.AssetID,
		ObjectKey:    input.ObjectKey,
		UploadToken:  token,
		UploadURL:    q.uploadURL,
		ExpiresAt:    expiry.UTC(),
		MaxSizeBytes: input.MaxSizeBytes,
	}, nil
}

// UploadPublicObject uploads a reader to a fixed bucket:key scope using the
// form uploader. The bucket:key scope allows overwriting, so re-running with the
// same object key is idempotent.
func (q *QiniuObjectStore) UploadPublicObject(ctx context.Context, input PublicUploadInput) (UploadedObject, error) {
	policy, err := uptoken.NewPutPolicyWithKey(q.cfg.Bucket, input.ObjectKey, time.Now().Add(10*time.Minute))
	if err != nil {
		return UploadedObject{}, apperr.Internal(err)
	}
	token, err := uptoken.NewSigner(policy, q.v2creds).GetUpToken(ctx)
	if err != nil {
		return UploadedObject{}, apperr.Internal(err)
	}

	uploader := storage.NewFormUploader(q.storageConfig())
	var ret storage.PutRet
	extra := &storage.PutExtra{MimeType: input.ContentType}
	if err := uploader.Put(ctx, &ret, token, input.ObjectKey, input.Reader, input.SizeBytes, extra); err != nil {
		return UploadedObject{}, apperr.Internal(err)
	}
	return UploadedObject{
		ObjectKey: input.ObjectKey,
		Hash:      ret.Hash,
		SizeBytes: input.SizeBytes,
		PublicURL: q.PublicURL(input.ObjectKey),
	}, nil
}

// VerifyUploadCallback validates the Qiniu callback signature then parses it.
func (q *QiniuObjectStore) VerifyUploadCallback(req *http.Request) (CallbackPayload, error) {
	ok, err := q.mac.VerifyCallback(req)
	if err != nil {
		return CallbackPayload{}, apperr.Internal(err)
	}
	if !ok {
		return CallbackPayload{}, apperr.Unauthenticated("invalid qiniu callback signature")
	}
	var payload CallbackPayload
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		return CallbackPayload{}, apperr.Invalid("invalid callback body")
	}
	return payload, nil
}

// PublicURL joins the public CDN domain with the object key. If the configured
// public domain carries no scheme, https:// is assumed.
func (q *QiniuObjectStore) PublicURL(objectKey string) string {
	domain := strings.TrimRight(q.cfg.PublicDomain, "/")
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}
	return domain + "/" + objectKey
}

// PrivateURL returns a short-lived signed download URL for private assets.
func (q *QiniuObjectStore) PrivateURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	base := strings.TrimRight(q.cfg.PrivateDomain, "/") + "/" + objectKey
	u, err := url.Parse(base)
	if err != nil {
		return "", apperr.Internal(err)
	}
	signer := downloader.NewCredentialsSigner(q.v2creds)
	if err := signer.Sign(ctx, u, &downloader.SignOptions{TTL: ttl}); err != nil {
		return "", apperr.Internal(err)
	}
	return u.String(), nil
}

// Delete removes an object from the bucket.
func (q *QiniuObjectStore) Delete(_ context.Context, objectKey string) error {
	mgr := storage.NewBucketManager(q.mac, &storage.Config{})
	if err := mgr.Delete(q.cfg.Bucket, objectKey); err != nil {
		return apperr.Internal(err)
	}
	return nil
}
