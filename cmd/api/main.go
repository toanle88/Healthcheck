package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/toanle88/healthcheck/internal/config"
	"github.com/toanle88/healthcheck/internal/handler"
	"github.com/toanle88/healthcheck/internal/middleware"
	"github.com/toanle88/healthcheck/internal/monitor"
	"github.com/toanle88/healthcheck/internal/store"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

var Version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Info("starting healthcheck api v2 - THE GREEN VERSION", "version", Version)

	cfg := config.Load()

	metricsHandler, shutdown, err := monitor.InitOTel(ctx, "healthcheck-api")
	if err != nil {
		slog.Error("otel init failed", "err", err)
	} else {
		defer shutdown(context.Background())
	}

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := st.InitSchema(ctx); err != nil {
		slog.Error("schema init failed", "err", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("healthcheck-api"))

	// --- CORS MIDDLEWARE ---
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			isAllowed := false
			// Default local development origins
			if cfg.Environment == "local" || cfg.Environment == "development" || cfg.Environment == "" {
				if origin == "http://localhost:5173" || origin == "http://localhost:3000" {
					isAllowed = true
				}
			}
			for _, allowed := range cfg.CORSAllowedOrigins {
				if origin == allowed {
					isAllowed = true
					break
				}
			}

			if isAllowed {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	h := handler.New(st)

	// --- PUBLIC ROUTES ---
	r.GET("/health", h.Health)

	// --- PROTECTED ROUTES ---
	api := r.Group("/api")
	if cfg.EntraTenantID != "" && cfg.EntraClientID != "" {
		slog.Info("enabling Entra ID authentication", "tenantID", cfg.EntraTenantID)
		api.Use(middleware.AuthMiddleware(cfg.EntraTenantID, cfg.EntraClientID))
	} else {
		slog.Warn("Entra ID configuration missing, running without authentication")
	}

	{
		api.GET("/status", h.Status)
		api.GET("/history", h.History)
		api.GET("/targets", h.GetTargets)
		api.POST("/targets", h.CreateTarget)
		api.DELETE("/targets/:id", h.DeleteTarget)
	}

	// --- TEST ROUTES (UNPROTECTED FOR CHAOS TESTING) ---
	test := r.Group("/api/test")
	{
		test.GET("/error", h.TestError)
		test.GET("/slow", h.TestSlow)
	}

	if metricsHandler != nil {
		r.GET("/metrics", gin.WrapH(metricsHandler))
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		slog.Info("api listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
