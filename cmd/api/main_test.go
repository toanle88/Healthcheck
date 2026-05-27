package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/toanle88/healthcheck/internal/config"
	"github.com/toanle88/healthcheck/internal/handler"
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

type mockStore struct{}

func (m *mockStore) GetLatestChecks(ctx context.Context) ([]store.Check, error) {
	return nil, nil
}
func (m *mockStore) GetTargets(ctx context.Context) ([]store.Target, error) {
	return nil, nil
}
func (m *mockStore) InsertTarget(ctx context.Context, params store.InsertTargetParams) (store.Target, error) {
	return store.Target{}, nil
}
func (m *mockStore) DeleteTarget(ctx context.Context, id int) error {
	return nil
}
func (m *mockStore) GetHistoricalChecks(ctx context.Context, target string, limit int) ([]store.Check, error) {
	return nil, nil
}
func (m *mockStore) GetPreviousCheckStatus(ctx context.Context, target string) (string, error) {
	return "", nil
}

func TestSetupRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Environment:        "local",
		Port:               "8080",
		CORSAllowedOrigins: []string{"http://allowed.com"},
	}

	broker := handler.NewBroker()
	r := setupRouter(cfg, &store.Store{}, broker, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test public endpoint (/health)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /health, got %d", w.Code)
	}

	// Test CORS Allowed Origin Options
	wOpt := httptest.NewRecorder()
	reqOpt, _ := http.NewRequest("OPTIONS", "/health", nil)
	reqOpt.Header.Set("Origin", "http://allowed.com")
	r.ServeHTTP(wOpt, reqOpt)

	if wOpt.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content for OPTIONS preflight, got %d", wOpt.Code)
	}
	if wOpt.Header().Get("Access-Control-Allow-Origin") != "http://allowed.com" {
		t.Errorf("expected Access-Control-Allow-Origin header, got %s", wOpt.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestAPIMain(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("skipping TestAPIMain: DATABASE_URL_TEST not set")
	}

	os.Setenv("DATABASE_URL", dbURL)
	os.Setenv("PORT", "8089") // use a different port for test
	os.Setenv("ENV", "local")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("PORT")
		os.Unsetenv("ENV")
	}()

	// Start main in background
	go main()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Send SIGTERM to ourselves to trigger graceful shutdown
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("failed to find own process: %v", err)
	}
	err = p.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("failed to send SIGTERM: %v", err)
	}

	// Wait for main to exit
	time.Sleep(100 * time.Millisecond)
}
