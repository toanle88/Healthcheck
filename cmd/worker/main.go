package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/toanle88/healthcheck/internal/config"
	"github.com/toanle88/healthcheck/internal/monitor"
	"github.com/toanle88/healthcheck/internal/store"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	// --- 1. SETUP LOGGING ---
	// Using the same JSON structured logging as the API.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// --- 2. LOAD CONFIG ---
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- 4. INITIALIZE OPENTELEMETRY ---
	metricsHandler, shutdown, err := monitor.InitOTel(ctx, "healthcheck-worker")
	if err != nil {
		slog.Error("otel init failed", "err", err)
	} else {
		defer shutdown(context.Background())
		
		// Start a small HTTP server for Prometheus metrics in the background
		go func() {
			http.Handle("/metrics", metricsHandler)
			slog.Info("worker metrics server starting", "port", 8081)
			if err := http.ListenAndServe(":8081", nil); err != nil {
				slog.Error("metrics server failed", "err", err)
			}
		}()
	}

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
		// Create a span for the entire batch of health checks
		tracer := otel.Tracer("healthcheck-worker")
		batchCtx, span := tracer.Start(context.Background(), "RunBatch")
		defer span.End()

		slog.Info("running health checks", "count", len(targets))

		for _, url := range targets {
			// Create a child span for each individual ping
			_, childSpan := tracer.Start(batchCtx, "PingTarget", trace.WithAttributes(
				attribute.String("http.url", url),
			))

			pingCtx, cancel := context.WithTimeout(batchCtx, 10*time.Second)
			
			status, latency := pingTarget(pingCtx, url)
			
			childSpan.SetAttributes(
				attribute.String("health.status", status),
				attribute.Int64("health.latency_ms", latency.Milliseconds()),
			)

			// Record metrics in Prometheus
			monitor.CheckCounter.Add(batchCtx, 1, metric.WithAttributes(
				attribute.String("target", url),
				attribute.String("status", status),
			))
			monitor.LatencyHistogram.Record(batchCtx, latency.Seconds(), metric.WithAttributes(
				attribute.String("target", url),
			))

			// Record the result in the database
			if err := st.InsertCheck(context.Background(), url, status, int(latency.Milliseconds())); err != nil {
				slog.Error("failed to save check", "target", url, "err", err)
				childSpan.RecordError(err)
			} else {
				slog.Info("check recorded", "target", url, "status", status, "latency_ms", latency.Milliseconds())
			}
			
			childSpan.End()
			cancel()
		}
	})

	if err != nil {
		slog.Error("failed to schedule ping job", "err", err)
		os.Exit(1)
	}

	// --- NEW: CLEANUP JOB ---
	// Add a job to clean up old data every hour. 
	// We keep data for 24 hours.
	_, err = c.AddFunc("@hourly", func() {
		slog.Info("running database cleanup")
		
		// 24h retention period
		count, err := st.CleanupOldChecks(context.Background(), 24*time.Hour)
		if err != nil {
			slog.Error("cleanup failed", "err", err)
		} else {
			slog.Info("cleanup finished", "rows_deleted", count)
		}
	})

	if err != nil {
		slog.Error("failed to schedule cleanup job", "err", err)
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
