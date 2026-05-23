package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/toanle88/healthcheck/internal/config"
	"github.com/toanle88/healthcheck/internal/handler"
	"github.com/toanle88/healthcheck/internal/store"
)

type mockStore struct{}

func (m *mockStore) GetLatestChecks(ctx context.Context) ([]store.Check, error) {
	return nil, nil
}
func (m *mockStore) GetTargets(ctx context.Context) ([]store.Target, error) {
	return nil, nil
}
func (m *mockStore) InsertTarget(ctx context.Context, name, url, method, headers string, expectedStatus int, responseContains string, failureThreshold int) (store.Target, error) {
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
