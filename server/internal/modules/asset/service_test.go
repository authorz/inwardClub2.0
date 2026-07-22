package asset

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// memRepo is an in-memory Repository for service tests.
type memRepo struct {
	seq    int64
	assets map[int64]*Asset
}

func newMemRepo() *memRepo { return &memRepo{assets: map[int64]*Asset{}} }

func (r *memRepo) CreatePending(_ context.Context, a Asset) (int64, error) {
	r.seq++
	a.ID = r.seq
	a.Status = StatusPending
	cp := a
	r.assets[a.ID] = &cp
	return a.ID, nil
}

func (r *memRepo) SetObjectKey(_ context.Context, id int64, key string) error {
	r.assets[id].ObjectKey = key
	return nil
}

func (r *memRepo) GetByID(_ context.Context, id int64) (Asset, error) {
	a, ok := r.assets[id]
	if !ok {
		return Asset{}, apperr.NotFound("asset not found")
	}
	return *a, nil
}

func (r *memRepo) MarkUploaded(_ context.Context, id int64, etag string, size int64, at time.Time) error {
	a := r.assets[id]
	a.Status = StatusUploaded
	a.Etag = etag
	a.SizeBytes = size
	a.UploadedAt = &at
	return nil
}

func (r *memRepo) MarkFailed(_ context.Context, id int64) error {
	r.assets[id].Status = StatusFailed
	return nil
}

func (r *memRepo) ExpirePending(_ context.Context, createdBefore time.Time) (int64, error) {
	var n int64
	for _, a := range r.assets {
		if a.Status == StatusPending && a.CreatedAt.Before(createdBefore) {
			a.Status = StatusFailed
			n++
		}
	}
	return n, nil
}

func newService() (*Service, *memRepo, *FakeObjectStore) {
	repo := newMemRepo()
	store := NewFakeObjectStore("bucket-test", "https://cdn.test")
	return NewService(repo, store, "test"), repo, store
}

func TestPublicURLByIDComposesDomainAndKey(t *testing.T) {
	svc, repo, _ := newService()
	// An asset persists only its object key (the stored reference), never a full
	// URL. Callers such as the VIP banner store just an asset id.
	id, err := repo.CreatePending(context.Background(), Asset{ObjectKey: "inwardclub/test/vip_banner/42.png"})
	if err != nil {
		t.Fatalf("seed asset: %v", err)
	}
	url, err := svc.PublicURLByID(context.Background(), id)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// The public URL is derived from the configured public domain
	// (QINIU_PUBLIC_DOMAIN) joined with the stored object key.
	if want := "https://cdn.test/inwardclub/test/vip_banner/42.png"; url != want {
		t.Fatalf("PublicURLByID = %q, want %q", url, want)
	}
	// A zero id resolves to an empty string (e.g. no banner asset set).
	if got, _ := svc.PublicURLByID(context.Background(), 0); got != "" {
		t.Fatalf("PublicURLByID(0) = %q, want empty", got)
	}
}

func TestCreateUploadCredential_ObjectKeyFormat(t *testing.T) {
	svc, _, _ := newService()
	cred, err := svc.CreateUploadCredential(context.Background(),
		Caller{SubjectType: authn.SubjectSuperAdmin, SubjectID: 1},
		UploadCredentialRequest{Purpose: "product", Filename: "a.jpg", ContentType: "image/jpeg", SizeBytes: 1024})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(cred.ObjectKey, "inwardclub/test/product/") {
		t.Fatalf("unexpected object key: %s", cred.ObjectKey)
	}
	if !strings.HasSuffix(cred.ObjectKey, ".jpg") {
		t.Fatalf("object key should end with real extension: %s", cred.ObjectKey)
	}
}

