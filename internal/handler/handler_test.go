package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/healthcheck/internal/store"
)

// mockStore is a simple mock that implements the Storer interface.
type mockStore struct {
	checks           []store.Check
	err              error
	targets          []store.Target
	getTargetsErr    error
	insertTargetErr  error
	deleteErr        error
	getHistoricalErr error
}

func (m *mockStore) GetLatestChecks(ctx context.Context) ([]store.Check, error) {
	return m.checks, m.err
}

func (m *mockStore) GetTargets(ctx context.Context) ([]store.Target, error) {
	return m.targets, m.getTargetsErr
}

func (m *mockStore) InsertTarget(ctx context.Context, params store.InsertTargetParams) (store.Target, error) {
	return store.Target{}, m.insertTargetErr
}

func (m *mockStore) DeleteTarget(ctx context.Context, id int) error {
	return m.deleteErr
}

func (m *mockStore) GetHistoricalChecks(ctx context.Context, target string, limit int) ([]store.Check, error) {
	return nil, m.getHistoricalErr
}

func (m *mockStore) GetPreviousCheckStatus(ctx context.Context, target string) (string, error) {
	return "", nil
}

func TestHealth(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	h := New(&mockStore{}, nil)
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

	h := New(&mockStore{checks: expectedChecks}, nil)
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

	h := New(&mockStore{}, nil)
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

func TestDocsWithEnv(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set env vars
	t.Setenv("ENTRA_TENANT_ID", "test-tenant-id")
	t.Setenv("ENTRA_CLIENT_ID", "test-client-id")
	t.Setenv("ENTRA_TENANT_DOMAIN", "test-tenant.ciamlogin.com")

	h := New(&mockStore{}, nil)
	r := gin.New()
	r.GET("/openapi.json", h.OpenAPISpec)
	r.GET("/docs", h.Docs)

	// Test OpenAPI Spec with Env
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/openapi.json", nil)
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("Expected status 200 for openapi.json, got %d", w1.Code)
	}
	if !strings.Contains(w1.Body.String(), "test-tenant.ciamlogin.com") {
		t.Errorf("Expected openapi.json to contain tenant domain, got %s", w1.Body.String())
	}
	if !strings.Contains(w1.Body.String(), "api://test-client-id/access_as_user") {
		t.Errorf("Expected openapi.json to contain client ID scope, got %s", w1.Body.String())
	}

	// Test Docs page with Env
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/docs", nil)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for docs, got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "test-client-id") {
		t.Errorf("Expected docs to contain client ID, got %s", w2.Body.String())
	}
}

func TestHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := New(&mockStore{}, nil)
	r := gin.New()
	r.GET("/api/history", h.History)

	// 1. Missing target query param
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/api/history", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for missing target, got %d", w1.Code)
	}

	// 2. Success case
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/history?target=test.com", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w2.Code)
	}
}

func TestGetTargets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := New(&mockStore{}, nil)
	r := gin.New()
	r.GET("/api/targets", h.GetTargets)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/targets", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}
}

func TestGetTargets_Redaction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockTargets := []store.Target{
		{
			ID:      1,
			Name:    "Test",
			URL:     "http://test.com",
			Headers: `{"Authorization":"Bearer secret-token"}`,
		},
	}

	// 1. Test as Admin
	{
		h := New(&mockStore{targets: mockTargets}, nil)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("claims", jwt.MapClaims{
				"roles": []interface{}{"Healthcheck.Admin"},
			})
			c.Next()
		})
		r.GET("/api/targets", h.GetTargets)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/targets", nil)
		r.ServeHTTP(w, req)

		var resp []store.Target
		json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp) != 1 || resp[0].Headers != `{"Authorization":"Bearer secret-token"}` {
			t.Errorf("Admin expected to see headers, got: %s", resp[0].Headers)
		}
	}

	// 2. Test as Non-Admin (headers should be redacted)
	{
		h := New(&mockStore{targets: mockTargets}, nil)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("claims", jwt.MapClaims{
				"roles": []interface{}{"Healthcheck.Reader"},
			})
			c.Next()
		})
		r.GET("/api/targets", h.GetTargets)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/targets", nil)
		r.ServeHTTP(w, req)

		var resp []store.Target
		json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp) != 1 || resp[0].Headers != "" {
			t.Errorf("Non-admin expected redacted headers, got: %s", resp[0].Headers)
		}
	}
}

func TestCreateTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := New(&mockStore{}, nil)
	r := gin.New()
	r.POST("/api/targets", h.CreateTarget)

	// 1. Validation error (missing required URL)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/targets", bytes.NewBufferString(`{"name":"Test"}`))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for missing URL, got %d", w1.Code)
	}

	// 2. Unsupported HTTP method
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/targets", bytes.NewBufferString(`{"name":"Test","url":"http://test.com","method":"INVALID"}`))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for unsupported HTTP method, got %d", w2.Code)
	}

	// 3. Invalid headers format
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/api/targets", bytes.NewBufferString(`{"name":"Test","url":"http://test.com","headers":"invalid-json"}`))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for invalid headers JSON, got %d", w3.Code)
	}

	// 4. Invalid expected status code
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("POST", "/api/targets", bytes.NewBufferString(`{"name":"Test","url":"http://test.com","expected_status":999}`))
	req4.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for invalid expected status code, got %d", w4.Code)
	}

	// 5. Success case
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest("POST", "/api/targets", bytes.NewBufferString(`{"name":"Test","url":"http://test.com","expected_status":200}`))
	req5.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w5, req5)
	if w5.Code != http.StatusCreated {
		t.Errorf("expected 201 Created, got %d", w5.Code)
	}
}

func TestDeleteTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := New(&mockStore{}, nil)
	r := gin.New()
	r.DELETE("/api/targets/:id", h.DeleteTarget)

	// 1. Invalid target ID format
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("DELETE", "/api/targets/abc", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for invalid target ID, got %d", w1.Code)
	}

	// 2. Success case
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("DELETE", "/api/targets/123", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w2.Code)
	}
}

func TestChaosRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := New(&mockStore{}, nil)
	r := gin.New()
	r.GET("/api/test/error", h.TestError)
	r.GET("/api/test/slow", h.TestSlow)

	// Test Error route
	wErr := httptest.NewRecorder()
	reqErr, _ := http.NewRequest("GET", "/api/test/error", nil)
	r.ServeHTTP(wErr, reqErr)
	if wErr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for error route, got %d", wErr.Code)
	}

	// Test Slow route
	wSlow := httptest.NewRecorder()
	reqSlow, _ := http.NewRequest("GET", "/api/test/slow", nil)
	r.ServeHTTP(wSlow, reqSlow)
	if wSlow.Code != http.StatusOK {
		t.Errorf("expected 200 for slow route, got %d", wSlow.Code)
	}
}

func TestBrokerAndStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := NewBroker()
	go broker.Start(ctx)

	// Test StreamStatus with nil broker
	hNil := New(&mockStore{}, nil)
	rNil := gin.New()
	rNil.GET("/api/status/stream", hNil.StreamStatus)
	wNil := httptest.NewRecorder()
	reqNil, _ := http.NewRequest("GET", "/api/status/stream", nil)
	rNil.ServeHTTP(wNil, reqNil)
	if wNil.Code != http.StatusNotImplemented {
		t.Errorf("expected 510 NotImplemented for nil broker stream, got %d", wNil.Code)
	}

	// Test Broker Broadcast channel logic
	broker.Broadcast([]store.Check{{Target: "test.com", Status: "up"}})
}

func TestStreamStatusSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)

	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go broker.Start(ctx)

	// Mock data for initial checks and broadcast checks
	initialChecks := []store.Check{{Target: "initial.com", Status: "up"}}
	broadcastChecks := []store.Check{{Target: "broadcast.com", Status: "down"}}

	storeMock := &mockStore{checks: initialChecks}
	h := New(storeMock, broker)

	r := gin.New()
	r.GET("/api/status/stream", h.StreamStatus)

	server := httptest.NewServer(r)
	defer server.Close()

	// Make request
	client := server.Client()
	reqCtx, reqCancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(reqCtx, "GET", server.URL+"/api/status/stream", nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Verify headers
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		t.Errorf("expected Content-Type to start with text/event-stream, got %s", contentType)
	}

	// Trigger broadcast in background
	go func() {
		time.Sleep(50 * time.Millisecond)
		broker.Broadcast(broadcastChecks)
		time.Sleep(50 * time.Millisecond)
		reqCancel()
	}()

	// Read body content
	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, resp.Body)

	body := buf.String()
	if !strings.Contains(body, "initial.com") {
		t.Errorf("expected body to contain initial.com, got:\n%s", body)
	}
	if !strings.Contains(body, "broadcast.com") {
		t.Errorf("expected body to contain broadcast.com, got:\n%s", body)
	}
}

func TestHandlerDBErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. GetTargets DB Error (500)
	{
		h := New(&mockStore{getTargetsErr: errors.New("db error")}, nil)
		r := gin.New()
		r.GET("/api/targets", h.GetTargets)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/targets", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 InternalServerError for targets DB error, got %d", w.Code)
		}
	}

	// 2. DeleteTarget Not Found (404)
	{
		h := New(&mockStore{deleteErr: pgx.ErrNoRows}, nil)
		r := gin.New()
		r.DELETE("/api/targets/:id", h.DeleteTarget)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/targets/123", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 NotFound for delete target not found, got %d", w.Code)
		}
	}

	// 3. DeleteTarget DB Error (500)
	{
		h := New(&mockStore{deleteErr: errors.New("db error")}, nil)
		r := gin.New()
		r.DELETE("/api/targets/:id", h.DeleteTarget)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/targets/123", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 InternalServerError for delete target DB error, got %d", w.Code)
		}
	}

	// 4. CreateTarget DB Error (500)
	{
		h := New(&mockStore{insertTargetErr: errors.New("db error")}, nil)
		r := gin.New()
		r.POST("/api/targets", h.CreateTarget)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/targets", bytes.NewBufferString(`{"name":"Test","url":"http://test.com","expected_status":200}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 InternalServerError for create target DB error, got %d", w.Code)
		}
	}

	// 5. History DB Error (500)
	{
		h := New(&mockStore{getHistoricalErr: errors.New("db error")}, nil)
		r := gin.New()
		r.GET("/api/history", h.History)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/history?target=test.com", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 InternalServerError for history DB error, got %d", w.Code)
		}
	}
}
