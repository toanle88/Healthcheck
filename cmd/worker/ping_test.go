package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPingTarget(t *testing.T) {
	// 1. Test "up" scenario
	tsUp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer tsUp.Close()

	status, latency := pingTarget(context.Background(), tsUp.URL)
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

	status, _ = pingTarget(context.Background(), tsDown.URL)
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

	status, _ = pingTarget(ctx, tsTimeout.URL)
	if status != "down" {
		t.Errorf("Expected status 'down' on timeout, got %s", status)
	}
}
