package config

import (
	"strings"
	"testing"
)

// TestOfflineConfigValidate covers the offline acquirer credential contract
// enforced when USE_FAKE_ADAPTERS=false.
func TestOfflineConfigValidate(t *testing.T) {
	full := OfflineConfig{
		Provider:   "generic",
		MerchantID: "M1",
		APIKey:     "key",
		BaseURL:    "https://acq.example.com",
		NotifyURL:  "https://api.example.com/notify",
	}
	if err := full.validate(); err != nil {
		t.Fatalf("full config should validate, got %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*OfflineConfig)
		wantEnv string
	}{
		{"missing provider", func(o *OfflineConfig) { o.Provider = "" }, "OFFLINE_ACQUIRER_PROVIDER"},
		{"missing merchant", func(o *OfflineConfig) { o.MerchantID = "" }, "OFFLINE_ACQUIRER_MERCHANT_ID"},
		{"missing key", func(o *OfflineConfig) { o.APIKey = "" }, "OFFLINE_ACQUIRER_API_KEY"},
		{"missing base url", func(o *OfflineConfig) { o.BaseURL = "" }, "OFFLINE_ACQUIRER_BASE_URL"},
		{"missing notify url", func(o *OfflineConfig) { o.NotifyURL = "" }, "OFFLINE_ACQUIRER_NOTIFY_URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := full
			tt.mutate(&cfg)
			err := cfg.validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantEnv) {
				t.Fatalf("expected error containing %q, got %v", tt.wantEnv, err)
			}
		})
	}
}

// TestXpyunConfigValidate covers the Xpyun printer credential contract.
func TestXpyunConfigValidate(t *testing.T) {
	if err := (XpyunConfig{User: "u", UKey: "k"}).validate(); err != nil {
		t.Fatalf("full config should validate, got %v", err)
	}
	if err := (XpyunConfig{UKey: "k"}).validate(); err == nil || !strings.Contains(err.Error(), "XPYUN_USER") {
		t.Fatalf("expected XPYUN_USER error, got %v", err)
	}
	if err := (XpyunConfig{User: "u"}).validate(); err == nil || !strings.Contains(err.Error(), "XPYUN_UKEY") {
		t.Fatalf("expected XPYUN_UKEY error, got %v", err)
	}
}
