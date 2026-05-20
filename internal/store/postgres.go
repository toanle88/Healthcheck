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
	// If we are NOT in local dev (local, development, or empty), we use Azure Managed Identity tokens
	env := os.Getenv("ENV")
	if env != "local" && env != "development" && env != "" {
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
	CREATE TABLE IF NOT EXISTS targets (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT UNIQUE NOT NULL,
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	INSERT INTO targets (name, url) VALUES
		('Httpbin', 'http://httpbin.org/get'),
		('GitHub', 'https://github.com'),
		('Azure Status', 'https://azure.microsoft.com/en-us/status/')
	ON CONFLICT (url) DO NOTHING;

	CREATE TABLE IF NOT EXISTS checks (
		id SERIAL PRIMARY KEY,                    -- Auto-incrementing ID
		target TEXT NOT NULL,                     -- What URL/service we checked, like "https://google.com"
		status TEXT NOT NULL,                     -- "up" or "down" 
		latency_ms INT NOT NULL,                  -- How long the check took, in milliseconds
		checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()  -- When we ran the check, with timezone
	);

	CREATE INDEX IF NOT EXISTS idx_checks_target_checked_at ON checks(target, checked_at DESC);
	CREATE INDEX IF NOT EXISTS idx_checks_checked_at_target ON checks(checked_at DESC, target);
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
	UptimeSLA float64   `json:"uptime_sla"`
}

// Target represents a monitored URL/endpoint.
type Target struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TargetSLA represents calculated uptime percentage.
type TargetSLA struct {
	Target           string  `json:"target"`
	UptimePercentage float64 `json:"uptime_percentage"`
}

// GetLatestChecks retrieves the most recent check result for each active target, including 24h SLA.
func (s *Store) GetLatestChecks(ctx context.Context) ([]Check, error) {
	rows, err := s.DB.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (target) target, status, latency_ms, checked_at 
			FROM checks 
			ORDER BY target, checked_at DESC
		),
		sla AS (
			SELECT 
				target,
				ROUND(100.0 * COUNT(CASE WHEN status = 'up' THEN 1 END) / COUNT(*), 2) as uptime_percentage
			FROM checks
			WHERE checked_at >= NOW() - INTERVAL '24 hours'
			GROUP BY target
		)
		SELECT 
			t.url as target, 
			COALESCE(l.status, 'pending') as status, 
			COALESCE(l.latency_ms, 0) as latency_ms, 
			COALESCE(l.checked_at, t.created_at) as checked_at,
			COALESCE(s.uptime_percentage, 100.0) as uptime_sla
		FROM targets t
		LEFT JOIN latest l ON t.url = l.target
		LEFT JOIN sla s ON t.url = s.target
		WHERE t.is_active = TRUE
		ORDER BY t.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var ck Check
		if err := rows.Scan(&ck.Target, &ck.Status, &ck.LatencyMs, &ck.CheckedAt, &ck.UptimeSLA); err != nil {
			return nil, err
		}
		checks = append(checks, ck)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating latest checks: %w", err)
	}
	return checks, nil
}

// GetTargets retrieves all monitored targets.
func (s *Store) GetTargets(ctx context.Context) ([]Target, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, name, url, is_active, created_at, updated_at 
		FROM targets 
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []Target
	for rows.Next() {
		var t Target
		if err := rows.Scan(&t.ID, &t.Name, &t.URL, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating targets: %w", err)
	}
	return targets, nil
}

// InsertTarget saves a new target.
func (s *Store) InsertTarget(ctx context.Context, name, url string) (Target, error) {
	var t Target
	err := s.DB.QueryRow(ctx, `
		INSERT INTO targets (name, url) 
		VALUES ($1, $2)
		RETURNING id, name, url, is_active, created_at, updated_at
	`, name, url).Scan(&t.ID, &t.Name, &t.URL, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

// DeleteTarget removes a target and its associated check history.
func (s *Store) DeleteTarget(ctx context.Context, id int) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var url string
	err = tx.QueryRow(ctx, "DELETE FROM targets WHERE id = $1 RETURNING url", id).Scan(&url)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "DELETE FROM checks WHERE target = $1", url)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// GetHistoricalChecks retrieves the last N checks for a target in chronological order.
func (s *Store) GetHistoricalChecks(ctx context.Context, target string, limit int) ([]Check, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT target, status, latency_ms, checked_at 
		FROM checks 
		WHERE target = $1 
		ORDER BY checked_at DESC 
		LIMIT $2
	`, target, limit)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating historical checks: %w", err)
	}

	// Reverse to return in chronological order
	for i, j := 0, len(checks)-1; i < j; i, j = i+1, j-1 {
		checks[i], checks[j] = checks[j], checks[i]
	}
	return checks, nil
}

// GetPreviousCheckStatus retrieves the status of the most recent check for a target.
func (s *Store) GetPreviousCheckStatus(ctx context.Context, target string) (string, error) {
	var status string
	err := s.DB.QueryRow(ctx, `
		SELECT status 
		FROM checks 
		WHERE target = $1 
		ORDER BY checked_at DESC 
		LIMIT 1
	`, target).Scan(&status)
	return status, err
}

// CleanupOldChecks deletes records older than the specified duration.
func (s *Store) CleanupOldChecks(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.DB.Exec(ctx, "DELETE FROM checks WHERE checked_at < $1", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
