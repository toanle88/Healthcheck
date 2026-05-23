package main

import (
	"context"
	"os"
	"testing"
)

func TestRunMigrate(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL_TEST not set")
	}

	ctx := context.Background()

	// 1. Success case
	err := runMigrate(ctx, dbURL)
	if err != nil {
		t.Errorf("expected no error from runMigrate, got: %v", err)
	}

	// 2. Failure case (bad connection string)
	err = runMigrate(ctx, "postgres://invalid:invalid@localhost:5432/invalid?sslmode=disable")
	if err == nil {
		t.Errorf("expected error from runMigrate with invalid DB URL, got nil")
	}
}
