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

// @title Healthcheck API
// @version 2.0.0
// @description API documentation for the Healthcheck Dashboard, a DevOps playground application monitoring public APIs.
// @host localhost:8080
// @BasePath /

// @securityDefinitions.oauth2.implicit EntraID
// @authorizationurl https://login.microsoftonline.com/common/oauth2/v2.0/authorize
// @scope.api://default/access_as_user Access Healthcheck API as user

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer <token>" to authenticate.
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

	broker := handler.NewBroker()
	go broker.Start(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// We need a dedicated connection for LISTEN/NOTIFY.
			conn, err := st.DB.Acquire(ctx)
			if err != nil {
				slog.Error("failed to acquire connection for LISTEN", "err", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
					continue
				}
			}

			_, err = conn.Exec(ctx, "LISTEN checks_channel")
			if err != nil {
				slog.Error("failed to listen on checks_channel", "err", err)
				conn.Release()
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
					continue
				}
			}

			slog.Info("successfully subscribed to postgres checks_channel")

			for {
				// WaitForNotification blocks until a notification is received or connection fails
				_, err := conn.Conn().WaitForNotification(ctx)
				if err != nil {
					slog.Warn("postgres notification connection lost, reconnecting...", "err", err)
					break
				}

				// Query the latest status of all targets
				checks, err := st.GetLatestChecks(ctx)
				if err != nil {
					slog.Error("failed to fetch latest checks after notification", "err", err)
					continue
				}

				// Broadcast to all connected SSE clients
				broker.Broadcast(checks)
			}

			conn.Release()
		}
	}()

	h := handler.New(st, broker)

	// --- PUBLIC ROUTES ---
	r.GET("/health", h.Health)
	r.GET("/openapi.json", h.OpenAPISpec)
	r.GET("/docs", h.Docs)

	// --- PROTECTED ROUTES ---
	api := r.Group("/api")
	isLocalDev := cfg.Environment == "local" || cfg.Environment == "development" || cfg.Environment == ""
	if cfg.EntraTenantID != "" && cfg.EntraClientID != "" {
		slog.Info("enabling Entra ID authentication", "tenantID", cfg.EntraTenantID)
		api.Use(middleware.AuthMiddleware(cfg.EntraTenantID, cfg.EntraClientID, cfg.Environment))
	} else {
		if !isLocalDev {
			slog.Error("Entra ID configuration missing in non-development environment! Failing closed.")
			os.Exit(1)
		}
		slog.Warn("Entra ID configuration missing, running without authentication (development mode only)")
	}

	{
		api.GET("/status", h.Status)
		api.GET("/status/stream", h.StreamStatus)
		api.GET("/history", h.History)
		api.GET("/targets", h.GetTargets)

		adminRequired := middleware.RequireRoleOrScope([]string{"Healthcheck.Admin"}, nil)
		api.POST("/targets", adminRequired, h.CreateTarget)
		api.DELETE("/targets/:id", adminRequired, h.DeleteTarget)
	}

	// --- TEST ROUTES (UNPROTECTED FOR CHAOS TESTING) ---
	if isLocalDev {
		test := r.Group("/api/test")
		{
			test.GET("/error", h.TestError)
			test.GET("/slow", h.TestSlow)
		}
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
