package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/toanle88/healthcheck/internal/config"
	"github.com/toanle88/healthcheck/internal/migrations"
	"github.com/toanle88/healthcheck/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Info("starting database migrations")

	cfg := config.Load()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	// Convert *pgxpool.Pool to *sql.DB
	db := stdlib.OpenDBFromPool(st.DB)
	defer db.Close()

	// Initialize the pgx/v5 driver for golang-migrate
	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		slog.Error("failed to create migration driver", "err", err)
		os.Exit(1)
	}

	// Initialize source driver using embedded FS
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		slog.Error("failed to create source driver", "err", err)
		os.Exit(1)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance(
		"iofs", sourceDriver,
		"postgres", driver,
	)
	if err != nil {
		slog.Error("failed to initialize migrate", "err", err)
		os.Exit(1)
	}

	// Run migration up
	slog.Info("applying database migrations...")
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			slog.Info("database is up to date, no migrations applied")
			os.Exit(0)
		}
		slog.Error("failed to run migrations", "err", err)
		os.Exit(1)
	}

	slog.Info("migrations applied successfully")
}
