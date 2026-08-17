package config

import (
	"strings"
	"testing"
)

func TestValidateRequiresCredentialEncryptionKeyInProduction(t *testing.T) {
	cfg := defaults()
	cfg.AppEnv = "production"
	cfg.MySQLDSN = "test"
	cfg.JWT.SigningKey = "test"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "CREDENTIAL_ENCRYPTION_PRIVATE_KEY_PATH") {
		t.Fatalf("expected credential encryption key requirement, got %v", err)
	}
}
