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

	if err := runMigrate(ctx, cfg.DatabaseURL); err != nil {
		slog.Error("failed to run migrations", "err", err)
		os.Exit(1)
	}

	slog.Info("migrations applied successfully")
}

func runMigrate(ctx context.Context, dbURL string) error {
	st, err := store.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer st.Close()

	// Convert *pgxpool.Pool to *sql.DB
	db := stdlib.OpenDBFromPool(st.DB)
	defer db.Close()

	// Initialize the pgx/v5 driver for golang-migrate
	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		return err
	}

	// Initialize source driver using embedded FS
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance(
		"iofs", sourceDriver,
		"postgres", driver,
	)
	if err != nil {
		return err
	}
	defer m.Close()

	// Run migration up
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}
