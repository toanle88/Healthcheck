package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
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
	"golang.org/x/sync/errgroup"
)

var Version = "dev"
var alertsWG sync.WaitGroup

// isPrivateIP checks if a net.IP is loopback, link-local, or private.
func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate()
}

// newSafeHTTPClient returns an HTTP client that blocks connections to loopback/private/link-local addresses
// to prevent SSRF and DNS Rebinding.
func newSafeHTTPClient(env string) *http.Client {
	isDev := env == "local" || env == "development"

	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
				port = "80"
			}

			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}

			for _, ip := range ips {
				if !isDev && isPrivateIP(ip) {
					return nil, fmt.Errorf("SSRF prevention: connection to private/loopback address %s is blocked", ip)
				}
			}

			var lastErr error
			for _, ip := range ips {
				if !isDev && isPrivateIP(ip) {
					continue
				}
				targetAddr := net.JoinHostPort(ip.String(), port)
				conn, err := dialer.DialContext(ctx, network, targetAddr)
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, fmt.Errorf("failed to connect to resolved IPs for host: %s", host)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
}

// runBatch performs health checks concurrently for a batch of targets,
// throttling concurrency to match or stay below the database connection limit.
func runBatch(ctx context.Context, client *http.Client, st *store.Store, tracer trace.Tracer, dbTargets []store.Target) {
	g, batchCtx := errgroup.WithContext(ctx)
	g.SetLimit(8) // Limit to 8 concurrent connections to be safe with DB connections (max 10)

	for _, target := range dbTargets {
		if target.IsActive {
			t := target // shadow loop variable for goroutine safety
			g.Go(func() error {
				runPingAndCheck(batchCtx, client, st, tracer, t)
				return nil
			})
		}
	}
	_ = g.Wait()
}

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
	var metricsSrv *http.Server
	if err != nil {
		slog.Error("otel init failed", "err", err)
	} else {
		defer shutdown(context.Background())

		// Start a small HTTP server for Prometheus metrics in the background
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/metrics", metricsHandler)
			metricsSrv = &http.Server{
				Addr:    ":8081",
				Handler: mux,
			}
			slog.Info("worker metrics server starting", "port", 8081)
			if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("metrics server failed", "err", err)
			}
		}()
	}

	safeClient := newSafeHTTPClient(cfg.Environment)

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
		defer alertsWG.Wait()

		dbTargets, err := st.GetTargets(ctx)
		if err != nil {
			slog.Error("failed to get targets from database", "err", err)
			os.Exit(1)
		}

		slog.Info("executing health pings", "count", len(dbTargets))
		runBatch(ctx, safeClient, st, nil, dbTargets)

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
		runBatch(batchCtx, safeClient, st, tracer, dbTargets)
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
		slog.Info("cron scheduler stopped")
	case <-time.After(10 * time.Second):
		slog.Warn("cron scheduler shutdown timed out")
	}

	if metricsSrv != nil {
		slog.Info("shutting down metrics server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = metricsSrv.Shutdown(shutdownCtx)
		cancel()
	}

	slog.Info("waiting for in-flight alerts to complete")
	alertsWG.Wait()
	slog.Info("worker stopped cleanly")
}

// pingTarget performs an HTTP request to the URL using custom method/headers/expected status/body contains and returns the status ("up"/"down") and latency.
func pingTarget(ctx context.Context, client *http.Client, target store.Target) (string, time.Duration) {
	start := time.Now()

	method := target.Method
	if method == "" {
		method = http.MethodGet
	}

	// Create a new HTTP request with the context (for timeout support)
	req, err := http.NewRequestWithContext(ctx, method, target.URL, nil)
	if err != nil {
		slog.Error("failed to create request", "target", target.URL, "err", err)
		return "down", time.Since(start)
	}

	// Parse and set headers
	if target.Headers != "" {
		var headersMap map[string]string
		if err := json.Unmarshal([]byte(target.Headers), &headersMap); err == nil {
			for k, v := range headersMap {
				req.Header.Set(k, v)
			}
		} else {
			slog.Warn("failed to parse headers JSON", "target", target.URL, "headers", target.Headers, "err", err)
		}
	}

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("ping request failed", "target", target.URL, "err", err)
		return "down", time.Since(start)
	}
	defer resp.Body.Close()

	// Read body for response contains check
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("failed to read response body", "target", target.URL, "err", err)
		return "down", time.Since(start)
	}

	// Verify status code
	expectedStatus := target.ExpectedStatus
	if expectedStatus <= 0 {
		expectedStatus = 200
	}

	statusMatches := false
	if expectedStatus == 200 {
		statusMatches = resp.StatusCode >= 200 && resp.StatusCode < 300
	} else {
		statusMatches = resp.StatusCode == expectedStatus
	}

	if !statusMatches {
		slog.Info("ping status mismatch", "target", target.URL, "expected", expectedStatus, "actual", resp.StatusCode)
		return "down", time.Since(start)
	}

	// Verify response body if configured
	if target.ResponseContains != "" {
		if !bytes.Contains(bodyBytes, []byte(target.ResponseContains)) {
			slog.Info("ping body search match failed", "target", target.URL, "substring", target.ResponseContains)
			return "down", time.Since(start)
		}
	}

	return "up", time.Since(start)
}

func runPingAndCheck(ctx context.Context, client *http.Client, st *store.Store, tracer trace.Tracer, target store.Target) {
	var childSpan trace.Span
	var pingCtx context.Context = ctx
	if tracer != nil {
		pingCtx, childSpan = tracer.Start(ctx, "PingTarget", trace.WithAttributes(
			attribute.String("http.url", target.URL),
			attribute.String("http.method", target.Method),
		))
	}

	runCtx, cancel := context.WithTimeout(pingCtx, 10*time.Second)
	status, latency := pingTarget(runCtx, client, target)
	cancel()

	if childSpan != nil {
		childSpan.SetAttributes(
			attribute.String("health.status", status),
			attribute.Int64("health.latency_ms", latency.Milliseconds()),
		)
	}

	// Record metrics in Prometheus
	monitor.CheckCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("target", target.URL),
		attribute.String("status", status),
	))
	monitor.LatencyHistogram.Record(ctx, latency.Seconds(), metric.WithAttributes(
		attribute.String("target", target.URL),
	))

	// Get previous status to check for transitions
	prevStatus, err := st.GetPreviousCheckStatus(ctx, target.URL)
	if err == nil && prevStatus != "" && prevStatus != status {
		// State transitioned! Alert!
		slog.Info("status transition detected, sending alert", "target", target.URL, "old_status", prevStatus, "new_status", status)
		alertsWG.Add(1)
		go func() {
			defer alertsWG.Done()
			sendWebhookAlert(context.Background(), client, target.URL, prevStatus, status, latency)
		}()
	}

	// Record the result in the database
	if err := st.InsertCheck(ctx, target.URL, status, int(latency.Milliseconds())); err != nil {
		slog.Error("failed to save check", "target", target.URL, "err", err)
		if childSpan != nil {
			childSpan.RecordError(err)
		}
	} else {
		slog.Info("check recorded", "target", target.URL, "status", status, "latency_ms", latency.Milliseconds())
	}

	if childSpan != nil {
		childSpan.End()
	}
}

func sendWebhookAlert(ctx context.Context, client *http.Client, target, oldStatus, newStatus string, latency time.Duration) {
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

	resp, err := client.Do(req)
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
