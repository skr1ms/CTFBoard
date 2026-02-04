package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/config"
)

const maxPoolSize = 10

func New(cfg *config.DB) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("postgres - New - pgxpool.ParseConfig: %w", err)
	}
	config.MaxConns = maxPoolSize

	var pool *pgxpool.Pool
	operation := func() error {
		var createErr error
		pool, createErr = pgxpool.NewWithConfig(context.Background(), config)
		if createErr != nil {
			return createErr
		}
		if pingErr := pool.Ping(context.Background()); pingErr != nil {
			pool.Close()
			pool = nil
			return pingErr
		}
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 30 * time.Second
	if err := backoff.Retry(operation, bo); err != nil {
		return nil, fmt.Errorf("postgres - New: %w", err)
	}

	return pool, nil
}
