package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/toanle88/healthcheck/internal/store"
)

// mockStore is a simple mock that implements the Storer interface.
type mockStore struct {
	checks []store.Check
	err    error
}

func (m *mockStore) GetLatestChecks(ctx context.Context) ([]store.Check, error) {
	return m.checks, m.err
}

func (m *mockStore) GetTargets(ctx context.Context) ([]store.Target, error) {
	return nil, nil
}

func (m *mockStore) InsertTarget(ctx context.Context, name, url, method, headers string, expectedStatus int, responseContains string) (store.Target, error) {
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

func TestHealth(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	h := New(&mockStore{})
	r := gin.New()
	r.GET("/health", h.Health)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", resp["status"])
	}
}

func TestStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedChecks := []store.Check{
		{Target: "test.com", Status: "up", LatencyMs: 100, CheckedAt: time.Now()},
	}

	h := New(&mockStore{checks: expectedChecks})
	r := gin.New()
	r.GET("/api/status", h.Status)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/status", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Check if we got our mocked data back
	checks := resp["checks"].([]interface{})
	if len(checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(checks))
	}
}

func TestDocs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := New(&mockStore{})
	r := gin.New()
	r.GET("/openapi.json", h.OpenAPISpec)
	r.GET("/docs", h.Docs)

	// Test OpenAPI Spec
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/openapi.json", nil)
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("Expected status 200 for openapi.json, got %d", w1.Code)
	}
	contentType1 := w1.Header().Get("Content-Type")
	if contentType1 != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType1)
	}

	// Test Docs page
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/docs", nil)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for docs, got %d", w2.Code)
	}
	contentType2 := w2.Header().Get("Content-Type")
	if contentType2 != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type text/html; charset=utf-8, got %s", contentType2)
	}
}
