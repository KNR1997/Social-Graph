// Package postgres builds a pgx connection pool. No ORM, no query builder: the
// SQL is written by hand in db/queries and compiled by sqlc.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kethaka-creskit/go-ddd-service/config"
)

// Backoff bounds for the readiness loop.
const (
	initialBackoff = 100 * time.Millisecond
	maxBackoff     = 2 * time.Second
)

// New opens and verifies a pool. It returns an error instead of panicking, and
// holds no package-level state, so a test can stand up as many pools as it
// likes without fighting a singleton.
func New(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DB.URL)
	if err != nil {
		return nil, fmt.Errorf("postgres - New - ParseConfig: %w", err)
	}

	poolCfg.MaxConns = cfg.DB.MaxConns
	poolCfg.MinConns = cfg.DB.MinConns
	poolCfg.MaxConnLifetime = cfg.DB.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.DB.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = cfg.DB.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres - New - NewWithConfig: %w", err)
	}

	if err := ping(ctx, pool, cfg.DB.ConnectAttempts); err != nil {
		pool.Close()

		return nil, err
	}

	return pool, nil
}

// ping waits for the database to accept connections, backing off between
// attempts. Containers and managed instances are routinely not ready yet when
// the process starts.
func ping(ctx context.Context, pool *pgxpool.Pool, attempts int) error {
	if attempts < 1 {
		attempts = 1
	}

	backoff := initialBackoff

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		lastErr = pool.Ping(ctx)
		if lastErr == nil {
			return nil
		}

		if ctx.Err() != nil {
			return fmt.Errorf("postgres - ping - context done after %d attempts: %w", attempt, ctx.Err())
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("postgres - ping: %w", ctx.Err())
		case <-time.After(backoff):
		}

		if backoff < maxBackoff {
			backoff *= 2
		}
	}

	return fmt.Errorf("postgres - ping - unreachable after %d attempts: %w", attempts, lastErr)
}
