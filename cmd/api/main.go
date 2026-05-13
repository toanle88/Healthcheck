package main

import (
	"context"
	"log/slog" // Structured logging, JSON format. Built into Go 1.21+
	"net/http"
	"os"
	"os/signal" // For catching OS signals like Ctrl+C
	"syscall"   // Defines system signals like SIGTERM
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/toanle88/healthcheck/internal/config"
	"github.com/toanle88/healthcheck/internal/handler"
	"github.com/toanle88/healthcheck/internal/monitor"
	"github.com/toanle88/healthcheck/internal/store"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	// --- 1. SETUP STRUCTURED JSON LOGGING ---
	// Create a JSON logger that writes to stdout. Makes logs easy to parse in cloud platforms.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger) // Set it as the global logger so slog.Info/Error works everywhere

	// --- 2. LOAD CONFIG ---
	// Load config from env vars, .env file, etc. Likely contains Port, DatabaseURL, etc.
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- 4. INITIALIZE OPENTELEMETRY ---
	// serviceName should be unique for each microservice
	shutdown, err := monitor.InitOTel(ctx, "healthcheck-api")
	if err != nil {
		slog.Error("otel init failed", "err", err)
	} else {
		// Ensure OTel provider is shut down cleanly on exit
		defer shutdown(context.Background())
	}

	// --- 4. CONNECT TO DATABASE ---
	// Pass ctx so DB connect can be canceled if shutdown is triggered during startup
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect failed", "err", err) // Log structured error
		os.Exit(1) // Can't run without DB, so exit
	}
	defer st.Close() // Ensure DB connection is closed when main returns

	// --- 5. RUN DB MIGRATIONS / SCHEMA INIT ---
	// Create tables if they don't exist. Note: error is logged but app continues.
	// You might want to os.Exit(1) here too if schema is critical.
	if err := st.InitSchema(ctx); err != nil {
		slog.Error("schema init failed", "err", err)
	}

	// --- 6. SETUP GIN ROUTER + MIDDLEWARE ---
	r := gin.New() // gin.New() is barebones, no default middleware like gin.Default()
	r.Use(gin.Recovery()) // Recover from panics and return 500 instead of crashing
	
	// Add OTel middleware for automatic tracing of all HTTP requests
	r.Use(otelgin.Middleware("healthcheck-api"))

	// Basic CORS middleware to allow our React app (on port 5173) to talk to the API
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Custom request logging middleware
	r.Use(func(c *gin.Context) {
		start := time.Now()  // Start timer
		c.Next()             // Process the actual request
		// After request finishes, log method, path, status code, and duration
		slog.Info("request", 
			"method", c.Request.Method, 
			"path", c.Request.URL.Path, 
			"status", c.Writer.Status(), 
			"dur_ms", time.Since(start).Milliseconds(),
		)
	})

	// --- 7. INITIALIZE HANDLERS AND DEFINE ROUTES ---
	h := handler.New(st) // Pass store to handlers so they can query DB

	r.GET("/health", h.Health)         // Basic health check endpoint, usually for k8s liveness
	r.GET("/api/status", h.Status)     // Current status of services you're monitoring
	r.GET("/api/history", h.History)   // Historical status data

	// Metrics endpoint for Prometheus/Azure Monitor scraping
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// --- 8. CONFIGURE HTTP SERVER ---
	srv := &http.Server{
		Addr:    ":" + cfg.Port, // Listen on port from config, e.g. ":8080"
		Handler: r,              // Use the Gin router
	}

	// --- 9. START SERVER IN GOROUTINE ---
	// Run server in background so we don't block and can wait for shutdown signal
	go func() {
		slog.Info("api listening", "port", cfg.Port)
		// ListenAndServe always returns non-nil error. ErrServerClosed is the "good" one
		// that happens during graceful shutdown, so we ignore it
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	// --- 10. WAIT FOR SHUTDOWN SIGNAL ---
	<-ctx.Done() // Block here until Ctrl+C or SIGTERM is received
	slog.Info("shutting down")

	// --- 11. GRACEFUL SHUTDOWN WITH TIMEOUT ---
	// Give active requests 5 seconds to finish before forcing shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx) // Shutdown stops accepting new requests, waits for active ones
}