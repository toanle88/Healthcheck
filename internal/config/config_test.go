package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	// 1. Test Default Values
	// Clear environment variables first to ensure we get defaults
	os.Unsetenv("PORT")
	os.Unsetenv("DATABASE_URL")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Expected default PORT 8080, got %s", cfg.Port)
	}

	// 2. Test Environment Overrides
	os.Setenv("PORT", "9999")
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")

	cfg = Load()

	if cfg.Port != "9999" {
		t.Errorf("Expected PORT 9999 from env, got %s", cfg.Port)
	}

	if cfg.DatabaseURL != "postgres://test:test@localhost:5432/test" {
		t.Errorf("Expected DATABASE_URL override, got %s", cfg.DatabaseURL)
	}

	// Clean up for other tests
	os.Unsetenv("PORT")
	os.Unsetenv("DATABASE_URL")
}

func TestLoadMore(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Setenv("DB_HOST", "some-azure-host.postgres.database.azure.com")
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://domain1.com, http://domain2.com ")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("CORS_ALLOWED_ORIGINS")
	}()

	cfg := Load()

	// Should contain sslmode=require since host is not localhost
	if !strings.Contains(cfg.DatabaseURL, "sslmode=require") {
		t.Errorf("expected DatabaseURL to contain sslmode=require for non-localhost, got: %s", cfg.DatabaseURL)
	}

	// Should split and trim CORS Allowed Origins
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Errorf("expected 2 CORS allowed origins, got %d", len(cfg.CORSAllowedOrigins))
	} else {
		if cfg.CORSAllowedOrigins[0] != "http://domain1.com" || cfg.CORSAllowedOrigins[1] != "http://domain2.com" {
			t.Errorf("unexpected CORS allowed origins content: %v", cfg.CORSAllowedOrigins)
		}
	}
}
