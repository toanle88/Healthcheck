package store

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const notifyChecksChannel = "NOTIFY checks_channel"

// InsertTargetParams represents the parameters for creating a new target.
type InsertTargetParams struct {
	Name             string
	URL              string
	Method           string
	Headers          string
	ExpectedStatus   int
	ResponseContains string
	FailureThreshold int
}

// Store holds all database connections and methods.
type Store struct {
	DB             *pgxpool.Pool
	slaCache       map[string]float64
	slaCacheExpiry time.Time
	slaMutex       sync.Mutex
}

// New creates and configures a new postgres database connection pool with hybrid Azure AD auth support.
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

// InsertCheck saves a new health check result into the database.
// This is used by the worker to record the status of various targets.
func (s *Store) InsertCheck(ctx context.Context, target, status string, latencyMs int) error {
	s.slaMutex.Lock()
	s.slaCacheExpiry = time.Time{}
	s.slaMutex.Unlock()

	// We use $1, $2, $3 as placeholders for parameters to prevent SQL injection.
	// pgx handles the mapping of Go types to Postgres types.
	_, err := s.DB.Exec(ctx, `
		INSERT INTO checks (target, status, latency_ms)
		VALUES ($1, $2, $3)
	`, target, status, latencyMs)

	if err == nil {
		_, _ = s.DB.Exec(ctx, notifyChecksChannel)
	}

	return err
}

// Check represents a single health check record in the database.
type Check struct {
	Name                string    `json:"name"`
	Target              string    `json:"target"`
	Status              string    `json:"status"`
	LatencyMs           int       `json:"latency_ms"`
	CheckedAt           time.Time `json:"checked_at"`
	UptimeSLA           float64   `json:"uptime_sla"`
	FailureThreshold    int       `json:"failure_threshold"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastAlertStatus     string    `json:"last_alert_status"`
}

// Target represents a monitored URL/endpoint.
type Target struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	URL                 string    `json:"url"`
	Method              string    `json:"method"`
	Headers             string    `json:"headers"`
	ExpectedStatus      int       `json:"expected_status"`
	ResponseContains    string    `json:"response_contains"`
	FailureThreshold    int       `json:"failure_threshold"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastAlertStatus     string    `json:"last_alert_status"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// getSLACache retrieves cached target SLA percentages or computes them from database records if expired.
func (s *Store) getSLACache(ctx context.Context) (map[string]float64, error) {
	s.slaMutex.Lock()
	defer s.slaMutex.Unlock()

	if s.slaCache != nil && time.Now().Before(s.slaCacheExpiry) {
		return s.slaCache, nil
	}

	rows, err := s.DB.Query(ctx, `
		SELECT 
			target,
			ROUND(100.0 * COUNT(CASE WHEN status = 'up' THEN 1 END) / COUNT(*), 2) as uptime_percentage
		FROM checks
		WHERE checked_at >= NOW() - INTERVAL '24 hours'
		GROUP BY target
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query SLA percentages: %w", err)
	}
	defer rows.Close()

	cache := make(map[string]float64)
	for rows.Next() {
		var target string
		var percentage float64
		if err := rows.Scan(&target, &percentage); err != nil {
			return nil, fmt.Errorf("failed to scan SLA percentage: %w", err)
		}
		cache[target] = percentage
	}

	s.slaCache = cache
	s.slaCacheExpiry = time.Now().Add(10 * time.Second)
	return cache, nil
}

