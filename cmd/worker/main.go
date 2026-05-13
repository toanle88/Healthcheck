package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3" // Library for scheduling tasks using cron expressions
	"github.com/toanle88/healthcheck/internal/config"
	"github.com/toanle88/healthcheck/internal/store"
)

func main() {
	// --- 1. SETUP LOGGING ---
	// Using the same JSON structured logging as the API.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// --- 2. LOAD CONFIG ---
	cfg := config.Load()

	// --- 3. SETUP CONTEXT FOR GRACEFUL SHUTDOWN ---
	// This context will be canceled when the OS sends a termination signal.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- 4. CONNECT TO DATABASE ---
	// We need the DB to store our ping results.
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("worker db connect failed", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	// --- 5. INITIALIZE CRON SCHEDULER ---
	// cron.New() creates a new scheduler.
	c := cron.New()

	// List of targets to monitor as defined in PROJECT.md
	targets := []string{
		"http://httpbin.org/get",
		"https://github.com",
		"https://azure.microsoft.com/en-us/status/",
	}

	// Add a job to run every minute
	// "@every 1m" is a shorthand for "0 * * * * *" (every minute at the 0th second)
	_, err = c.AddFunc("@every 1m", func() {
		slog.Info("running health checks", "count", len(targets))

		for _, url := range targets {
			// We use a background context with a timeout for each individual ping
			// so one slow target doesn't block the whole worker forever.
			pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			
			status, latency := pingTarget(pingCtx, url)
			
			// Record the result in the database
			if err := st.InsertCheck(context.Background(), url, status, int(latency.Milliseconds())); err != nil {
				slog.Error("failed to save check", "target", url, "err", err)
			} else {
				slog.Info("check recorded", "target", url, "status", status, "latency_ms", latency.Milliseconds())
			}
			
			cancel() // Clean up the timeout context
		}
	})

	if err != nil {
		slog.Error("failed to schedule cron job", "err", err)
		os.Exit(1)
	}

	// Start the cron scheduler in the background
	c.Start()
	slog.Info("worker started", "interval", "1m")

	// --- 6. WAIT FOR SHUTDOWN ---
	<-ctx.Done() // Block here until we get a signal (Ctrl+C, etc.)
	slog.Info("worker shutting down")

	// Stop the cron scheduler and wait for active jobs to finish
	stopCtx := c.Stop()
	select {
	case <-stopCtx.Done():
		slog.Info("worker stopped cleanly")
	case <-time.After(10 * time.Second):
		slog.Warn("worker shutdown timed out, forcing exit")
	}
}

// pingTarget performs an HTTP GET request to the URL and returns the status ("up"/"down") and latency.
func pingTarget(ctx context.Context, url string) (string, time.Duration) {
	start := time.Now()

	// Create a new HTTP request with the context (for timeout support)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "down", time.Since(start)
	}

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "down", time.Since(start)
	}
	defer resp.Body.Close()

	// If we get a 2xx status code, we consider it "up"
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "up", time.Since(start)
	}

	return "down", time.Since(start)
}
