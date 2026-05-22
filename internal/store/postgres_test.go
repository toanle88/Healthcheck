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
	// Insert target first, so GetLatestChecks finds it
	_, err = st.InsertTarget(ctx, "Test Integration Target", target, "GET", "", 200, "", 3)
	if err != nil {
		t.Fatalf("InsertTarget for integration test failed: %v", err)
	}

	err = st.InsertCheck(ctx, target, "up", 123)
	if err != nil {
		t.Errorf("InsertCheck failed: %v", err)
	}

	// 6. Test GetLatestChecks
	checks, err := st.GetLatestChecks(ctx)
	if err != nil {
		t.Errorf("GetLatestChecks failed: %v", err)
	}

	foundTarget := false
	for _, c := range checks {
		if c.Target == target {
			foundTarget = true
			if c.Status != "up" {
				t.Errorf("Expected check status to be 'up', got '%s'", c.Status)
			}
			break
		}
	}
	if !foundTarget {
		t.Errorf("Expected to find target %s in latest checks, but it was missing", target)
	}

	// 7. Test InsertTarget
	targetURL := "http://test-alert-transition.com"
	insertedTarget, err := st.InsertTarget(ctx, "Test Alert Target", targetURL, "GET", "", 200, "", 3)
	if err != nil {
		t.Fatalf("InsertTarget failed: %v", err)
	}
	if insertedTarget.FailureThreshold != 3 {
		t.Errorf("Expected FailureThreshold to be 3, got %d", insertedTarget.FailureThreshold)
	}

	// 8. Test UpdateTargetAlertState Transitions
	// First failure check (consecutive_failures = 1/3) -> shouldAlert = false, alert status remains "up"
	shouldAlert, oldAlert, newAlert, err := st.UpdateTargetAlertState(ctx, targetURL, "down")
	if err != nil {
		t.Fatalf("UpdateTargetAlertState 1st fail failed: %v", err)
	}
	if shouldAlert || newAlert != "up" || oldAlert != "up" {
		t.Errorf("1st fail: expected shouldAlert=false, oldAlert=up, newAlert=up; got shouldAlert=%t, old=%s, new=%s", shouldAlert, oldAlert, newAlert)
	}

	// Second failure check (consecutive_failures = 2/3) -> shouldAlert = false, alert status remains "up"
	shouldAlert, oldAlert, newAlert, err = st.UpdateTargetAlertState(ctx, targetURL, "down")
	if err != nil {
		t.Fatalf("UpdateTargetAlertState 2nd fail failed: %v", err)
	}
	if shouldAlert || newAlert != "up" || oldAlert != "up" {
		t.Errorf("2nd fail: expected shouldAlert=false, oldAlert=up, newAlert=up; got shouldAlert=%t, old=%s, new=%s", shouldAlert, oldAlert, newAlert)
	}

	// Third failure check (consecutive_failures = 3/3) -> shouldAlert = true, alert status becomes "down"
	shouldAlert, oldAlert, newAlert, err = st.UpdateTargetAlertState(ctx, targetURL, "down")
	if err != nil {
		t.Fatalf("UpdateTargetAlertState 3rd fail failed: %v", err)
	}
	if !shouldAlert || newAlert != "down" || oldAlert != "up" {
		t.Errorf("3rd fail: expected shouldAlert=true, oldAlert=up, newAlert=down; got shouldAlert=%t, old=%s, new=%s", shouldAlert, oldAlert, newAlert)
	}

	// Fourth failure check (consecutive_failures = 4/3) -> shouldAlert = false, alert status remains "down" (no alert trigger)
	shouldAlert, oldAlert, newAlert, err = st.UpdateTargetAlertState(ctx, targetURL, "down")
	if err != nil {
		t.Fatalf("UpdateTargetAlertState 4th fail failed: %v", err)
	}
	if shouldAlert || newAlert != "down" || oldAlert != "down" {
		t.Errorf("4th fail: expected shouldAlert=false, oldAlert=down, newAlert=down; got shouldAlert=%t, old=%s, new=%s", shouldAlert, oldAlert, newAlert)
	}

	// Recovery check -> shouldAlert = true, alert status becomes "up" (recovery alert triggered!)
	shouldAlert, oldAlert, newAlert, err = st.UpdateTargetAlertState(ctx, targetURL, "up")
	if err != nil {
		t.Fatalf("UpdateTargetAlertState recovery failed: %v", err)
	}
	if !shouldAlert || newAlert != "up" || oldAlert != "down" {
		t.Errorf("Recovery: expected shouldAlert=true, oldAlert=down, newAlert=up; got shouldAlert=%t, old=%s, new=%s", shouldAlert, oldAlert, newAlert)
	}

	// Cleanup targets for test repeatability
	_, _ = st.DB.Exec(ctx, "DELETE FROM targets WHERE url IN ($1, $2)", targetURL, target)
	_, _ = st.DB.Exec(ctx, "DELETE FROM checks WHERE target IN ($1, $2)", targetURL, target)

	// 9. Test CleanupOldChecks
	count, err := st.CleanupOldChecks(ctx, 1*time.Hour)
	if err != nil {
		t.Errorf("CleanupOldChecks failed: %v", err)
	}
	t.Logf("Cleaned up %d old records", count)
}