// GetLatestChecks retrieves the most recent check result for each active target, including 24h SLA.
func (s *Store) GetLatestChecks(ctx context.Context) ([]Check, error) {
	// 1. Get SLA cache
	cache, err := s.getSLACache(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Fetch the latest status for active targets (fast query)
	rows, err := s.DB.Query(ctx, `
		SELECT 
			t.name as name,
			t.url as target, 
			COALESCE(l.status, 'pending') as status, 
			COALESCE(l.latency_ms, 0) as latency_ms, 
			COALESCE(l.checked_at, t.created_at) as checked_at,
			t.failure_threshold,
			t.consecutive_failures,
			t.last_alert_status
		FROM targets t
		LEFT JOIN LATERAL (
			SELECT status, latency_ms, checked_at 
			FROM checks 
			WHERE target = t.url 
			ORDER BY checked_at DESC 
			LIMIT 1
		) l ON TRUE
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
		if err := rows.Scan(
			&ck.Name, &ck.Target, &ck.Status, &ck.LatencyMs, &ck.CheckedAt,
			&ck.FailureThreshold, &ck.ConsecutiveFailures, &ck.LastAlertStatus,
		); err != nil {
			return nil, err
		}
		// Look up SLA from the immutable cache copy
		slaVal, exists := cache[ck.Target]
		if exists {
			ck.UptimeSLA = slaVal
		} else {
			ck.UptimeSLA = 100.0 // Default to 100% for new targets
		}
		checks = append(checks, ck)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating latest checks: %w", err)
	}
	return checks, nil
}

// GetTargets queries all monitored target endpoints from the targets database table.
func (s *Store) GetTargets(ctx context.Context) ([]Target, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, name, url, method, headers, expected_status, response_contains, failure_threshold, consecutive_failures, last_alert_status, is_active, created_at, updated_at 
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
		var headersPtr, responseContainsPtr *string
		if err := rows.Scan(
			&t.ID, &t.Name, &t.URL, &t.Method, &headersPtr, &t.ExpectedStatus, &responseContainsPtr,
			&t.FailureThreshold, &t.ConsecutiveFailures, &t.LastAlertStatus,
			&t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if headersPtr != nil {
			t.Headers = *headersPtr
		}
		if responseContainsPtr != nil {
			t.ResponseContains = *responseContainsPtr
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating targets: %w", err)
	}
	return targets, nil
}

// InsertTarget adds a new monitoring target config into the targets database table.
func (s *Store) InsertTarget(ctx context.Context, params InsertTargetParams) (Target, error) {
	var t Target

	var headersVal *string
	if params.Headers != "" {
		headersVal = &params.Headers
	}
	var responseContainsVal *string
	if params.ResponseContains != "" {
		responseContainsVal = &params.ResponseContains
	}

	err := s.DB.QueryRow(ctx, `
		INSERT INTO targets (name, url, method, headers, expected_status, response_contains, failure_threshold) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, url, method, COALESCE(headers, ''), expected_status, COALESCE(response_contains, ''), failure_threshold, consecutive_failures, last_alert_status, is_active, created_at, updated_at
	`, params.Name, params.URL, params.Method, headersVal, params.ExpectedStatus, responseContainsVal, params.FailureThreshold).Scan(
		&t.ID, &t.Name, &t.URL, &t.Method, &t.Headers, &t.ExpectedStatus, &t.ResponseContains, &t.FailureThreshold, &t.ConsecutiveFailures, &t.LastAlertStatus, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == nil {
		_, _ = s.DB.Exec(ctx, notifyChecksChannel)
	}
	return t, err
}

// DeleteTarget removes a target and its check logs inside a database transaction.
func (s *Store) DeleteTarget(ctx context.Context, id int) error {
	s.slaMutex.Lock()
	s.slaCacheExpiry = time.Time{}
	s.slaMutex.Unlock()

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
	_, _ = s.DB.Exec(ctx, notifyChecksChannel)
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

// GetPreviousCheckStatus retrieves the status of the single most recent check execution for a target.
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

// CleanupOldChecks deletes all health check logs older than the specified duration.
func (s *Store) CleanupOldChecks(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.DB.Exec(ctx, "DELETE FROM checks WHERE checked_at < $1", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// UpdateTargetAlertState updates consecutive failures and determines if a status transition alert should trigger.
func (s *Store) UpdateTargetAlertState(ctx context.Context, url string, currentStatus string) (bool, string, string, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return false, "", "", err
	}
	defer tx.Rollback(ctx)

	var consecutiveFailures int
	var lastAlertStatus string
	var failureThreshold int

	err = tx.QueryRow(ctx, `
		SELECT consecutive_failures, last_alert_status, failure_threshold 
		FROM targets 
		WHERE url = $1
	`, url).Scan(&consecutiveFailures, &lastAlertStatus, &failureThreshold)
	if err != nil {
		return false, "", "", err
	}

	shouldAlert := false
	oldAlertStatus := lastAlertStatus
	newAlertStatus := lastAlertStatus

	if currentStatus == "up" {
		consecutiveFailures = 0
		if lastAlertStatus == "down" {
			shouldAlert = true
			newAlertStatus = "up"
			lastAlertStatus = "up"
		}
	} else { // "down"
		consecutiveFailures++
		if consecutiveFailures >= failureThreshold {
			if lastAlertStatus == "up" {
				shouldAlert = true
				newAlertStatus = "down"
				lastAlertStatus = "down"
			}
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE targets 
		SET consecutive_failures = $1, last_alert_status = $2 
		WHERE url = $3
	`, consecutiveFailures, lastAlertStatus, url)
	if err != nil {
		return false, "", "", err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return false, "", "", err
	}

	return shouldAlert, oldAlertStatus, newAlertStatus, nil
}
