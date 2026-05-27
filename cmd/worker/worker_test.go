package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
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
	"github.com/toanle88/healthcheck/internal/store"
)

func runMigrations(ctx context.Context, dbURL string) error {
	st, err := store.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer st.Close()

	db := stdlib.OpenDBFromPool(st.DB)
	defer db.Close()

	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		return err
	}

	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

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

		// Run migrations so the DB is ready
		if err := runMigrations(ctx, dbURL); err != nil {
			panic(fmt.Sprintf("failed to run migrations: %v", err))
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

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"172.16.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}
	for _, tc := range tests {
		ip := net.ParseIP(tc.ip)
		if isPrivateIP(ip) != tc.expected {
			t.Errorf("expected isPrivateIP(%s) to be %t, got %t", tc.ip, tc.expected, !tc.expected)
		}
	}
}

func TestNewSafeHTTPClient(t *testing.T) {
	// 1. Dev mode: should allow loopback
	clientDev := newSafeHTTPClient("local")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := clientDev.Get(server.URL)
	if err != nil {
		t.Fatalf("expected no error in local dev environment connecting to loopback, got: %v", err)
	}
	resp.Body.Close()

	// 2. Production mode: should block loopback/private IPs
	clientProd := newSafeHTTPClient("production")
	_, err = clientProd.Get(server.URL)
	if err == nil {
		t.Errorf("expected error connecting to loopback server in production mode due to SSRF protection, got nil")
	}

	// 3. Production mode: request to a public IP with closed port to cover dial failure paths
	_, err = clientProd.Get("http://8.8.8.8:9999")
	if err == nil {
		t.Errorf("expected connection failure for closed public port, got nil")
	}

	// 4. Production mode: request to an unresolvable host to cover resolver failure path
	_, err = clientProd.Get("http://unresolvable.invalid")
	if err == nil {
		t.Errorf("expected resolution failure for invalid host, got nil")
	}
}

func TestSendWebhookAlert(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	os.Setenv("ALERT_WEBHOOK_URL", server.URL)
	defer os.Unsetenv("ALERT_WEBHOOK_URL")

	sendWebhookAlert(context.Background(), http.DefaultClient, "http://test-target.com", "down", "up", 123*time.Millisecond)

	if !called {
		t.Errorf("expected webhook server to be called, but it wasn't")
	}
}

func TestRunPingAndCheck(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("skipping TestRunPingAndCheck: DATABASE_URL_TEST not set")
	}

	ctx := context.Background()
	st, err := store.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	defer st.Close()

	// Clear table data
	_, _ = st.DB.Exec(ctx, "DELETE FROM checks")
	_, _ = st.DB.Exec(ctx, "DELETE FROM targets")

	// Set up a mock http server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Insert target
	targetURL := ts.URL
	target, err := st.InsertTarget(ctx, store.InsertTargetParams{
		Name:             "Test Worker Target",
		URL:              targetURL,
		Method:           "GET",
		Headers:          "",
		ExpectedStatus:   200,
		ResponseContains: "",
		FailureThreshold: 3,
	})
	if err != nil {
		t.Fatalf("failed to insert target: %v", err)
	}

	// Run runPingAndCheck
	runPingAndCheck(ctx, http.DefaultClient, st, nil, target)

	// Verify a check was recorded in the database
	checks, err := st.GetLatestChecks(ctx)
	if err != nil {
		t.Fatalf("failed to get latest checks: %v", err)
	}
	if len(checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(checks))
	} else if checks[0].Status != "up" {
		t.Errorf("expected check status 'up', got '%s'", checks[0].Status)
	}
}

func TestRunBatch(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("skipping TestRunBatch: DATABASE_URL_TEST not set")
	}

	ctx := context.Background()
	st, err := store.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	defer st.Close()

	// Set up mock http server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	targets := []store.Target{
		{Name: "Target 1", URL: ts.URL, Method: "GET", ExpectedStatus: 200, IsActive: true},
	}

	runBatch(ctx, http.DefaultClient, st, nil, targets)
}

func TestWorkerMain(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("skipping TestWorkerMain: DATABASE_URL_TEST not set")
	}

	os.Setenv("WORKER_MODE", "job")
	os.Setenv("DATABASE_URL", dbURL)
	os.Setenv("ENV", "local")
	defer func() {
		os.Unsetenv("WORKER_MODE")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("ENV")
	}()

	// Run main
	main()
}

func TestWorkerMainService(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("skipping TestWorkerMainService: DATABASE_URL_TEST not set")
	}

	os.Setenv("WORKER_MODE", "service")
	os.Setenv("DATABASE_URL", dbURL)
	os.Setenv("ENV", "local")
	defer func() {
		os.Unsetenv("WORKER_MODE")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("ENV")
	}()

	// Start main in background
	go main()

	// Wait for OTel / Server to start
	time.Sleep(100 * time.Millisecond)

	// Send SIGTERM to terminate graceful wait loop
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("failed to find own process: %v", err)
	}
	err = p.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("failed to send SIGTERM: %v", err)
	}

	// Wait for shutdown to complete
	time.Sleep(100 * time.Millisecond)
}

func TestSafeHTTPClientRedirects(t *testing.T) {
	// Create client in prod mode to trigger SSRF validation on redirects
	client := newSafeHTTPClient("production")

	// 1. More than 3 redirects failure
	tsRedirectLoop := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirect", http.StatusFound)
	}))
	defer tsRedirectLoop.Close()

	_, err := client.Get(tsRedirectLoop.URL)
	if err == nil {
		t.Errorf("expected error after redirect loop, got nil")
	}

	// 2. Redirect to private IP (SSRF)
	tsRedirectPrivate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:9999", http.StatusFound)
	}))
	defer tsRedirectPrivate.Close()

	_, err = client.Get(tsRedirectPrivate.URL)
	if err == nil {
		t.Errorf("expected SSRF error redirecting to loopback, got nil")
	}
}
