package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

var Version = "dev"

func main() {
	// --- 1. SETUP LOGGING ---
	// Using the same JSON structured logging as the API.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("starting healthcheck worker", "version", Version)

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

	// --- 5. DETERMINE EXECUTION MODE ---
	// "job" = run once and exit (Azure Jobs)
	// "service" = run forever with internal cron (Local / Traditional)
	mode := os.Getenv("WORKER_MODE")
	if mode == "job" {
		slog.Info("running in JOB mode (one-time execution)")

		dbTargets, err := st.GetTargets(ctx)
		if err != nil {
			slog.Error("failed to get targets from database", "err", err)
			os.Exit(1)
		}

		slog.Info("executing health pings", "count", len(dbTargets))
		for _, target := range dbTargets {
			if target.IsActive {
				runPingAndCheck(ctx, st, nil, target.URL)
			}
		}

		// Run Cleanup too
		slog.Info("executing database cleanup")
		count, _ := st.CleanupOldChecks(ctx, 24*time.Hour)
		slog.Info("cleanup finished", "rows_deleted", count)

		slog.Info("job completed successfully")
		return
	}

	// --- 6. SERVICE MODE (CRON) ---
	slog.Info("running in SERVICE mode (background cron)")
	c := cron.New()

	// Add a job to run every minute
	// "@every 1m" is a shorthand for "0 * * * * *" (every minute at the 0th second)
	_, err = c.AddFunc("@every 1m", func() {
		// Create a span for the entire batch of health checks
		tracer := otel.Tracer("healthcheck-worker")
		batchCtx, span := tracer.Start(context.Background(), "RunBatch")
		defer span.End()

		dbTargets, err := st.GetTargets(batchCtx)
		if err != nil {
			slog.Error("failed to get targets from database", "err", err)
			return
		}

		slog.Info("running health checks", "count", len(dbTargets))

		for _, target := range dbTargets {
			if target.IsActive {
				runPingAndCheck(batchCtx, st, tracer, target.URL)
			}
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

func runPingAndCheck(ctx context.Context, st *store.Store, tracer trace.Tracer, url string) {
	var childSpan trace.Span
	var pingCtx context.Context = ctx
	if tracer != nil {
		pingCtx, childSpan = tracer.Start(ctx, "PingTarget", trace.WithAttributes(
			attribute.String("http.url", url),
		))
	}

	runCtx, cancel := context.WithTimeout(pingCtx, 10*time.Second)
	status, latency := pingTarget(runCtx, url)
	cancel()

	if childSpan != nil {
		childSpan.SetAttributes(
			attribute.String("health.status", status),
			attribute.Int64("health.latency_ms", latency.Milliseconds()),
		)
	}

	// Record metrics in Prometheus
	monitor.CheckCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("target", url),
		attribute.String("status", status),
	))
	monitor.LatencyHistogram.Record(ctx, latency.Seconds(), metric.WithAttributes(
		attribute.String("target", url),
	))

	// Get previous status to check for transitions
	prevStatus, err := st.GetPreviousCheckStatus(ctx, url)
	if err == nil && prevStatus != "" && prevStatus != status {
		// State transitioned! Alert!
		slog.Info("status transition detected, sending alert", "target", url, "old_status", prevStatus, "new_status", status)
		go sendWebhookAlert(context.Background(), url, prevStatus, status, latency)
	}

	// Record the result in the database
	if err := st.InsertCheck(ctx, url, status, int(latency.Milliseconds())); err != nil {
		slog.Error("failed to save check", "target", url, "err", err)
		if childSpan != nil {
			childSpan.RecordError(err)
		}
	} else {
		slog.Info("check recorded", "target", url, "status", status, "latency_ms", latency.Milliseconds())
	}

	if childSpan != nil {
		childSpan.End()
	}
}

func sendWebhookAlert(ctx context.Context, target, oldStatus, newStatus string, latency time.Duration) {
	webhookURL := os.Getenv("ALERT_WEBHOOK_URL")
	if webhookURL == "" {
		return
	}

	var statusEmoji string
	if newStatus == "up" {
		statusEmoji = "🟢"
	} else {
		statusEmoji = "🔴"
	}

	message := fmt.Sprintf("%s *Healthcheck Alert*\n*Target:* %s\n*Event:* Status changed from `%s` to `%s`\n*Latency:* %dms\n*Time:* %s",
		statusEmoji, target, oldStatus, newStatus, latency.Milliseconds(), time.Now().UTC().Format(time.RFC3339))

	payload := map[string]string{
		"text": message,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal webhook payload", "err", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		slog.Error("failed to create webhook request", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to send webhook alert", "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Error("webhook alert returned non-2xx status", "code", resp.StatusCode)
	} else {
		slog.Info("webhook alert sent successfully", "target", target, "new_status", newStatus)
	}
}
