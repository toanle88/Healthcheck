package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCheckHealth(t *testing.T) {
	// 1. Success case (200 OK)
	serverOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer serverOK.Close()

	uOK, err := url.Parse(serverOK.URL)
	if err != nil {
		t.Fatalf("failed to parse test server url: %v", err)
	}

	if err := checkHealth(uOK.Port()); err != nil {
		t.Errorf("expected no error for 200 OK, got: %v", err)
	}

	// 2. Error case (500 Internal Server Error)
	serverError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer serverError.Close()

	uErr, err := url.Parse(serverError.URL)
	if err != nil {
		t.Fatalf("failed to parse test server url: %v", err)
	}

	if err := checkHealth(uErr.Port()); err == nil {
		t.Errorf("expected error for 500 status code, got nil")
	}

	// 3. Network connection failure case
	if err := checkHealth("9999"); err == nil {
		t.Errorf("expected connection error for invalid port, got nil")
	}
}
