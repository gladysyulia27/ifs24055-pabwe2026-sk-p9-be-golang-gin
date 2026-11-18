package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigDefaultsAndEnv(t *testing.T) {
	// Clear relevant env vars and restore after test
	keys := []string{
		"APP_PORT", "APP_ENV", "APP_KEY", "ADMIN_USERNAME", "ADMIN_PASSWORD",
		"DB_CONNECTION", "DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME",
		"DB_PASSWORD", "DB_SSLMODE", "OPENAI_ENDPOINT", "OPENAI_KEY",
		"OPENAI_MODEL", "SEARCH_ENGINE_ENDPOINT",
	}
	saved := make(map[string]string)
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		_ = os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				_ = os.Setenv(k, v)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}()

	// 1. Default values (pointing to non-existent .env so it doesn't load local .env)
	cfg := LoadConfig("non_existent.env")
	if cfg.AppPort != "8081" {
		t.Fatalf("expected default port 8081, got %s", cfg.AppPort)
	}
	if cfg.AppEnv != "development" {
		t.Fatalf("expected default env development, got %s", cfg.AppEnv)
	}

	// 2. Load from custom .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	envContent := "APP_PORT=9090\nDB_HOST=localhost\nDB_DATABASE=testdb\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg2 := LoadConfig(envFile)
	if cfg2.AppPort != "9090" || cfg2.DBHost != "localhost" || cfg2.DBDatabase != "testdb" {
		t.Fatalf("unexpected config from file: %+v", cfg2)
	}

	// 3. LoadConfig without args
	_ = LoadConfig()

	// 4. DSN format check
	dsn := cfg2.DSN()
	if !strings.Contains(dsn, "host=localhost") || !strings.Contains(dsn, "dbname=testdb") {
		t.Fatalf("unexpected DSN: %s", dsn)
	}
}
