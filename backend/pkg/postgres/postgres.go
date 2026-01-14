package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/config"
)

const (
	maxPoolSize  = 10
	connAttempts = 10
	connTimeout  = time.Second
)

func New(cfg *config.DB) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("postgres - New - pgxpool.ParseConfig: %w", err)
	}

	config.MaxConns = maxPoolSize

	var pool *pgxpool.Pool

	for i := 0; i < connAttempts; i++ {
		pool, err = pgxpool.NewWithConfig(context.Background(), config)
		if err == nil {
			break
		}

		time.Sleep(connTimeout)
	}

	if err != nil {
		return nil, fmt.Errorf("postgres - New - pgxpool.NewWithConfig: %w", err)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("postgres - New - pool.Ping: %w", err)
	}

	return pool, nil
}
