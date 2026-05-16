package store

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store holds all database connections and methods.
type Store struct {
	DB *pgxpool.Pool
}

// New creates a new database connection pool with hybrid auth support.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// HYBRID AUTH LOGIC
	// If we are NOT in local dev, we use Azure Managed Identity tokens
	if os.Getenv("ENV") != "local" && os.Getenv("ENV") != "" {
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create azure credential: %w", err)
		}

		// This hook runs every time a NEW connection is opened in the pool.
		// It fetches a fresh token so we never have to worry about expiry.
		cfg.BeforeConnect = func(ctx context.Context, pgc *pgx.ConnConfig) error {
			token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
				Scopes: []string{"https://ossrdbms-aad.database.windows.net/.default"},
			})
			if err != nil {
				return fmt.Errorf("failed to get azure ad token: %w", err)
			}
			pgc.Password = token.Token
			return nil
		}
	}

	cfg.MaxConns = 10
	cfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &Store{DB: pool}, nil
}

// Close shuts down all connections in the pool gracefully.
// You call this with defer s.Close() in main.go so it always runs on shutdown.
func (s *Store) Close() {
	if s.DB != nil {
		s.DB.Close() // Waits for active queries to finish, then closes
	}
}

// InitSchema creates tables if they don't exist yet.
// This is "auto-migration" for Day 1. In production you'd use a real migration tool.
func (s *Store) InitSchema(ctx context.Context) error {
	// Exec runs SQL that doesn't return rows, like CREATE TABLE
	_, err := s.DB.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS checks (
		id SERIAL PRIMARY KEY,                    -- Auto-incrementing ID
		target TEXT NOT NULL,                     -- What URL/service we checked, like "https://google.com"
		status TEXT NOT NULL,                     -- "up" or "down" 
		latency_ms INT NOT NULL,                  -- How long the check took, in milliseconds
		checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()  -- When we ran the check, with timezone
	);
	`)
	return err // Return nil if table created/exists, or error if SQL failed
}

// InsertCheck saves a new health check result into the database.
// This is used by the worker to record the status of various targets.
func (s *Store) InsertCheck(ctx context.Context, target, status string, latencyMs int) error {
	// We use $1, $2, $3 as placeholders for parameters to prevent SQL injection.
	// pgx handles the mapping of Go types to Postgres types.
	_, err := s.DB.Exec(ctx, `
		INSERT INTO checks (target, status, latency_ms)
		VALUES ($1, $2, $3)
	`, target, status, latencyMs)

	return err
}

// Check represents a single health check record in the database.
type Check struct {
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	LatencyMs int       `json:"latency_ms"`
	CheckedAt time.Time `json:"checked_at"`
}

// GetLatestChecks retrieves the most recent check result for each unique target.
// It uses Postgres' DISTINCT ON feature to efficiently group by target.
func (s *Store) GetLatestChecks(ctx context.Context) ([]Check, error) {
	// DISTINCT ON (target) ensures we only get one row per target.
	// We order by target first (required by DISTINCT ON) and then by checked_at DESC
	// to make sure we get the newest one.
	rows, err := s.DB.Query(ctx, `
		SELECT DISTINCT ON (target) target, status, latency_ms, checked_at 
		FROM checks 
		ORDER BY target, checked_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var ck Check
		if err := rows.Scan(&ck.Target, &ck.Status, &ck.LatencyMs, &ck.CheckedAt); err != nil {
			return nil, err
		}
		checks = append(checks, ck)
	}
	return checks, nil
}

// CleanupOldChecks deletes records older than the specified duration.
// This prevents the database from growing indefinitely.
func (s *Store) CleanupOldChecks(ctx context.Context, olderThan time.Duration) (int64, error) {
	// Calculate the cutoff time
	cutoff := time.Now().Add(-olderThan)

	// Exec the DELETE statement
	result, err := s.DB.Exec(ctx, "DELETE FROM checks WHERE checked_at < $1", cutoff)
	if err != nil {
		return 0, err
	}

	// RowsAffected() tells us how many rows were cleaned up
	return result.RowsAffected(), nil
}
