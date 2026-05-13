package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool" // pgx is a Postgres driver. pgxpool = connection pooling
)

// Store holds all database connections and methods.
// This is the "database layer" that your handlers will use.
type Store struct {
	DB *pgxpool.Pool // Pool = a collection of reusable DB connections. Way faster than opening new ones each time
}

// New creates a new database connection pool.
// This gets called once when your app starts up.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	// Wrap the parent context with a 5 second timeout.
	// If DB doesn't connect in 5s, we give up instead of hanging forever.
	// defer cancel() ensures we clean up the timeout when function returns.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Parse the connection string like "postgres://user:pass@host:5432/dbname"
	// into a config struct pgx can understand
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err // Return error if URL is malformed
	}
	
	// Tune the connection pool. These are conservative defaults for small apps.
	cfg.MaxConns = 5 // Max 5 connections open at once. Prevents overwhelming DB
	cfg.MinConns = 1 // Keep at least 1 connection warm so first request isn't slow

	// Create the actual pool using our config
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err // DB might be down, wrong password, etc
	}

	// Ping = send a simple "are you alive?" query to DB.
	// This verifies the connection actually works before we return.
	// If ping fails, close the pool so we don't leak connections.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	// Success: wrap the pool in our Store struct and return it
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