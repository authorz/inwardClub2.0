package asset

import (
	"context"
	"strings"
	"testing"

	"github.com/inwardclub/server/internal/platform/config"
)

func TestQiniuPublicURLDefaultsToHTTPS(t *testing.T) {
	cases := map[string]string{
		"cdn.example.com":         "https://cdn.example.com/pic/a.png",
		"https://cdn.example.com": "https://cdn.example.com/pic/a.png",
		"http://cdn.example.com":  "http://cdn.example.com/pic/a.png",
		"cdn.example.com/":        "https://cdn.example.com/pic/a.png",
	}
	for domain, want := range cases {
		q := NewQiniuObjectStore(config.QiniuConfig{PublicDomain: domain})
		if got := q.PublicURL("pic/a.png"); got != want {
			t.Errorf("PublicURL(domain=%q) = %q, want %q", domain, got, want)
		}
	}
}

func TestQiniuStorageConfigRegionHosts(t *testing.T) {
	// No region: SDK resolves the default zone, so no explicit region is set.
	if got := NewQiniuObjectStore(config.QiniuConfig{}).storageConfig(); got.Region != nil {
		t.Errorf("empty region should leave storage.Config.Region nil, got %+v", got.Region)
	}

	// With a region, upload hosts must match uploadURL's up-${region}/upload-${region}.
	q := NewQiniuObjectStore(config.QiniuConfig{Region: "z2"})
	region := q.storageConfig().Region
	if region == nil {
		t.Fatal("region config should be set when Region is provided")
	}
	if want := []string{"up-z2.qiniup.com"}; !equalHosts(region.SrcUpHosts, want) {
		t.Errorf("SrcUpHosts = %v, want %v", region.SrcUpHosts, want)
	}
	if want := []string{"upload-z2.qiniup.com"}; !equalHosts(region.CdnUpHosts, want) {
		t.Errorf("CdnUpHosts = %v, want %v", region.CdnUpHosts, want)
	}
	if q.uploadURL != "https://up-z2.qiniup.com" {
		t.Errorf("uploadURL = %q, want https://up-z2.qiniup.com", q.uploadURL)
	}
}

func equalHosts(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestFakeUploadPublicObject(t *testing.T) {
	f := NewFakeObjectStore("bucket", "https://cdn.dev.example.com")
	in := PublicUploadInput{
		ObjectKey:   "seed/logo.png",
		Reader:      strings.NewReader("bytes"),
		SizeBytes:   5,
		ContentType: "image/png",
	}
	got, err := f.UploadPublicObject(context.Background(), in)
	if err != nil {
		t.Fatalf("UploadPublicObject: %v", err)
	}
	if got.ObjectKey != in.ObjectKey {
		t.Errorf("ObjectKey = %q, want %q", got.ObjectKey, in.ObjectKey)
	}
	if got.SizeBytes != in.SizeBytes {
		t.Errorf("SizeBytes = %d, want %d", got.SizeBytes, in.SizeBytes)
	}
	if got.PublicURL != "https://cdn.dev.example.com/seed/logo.png" {
		t.Errorf("PublicURL = %q", got.PublicURL)
	}
	if !strings.HasPrefix(got.Hash, "fake:") {
		t.Errorf("Hash = %q, want fake: prefix", got.Hash)
	}
	// Deterministic across runs.
	again, _ := f.UploadPublicObject(context.Background(), in)
	if again.Hash != got.Hash {
		t.Errorf("Hash not deterministic: %q vs %q", again.Hash, got.Hash)
	}
}
