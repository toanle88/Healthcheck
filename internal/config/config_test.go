package config

import (
	"os"
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