func TestCreateUploadCredential_RejectsBadMimeAndSize(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	caller := Caller{SubjectType: authn.SubjectSuperAdmin, SubjectID: 1}

	if _, err := svc.CreateUploadCredential(ctx, caller,
		UploadCredentialRequest{Purpose: "product", Filename: "a.gif", ContentType: "image/gif", SizeBytes: 1024}); err == nil {
		t.Fatal("expected disallowed mime to be rejected")
	}
	if _, err := svc.CreateUploadCredential(ctx, caller,
		UploadCredentialRequest{Purpose: "product", Filename: "a.jpg", ContentType: "image/jpeg", SizeBytes: 999 * 1024 * 1024}); err == nil {
		t.Fatal("expected oversize file to be rejected")
	}
}

func TestCreateUploadCredential_PurposePermission(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()

	// Member may upload an avatar but not a product image.
	if _, err := svc.CreateUploadCredential(ctx, Caller{SubjectType: authn.SubjectMember, SubjectID: 5},
		UploadCredentialRequest{Purpose: "avatar", Filename: "me.png", ContentType: "image/png", SizeBytes: 2048}); err != nil {
		t.Fatalf("member avatar should be allowed: %v", err)
	}
	if _, err := svc.CreateUploadCredential(ctx, Caller{SubjectType: authn.SubjectMember, SubjectID: 5},
		UploadCredentialRequest{Purpose: "product", Filename: "p.png", ContentType: "image/png", SizeBytes: 2048}); err == nil {
		t.Fatal("member must not upload product images")
	}
}

func TestHandleCallback_HappyPathIdempotentAndForged(t *testing.T) {
	svc, _, store := newService()
	ctx := context.Background()
	cred, err := svc.CreateUploadCredential(ctx, Caller{SubjectType: authn.SubjectSuperAdmin, SubjectID: 1},
		UploadCredentialRequest{Purpose: "product", Filename: "a.jpg", ContentType: "image/jpeg", SizeBytes: 1024})
	if err != nil {
		t.Fatalf("credential: %v", err)
	}

	body, _ := json.Marshal(CallbackPayload{
		AssetID: cred.AssetID, Key: cred.ObjectKey, Etag: "etag123", Fsize: 2048,
		MimeType: "image/jpeg", Bucket: "bucket-test",
	})

	// Forged (missing signature) is rejected.
	forged := httptest.NewRequest("POST", "/cb", bytes.NewReader(body))
	if _, err := svc.HandleCallback(ctx, forged); err == nil {
		t.Fatal("expected forged callback to be rejected")
	}

	// Signed callback succeeds and marks uploaded.
	req := httptest.NewRequest("POST", "/cb", bytes.NewReader(body))
	req.Header.Set("X-Fake-Signature", store.Sign(body))
	if _, err := svc.HandleCallback(ctx, req); err != nil {
		t.Fatalf("signed callback: %v", err)
	}
	got, _ := svc.repo.GetByID(ctx, cred.AssetID)
	if got.Status != StatusUploaded {
		t.Fatalf("expected uploaded, got %s", got.Status)
	}

	// Replay is idempotent (no error, still uploaded).
	replay := httptest.NewRequest("POST", "/cb", bytes.NewReader(body))
	replay.Header.Set("X-Fake-Signature", store.Sign(body))
	if _, err := svc.HandleCallback(ctx, replay); err != nil {
		t.Fatalf("replay should be idempotent: %v", err)
	}
}

func TestHandleCallback_KeyMismatchFails(t *testing.T) {
	svc, _, store := newService()
	ctx := context.Background()
	cred, _ := svc.CreateUploadCredential(ctx, Caller{SubjectType: authn.SubjectSuperAdmin, SubjectID: 1},
		UploadCredentialRequest{Purpose: "product", Filename: "a.jpg", ContentType: "image/jpeg", SizeBytes: 1024})

	body, _ := json.Marshal(CallbackPayload{
		AssetID: cred.AssetID, Key: "inwardclub/test/product/9999/99/9-deadbeef.jpg",
		Bucket: "bucket-test",
	})
	req := httptest.NewRequest("POST", "/cb", bytes.NewReader(body))
	req.Header.Set("X-Fake-Signature", store.Sign(body))
	if _, err := svc.HandleCallback(ctx, req); err == nil {
		t.Fatal("expected key mismatch to fail")
	}
}
