package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/toanle88/healthcheck/internal/store"
)

func TestPingTarget(t *testing.T) {
	// 1. Test "up" scenario
	tsUp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer tsUp.Close()

	status, latency := pingTarget(context.Background(), http.DefaultClient, store.Target{URL: tsUp.URL, Method: "GET", ExpectedStatus: 200})
	if status != "up" {
		t.Errorf("Expected status 'up', got %s", status)
	}
	if latency <= 0 {
		t.Errorf("Expected positive latency, got %v", latency)
	}

	// 2. Test "down" scenario (500 Error)
	tsDown := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tsDown.Close()

	status, _ = pingTarget(context.Background(), http.DefaultClient, store.Target{URL: tsDown.URL, Method: "GET", ExpectedStatus: 200})
	if status != "down" {
		t.Errorf("Expected status 'down', got %s", status)
	}

	// 3. Test "timeout" scenario
	tsTimeout := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer tsTimeout.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	status, _ = pingTarget(ctx, http.DefaultClient, store.Target{URL: tsTimeout.URL, Method: "GET", ExpectedStatus: 200})
	if status != "down" {
		t.Errorf("Expected status 'down' on timeout, got %s", status)
	}
}

func TestPingTargetSyntheticMonitoring(t *testing.T) {
	// 1. Test custom expected status code (e.g. 418 Teapot)
	tsTeapot := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer tsTeapot.Close()

	// Should be 'up' because we expect 418
	status, _ := pingTarget(context.Background(), http.DefaultClient, store.Target{
		URL:            tsTeapot.URL,
		Method:         "GET",
		ExpectedStatus: 418,
	})
	if status != "up" {
		t.Errorf("Expected status 'up' for expected 418 status code, got %s", status)
	}

	// Should be 'down' because we expected 200 but got 418
	status, _ = pingTarget(context.Background(), http.DefaultClient, store.Target{
		URL:            tsTeapot.URL,
		Method:         "GET",
		ExpectedStatus: 200,
	})
	if status != "down" {
		t.Errorf("Expected status 'down' when expected 200 but got 418 status code, got %s", status)
	}

	// 2. Test response body contains check
	tsBody := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "service_operational", "version": "1.0"}`))
	}))
	defer tsBody.Close()

	// Should be 'up' because body contains 'operational'
	status, _ = pingTarget(context.Background(), http.DefaultClient, store.Target{
		URL:              tsBody.URL,
		Method:           "GET",
		ExpectedStatus:   200,
		ResponseContains: "operational",
	})
	if status != "up" {
		t.Errorf("Expected status 'up' when response body contains matching substring, got %s", status)
	}

	// Should be 'down' because body does not contain 'maintenance'
	status, _ = pingTarget(context.Background(), http.DefaultClient, store.Target{
		URL:              tsBody.URL,
		Method:           "GET",
		ExpectedStatus:   200,
		ResponseContains: "maintenance",
	})
	if status != "down" {
		t.Errorf("Expected status 'down' when response body does not contain substring, got %s", status)
	}

	// 3. Test custom request headers injection
	tsHeaders := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Secret-Auth") == "SuperSecret" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer tsHeaders.Close()

	// Should be 'up' when header is correctly set
	status, _ = pingTarget(context.Background(), http.DefaultClient, store.Target{
		URL:            tsHeaders.URL,
		Method:         "GET",
		Headers:        `{"X-Secret-Auth": "SuperSecret"}`,
		ExpectedStatus: 200,
	})
	if status != "up" {
		t.Errorf("Expected status 'up' when secret headers are correctly injected, got %s", status)
	}

	// Should be 'down' when header is missing/incorrect
	status, _ = pingTarget(context.Background(), http.DefaultClient, store.Target{
		URL:            tsHeaders.URL,
		Method:         "GET",
		Headers:        `{"X-Secret-Auth": "WrongSecret"}`,
		ExpectedStatus: 200,
	})
	if status != "down" {
		t.Errorf("Expected status 'down' when headers are incorrect, got %s", status)
	}
}
