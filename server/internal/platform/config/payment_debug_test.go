package config

import "testing"

func TestWeChatPayAmountOverrideCent(t *testing.T) {
	t.Run("development debug mode charges one cent", func(t *testing.T) {
		cfg := Config{AppEnv: "development", PaymentDebugMode: true}
		if got := cfg.WeChatPayAmountOverrideCent(); got != 1 {
			t.Fatalf("override amount = %d, want 1", got)
		}
	})

	t.Run("disabled mode keeps the original amount", func(t *testing.T) {
		cfg := Config{AppEnv: "development"}
		if got := cfg.WeChatPayAmountOverrideCent(); got != 0 {
			t.Fatalf("override amount = %d, want 0", got)
		}
	})

	t.Run("production ignores the debug flag", func(t *testing.T) {
		cfg := Config{AppEnv: "production", PaymentDebugMode: true}
		if got := cfg.WeChatPayAmountOverrideCent(); got != 0 {
			t.Fatalf("override amount = %d, want 0", got)
		}
	})
}
