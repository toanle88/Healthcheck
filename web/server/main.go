package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Set JSON structured logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	distDir := os.Getenv("DIST_DIR")
	if distDir == "" {
		distDir = "./dist"
	}

	// 1. Generate env.js dynamically at startup
	type EnvConfig struct {
		APIURL        string `json:"VITE_API_URL"`
		AppVersion    string `json:"VITE_APP_VERSION"`
		EntraClientID string `json:"VITE_ENTRA_CLIENT_ID"`
		EntraTenantID string `json:"VITE_ENTRA_TENANT_ID"`
	}

	envObj := EnvConfig{
		APIURL:        os.Getenv("VITE_API_URL"),
		AppVersion:    os.Getenv("VITE_APP_VERSION"),
		EntraClientID: os.Getenv("VITE_ENTRA_CLIENT_ID"),
		EntraTenantID: os.Getenv("VITE_ENTRA_TENANT_ID"),
	}

	jsonData, err := json.Marshal(envObj)
	if err != nil {
		slog.Error("Failed to marshal env config", "err", err)
		os.Exit(1)
	}

	envJSContent := fmt.Sprintf("window.ENV = %s;", jsonData)

	envFilePath := filepath.Join(distDir, "env.js")
	if err := os.WriteFile(envFilePath, []byte(envJSContent), 0644); err != nil {
		slog.Error("Failed to write env.js", "err", err)
		os.Exit(1)
	}
	slog.Info("Successfully generated env.js at startup")

	// 2. Setup the SPA and Security Headers Handler
	mux := http.NewServeMux()

	// Create SPA handler
	fileServer := http.FileServer(http.Dir(distDir))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (Hardened)
		// Removed 'unsafe-inline' and external script CDN sources from script-src
		csp := "default-src 'self'; " +
			"script-src 'self'; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"font-src 'self' https://fonts.gstatic.com; " +
			"connect-src 'self' https://*.ciamlogin.com https://*.azurecontainerapps.io http://localhost:8080;"
		w.Header().Set("Content-Security-Policy", csp)

		// Check if file exists, if not serve index.html for SPA routing
		path := filepath.Clean(r.URL.Path)
		fullPath := filepath.Join(distDir, path)

		// Hardening: Verify resolved path remains inside distDir to prevent directory traversal
		absDistDir, err := filepath.Abs(distDir)
		if err != nil {
			slog.Error("Failed to get absolute path of distDir", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		absFullPath, err := filepath.Abs(fullPath)
		if err != nil {
			slog.Error("Failed to get absolute path of fullPath", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !strings.HasPrefix(absFullPath, absDistDir) {
			slog.Warn("Directory traversal attempt detected", "path", r.URL.Path)
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}

		stat, err := os.Stat(fullPath)
		if err != nil || stat.IsDir() {
			// Serve index.html
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}

		fileServer.ServeHTTP(w, r)
	})

	mux.Handle("/", handler)

	slog.Info("Starting frontend webserver", "port", port, "distDir", distDir)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("Webserver failed", "err", err)
		os.Exit(1)
	}
}
