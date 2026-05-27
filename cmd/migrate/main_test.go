package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	var pgContainer *postgres.PostgresContainer
	var err error

	if dbURL == "" {
		ctx := context.Background()
		pgContainer, err = postgres.Run(ctx,
			"postgres:18-alpine",
			postgres.WithDatabase("healthcheck"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("postgres"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second),
			),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to start postgres container: %v", err))
		}

		dbURL, err = pgContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			panic(fmt.Sprintf("failed to get connection string: %v", err))
		}

		os.Setenv("DATABASE_URL_TEST", dbURL)
	}

	code := m.Run()

	if pgContainer != nil {
		ctx := context.Background()
		if err := pgContainer.Terminate(ctx); err != nil {
			panic(fmt.Sprintf("failed to terminate container: %v", err))
		}
	}

	os.Exit(code)
}

func TestRunMigrate(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL_TEST not set")
	}

	ctx := context.Background()

	// 1. Success case
	err := runMigrate(ctx, dbURL)
	if err != nil {
		t.Errorf("expected no error from runMigrate, got: %v", err)
	}

	// 2. Failure case (bad connection string)
	err = runMigrate(ctx, "postgres://invalid:invalid@localhost:5432/invalid?sslmode=disable")
	if err == nil {
		t.Errorf("expected error from runMigrate with invalid DB URL, got nil")
	}
}
