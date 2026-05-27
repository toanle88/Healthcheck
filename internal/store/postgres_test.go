package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/toanle88/healthcheck/internal/migrations"
)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	var pgContainer *postgres.PostgresContainer
	var err error

	if dbURL == "" {
		ctx := context.Background()
		pgContainer, err = postgres.Run(ctx,
			"postgres:18-alpine",
			postgres.WithDatabase("healthcheck"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("postgres"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second),
			),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to start postgres container: %v", err))
		}

		dbURL, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			panic(fmt.Sprintf("failed to get connection string: %v", err))
		}

		os.Setenv("DATABASE_URL_TEST", dbURL)
	}

	code := m.Run()

	if pgContainer != nil {
		ctx := context.Background()
		if err := pgContainer.Terminate(ctx); err != nil {
			panic(fmt.Sprintf("failed to terminate container: %v", err))
		}
	}

	os.Exit(code)
}

func TestStoreIntegration(t *testing.T) {
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

	// 4. Test Migrations
	db := stdlib.OpenDBFromPool(st.DB)
	defer db.Close()

	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		t.Fatalf("Failed to create migration driver: %v", err)
	}

	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		t.Fatalf("Failed to create source driver: %v", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		t.Fatalf("Failed to initialize migrate: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Migrations failed: %v", err)
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

	// 8b. Test GetTargets
	targets, err := st.GetTargets(ctx)
	if err != nil {
		t.Errorf("GetTargets failed: %v", err)
	}
	if len(targets) < 2 {
		t.Errorf("expected at least 2 targets, got %d", len(targets))
	}

	// 8c. Test GetHistoricalChecks
	histChecks, err := st.GetHistoricalChecks(ctx, target, 10)
	if err != nil {
		t.Errorf("GetHistoricalChecks failed: %v", err)
	}
	if len(histChecks) != 1 {
		t.Errorf("expected 1 historical check for target %s, got %d", target, len(histChecks))
	}

	// 8d. Test GetPreviousCheckStatus
	prevStatus, err := st.GetPreviousCheckStatus(ctx, target)
	if err != nil {
		t.Errorf("GetPreviousCheckStatus failed: %v", err)
	}
	if prevStatus != "up" {
		t.Errorf("expected previous status to be 'up', got '%s'", prevStatus)
	}

	// 8e. Test DeleteTarget
	err = st.DeleteTarget(ctx, insertedTarget.ID)
	if err != nil {
		t.Errorf("DeleteTarget failed: %v", err)
	}
	// Verify it was deleted
	targetsAfterDelete, err := st.GetTargets(ctx)
	if err != nil {
		t.Errorf("GetTargets after delete failed: %v", err)
	}
	for _, tg := range targetsAfterDelete {
		if tg.ID == insertedTarget.ID {
			t.Errorf("expected target %s to be deleted, but it still exists", targetURL)
		}
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

func TestStoreFailures(t *testing.T) {
	ctx := context.Background()

	// 1. Connection failure
	_, err := New(ctx, "postgres://invalid:invalid@localhost:5432/invalid?sslmode=disable")
	if err == nil {
		t.Errorf("expected connection error for invalid database URL, got nil")
	}

	// 2. Delete non-existent target failure
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		return
	}
	st, err := New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	defer st.Close()

	err = st.DeleteTarget(ctx, -999) // Negative ID, won't exist
	if err == nil {
		t.Errorf("expected error deleting non-existent target, got nil")
	}
}

func TestStoreContextErrors(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("skipping TestStoreContextErrors: DATABASE_URL_TEST not set")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Already cancelled context

	st, err := New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	defer st.Close()

	// 1. GetLatestChecks error
	_, err = st.GetLatestChecks(ctx)
	if err == nil {
		t.Errorf("expected error with cancelled context, got nil")
	}

	// 2. GetTargets error
	_, err = st.GetTargets(ctx)
	if err == nil {
		t.Errorf("expected error with cancelled context, got nil")
	}

	// 3. InsertTarget error
	_, err = st.InsertTarget(ctx, "Name", "http://url.com", "GET", "", 200, "", 3)
	if err == nil {
		t.Errorf("expected error with cancelled context, got nil")
	}

	// 4. DeleteTarget error
	err = st.DeleteTarget(ctx, 1)
	if err == nil {
		t.Errorf("expected error with cancelled context, got nil")
	}

	// 5. GetHistoricalChecks error
	_, err = st.GetHistoricalChecks(ctx, "http://url.com", 10)
	if err == nil {
		t.Errorf("expected error with cancelled context, got nil")
	}

	// 6. GetPreviousCheckStatus error
	_, err = st.GetPreviousCheckStatus(ctx, "http://url.com")
	if err == nil {
		t.Errorf("expected error with cancelled context, got nil")
	}

	// 7. CleanupOldChecks error
	_, err = st.CleanupOldChecks(ctx, 1*time.Hour)
	if err == nil {
		t.Errorf("expected error with cancelled context, got nil")
	}

	// 8. UpdateTargetAlertState error
	_, _, _, err = st.UpdateTargetAlertState(ctx, "http://url.com", "up")
	if err == nil {
		t.Errorf("expected error with cancelled context, got nil")
	}
}
