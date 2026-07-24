package main

import (
	"os"
	"testing"
)

func TestDotEnvProvidesMySQLDSN(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".env", []byte("MYSQL_DSN=test-user:test-pass@tcp(127.0.0.1:3306)/test-db?parseTime=true&loc=UTC\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	oldValue, wasSet := os.LookupEnv("MYSQL_DSN")
	if err := os.Unsetenv("MYSQL_DSN"); err != nil {
		t.Fatalf("unset MYSQL_DSN: %v", err)
	}
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv("MYSQL_DSN", oldValue)
		} else {
			_ = os.Unsetenv("MYSQL_DSN")
		}
	})

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MySQLDSN == "" {
		t.Fatal("expected MYSQL_DSN to be loaded into config from .env")
	}
}
