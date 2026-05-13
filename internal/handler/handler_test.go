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
