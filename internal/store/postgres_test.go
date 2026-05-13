package store

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestStoreIntegration(t *testing.T) {
	// 1. Get test DB URL from environment
	// Example: DATABASE_URL_TEST="postgres://postgres:postgres@localhost:5432/healthcheck?sslmode=disable"
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL_TEST not set")
	}

	ctx := context.Background()

	// 2. Connect to the test DB
	st, err := New(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test DB: %v", err)
	}
	defer st.Close()

	// 3. Clear existing data for a clean test
	_, _ = st.DB.Exec(ctx, "DELETE FROM checks")

	// 4. Test InitSchema
	if err := st.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// 5. Test InsertCheck
	target := "test-integration.com"
	err = st.InsertCheck(ctx, target, "up", 123)
	if err != nil {
		t.Errorf("InsertCheck failed: %v", err)
	}

	// 6. Test GetLatestChecks
	checks, err := st.GetLatestChecks(ctx)
	if err != nil {
		t.Errorf("GetLatestChecks failed: %v", err)
	}

	if len(checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(checks))
	} else if checks[0].Target != target {
		t.Errorf("Expected target %s, got %s", target, checks[0].Target)
	}

	// 7. Test CleanupOldChecks
	// Insert an old record (this is tricky without changing DB time, so we just test the logic doesn't crash)
	count, err := st.CleanupOldChecks(ctx, 1*time.Hour)
	if err != nil {
		t.Errorf("CleanupOldChecks failed: %v", err)
	}
	t.Logf("Cleaned up %d old records", count)
}
