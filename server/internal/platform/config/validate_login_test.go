package config

import (
	"strings"
	"testing"
)

// TestWeChatConfigValidateLogin covers the mini-program login credential
// contract enforced when USE_FAKE_ADAPTERS=false.
func TestWeChatConfigValidateLogin(t *testing.T) {
	tests := []struct {
		name    string
		cfg     WeChatConfig
		wantErr string // substring; empty means no error expected
	}{
		{"ok", WeChatConfig{MiniAppID: "wxid", MiniAppSecret: "secret"}, ""},
		{"missing app id", WeChatConfig{MiniAppSecret: "secret"}, "WECHAT_MINI_APP_ID"},
		{"missing secret", WeChatConfig{MiniAppID: "wxid"}, "WECHAT_MINI_APP_SECRET"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validateLogin()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
