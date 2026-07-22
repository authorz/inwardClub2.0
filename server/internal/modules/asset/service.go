package asset

import (
	"context"
	"net/http"
	"time"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// Caller identifies who is requesting an upload credential.
type Caller struct {
	SubjectType authn.SubjectType
	SubjectID   int64
}

// backOfficeSubjects may upload operational imagery (logos, banners, products…).
var backOfficeSubjects = map[authn.SubjectType]bool{
	authn.SubjectSuperAdmin: true,
	authn.SubjectStoreAdmin: true,
	authn.SubjectCashier:    true,
}

// Service orchestrates upload credential issuance and callback handling.
type Service struct {
	repo  Repository
	store ObjectStore
	env   string
}

// NewService builds the asset service.
func NewService(repo Repository, store ObjectStore, env string) *Service {
	return &Service{repo: repo, store: store, env: env}
}

// CreateUploadCredential validates the request and caller, records a pending
// asset, generates its object key and returns a scoped upload token.
func (s *Service) CreateUploadCredential(ctx context.Context, caller Caller, req UploadCredentialRequest) (UploadCredential, error) {
	purpose := UploadPurpose(req.Purpose)
	policy, ok := policies[purpose]
	if !ok {
		return UploadCredential{}, apperr.Invalid("unsupported upload purpose")
	}
	ext, ok := policy.mimes[req.ContentType]
	if !ok {
		return UploadCredential{}, apperr.Invalid("content type not allowed for purpose")
	}
	if req.SizeBytes <= 0 || req.SizeBytes > policy.maxSize {
		return UploadCredential{}, apperr.Invalid("file size out of allowed range")
	}
	if !s.callerAllowed(caller, purpose) {
		return UploadCredential{}, apperr.Forbidden("not permitted to upload this purpose")
	}
	visibility := req.Visibility
	if visibility == "" {
		visibility = VisibilityPublic
	}

	// Record a pending asset first, then derive the object key from its id.
	id, err := s.repo.CreatePending(ctx, Asset{
		Bucket:           s.store.Bucket(),
		ObjectKey:        "pending/" + mustRandom(),
		OriginalFilename: req.Filename,
		ContentType:      req.ContentType,
		SizeBytes:        req.SizeBytes,
		Purpose:          string(purpose),
		Visibility:       visibility,
		UploadedByType:   string(caller.SubjectType),
		UploadedByID:     caller.SubjectID,
	})
	if err != nil {
		return UploadCredential{}, err
	}
	objectKey, err := buildObjectKey(s.env, purpose, id, ext)
	if err != nil {
		return UploadCredential{}, apperr.Internal(err)
	}
	if err := s.repo.SetObjectKey(ctx, id, objectKey); err != nil {
		return UploadCredential{}, err
	}
	return s.store.CreateUploadCredential(ctx, CreateCredentialInput{
		AssetID:      id,
		ObjectKey:    objectKey,
		MaxSizeBytes: policy.maxSize,
		Visibility:   visibility,
	})
}

// HandleCallback verifies the callback signature, matches the pending asset and
// marks it uploaded. Repeated callbacks return the same success (idempotent).
func (s *Service) HandleCallback(ctx context.Context, req *http.Request) (CallbackPayload, error) {
	payload, err := s.store.VerifyUploadCallback(req)
	if err != nil {
		return CallbackPayload{}, err
	}
	asset, err := s.repo.GetByID(ctx, payload.AssetID)
	if err != nil {
		return CallbackPayload{}, err
	}
	if asset.ObjectKey != payload.Key || asset.Bucket != payload.Bucket {
		_ = s.repo.MarkFailed(ctx, asset.ID)
		return CallbackPayload{}, apperr.Invalid("callback key/bucket mismatch")
	}
	if asset.Status == StatusUploaded || asset.Status == StatusBound {
		return payload, nil // idempotent replay
	}
	if asset.Status != StatusPending {
		return CallbackPayload{}, apperr.Conflict("asset not in pending state")
	}
	if err := s.repo.MarkUploaded(ctx, asset.ID, payload.Etag, payload.Fsize, time.Now().UTC()); err != nil {
		return CallbackPayload{}, err
	}
	return payload, nil
}

// PublicURLByID resolves an asset id to its public URL. A zero id returns "".
// Used by other modules that store only asset_id and need a display URL.
func (s *Service) PublicURLByID(ctx context.Context, id int64) (string, error) {
	if id == 0 {
		return "", nil
	}
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	return s.store.PublicURL(a.ObjectKey), nil
}

// PublicURL resolves a stored object key/path to its public URL (public CDN
// domain + key). An empty key returns "". Used by modules that store a raw
// object path and need a display URL.
func (s *Service) PublicURL(objectKey string) string {
	if objectKey == "" {
		return ""
	}
	return s.store.PublicURL(objectKey)
}

// ViewFor builds a public AssetView for an uploaded asset.
func (s *Service) ViewFor(a Asset) AssetView {
	return AssetView{
		AssetID: a.ID,
		URL:     s.store.PublicURL(a.ObjectKey),
		Width:   a.Width,
		Height:  a.Height,
		Status:  a.Status,
	}
}

func (s *Service) callerAllowed(caller Caller, purpose UploadPurpose) bool {
	if purpose == UploadPurposeAvatar {
		return caller.SubjectType == authn.SubjectMember || caller.SubjectType == authn.SubjectStaff
	}
	return backOfficeSubjects[caller.SubjectType]
}

func mustRandom() string {
	r, err := randomHex(16)
	if err != nil {
		// randomHex only fails if the OS RNG fails, which is fatal anyway.
		panic(err)
	}
	return r
}
