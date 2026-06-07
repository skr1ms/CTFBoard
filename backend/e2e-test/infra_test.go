package e2e_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/testutil"
)

// setupInfrastructure starts DB and Redis (testcontainers or external) and returns cleanup.
func setupInfrastructure(ctx context.Context) (func(), error) {
	if os.Getenv("USE_EXTERNAL_DB") == "true" {
		fmt.Println("Using EXTERNAL infrastructure (CI mode)...")

		return setupExternalInfra(ctx)
	}

	fmt.Println("Using TESTCONTAINERS infrastructure...")

	return setupTestContainers(ctx)
}

func setupTestContainers(ctx context.Context) (func(), error) {
	postgresC, connStr, err := testutil.StartPostgres(ctx,
		testutil.PostgresWithUser(string(domain.RoleUser)),
		testutil.PostgresWithPassword("password"),
		testutil.PostgresWithStartupTimeout(120*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	e2eConnStr = connStr

	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	poolCfg.MaxConns = 20

	TestPool, err = pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := TestPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	var redisCleanup func()

	TestRedis, redisCleanup, err = testutil.StartRedisClient(ctx)
	if err != nil {
		termErr := postgresC.Terminate(ctx)
		if termErr != nil {
			fmt.Printf("postgres terminate on cleanup: %v\n", termErr)
		}

		return nil, fmt.Errorf("failed to start redis: %w", err)
	}

	cleanup := func() {
		fmt.Println("Cleaning up containers...")
		TestPool.Close()
		redisCleanup()

		err := postgresC.Terminate(ctx)
		if err != nil {
			fmt.Printf("postgres terminate: %v\n", err)
		}
	}

	return cleanup, nil
}

// setupExternalInfra connects to existing Postgres and Redis from env (CI) and returns cleanup.
func setupExternalInfra(ctx context.Context) (func(), error) {
	// PostgreSQL Setup
	dbUser := testutil.GetEnv("POSTGRES_USER", "test_user")
	dbPass := testutil.GetEnv("POSTGRES_PASSWORD", "test_password")
	dbHost := testutil.GetEnv("POSTGRES_HOST", "postgres")
	dbPort := testutil.GetEnv("POSTGRES_PORT", "5432")
	dbName := testutil.GetEnv("POSTGRES_DB", "test_board")

	e2eConnStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)

	poolCfg, err := pgxpool.ParseConfig(e2eConnStr)
	if err != nil {
		return nil, err
	}

	poolCfg.MaxConns = 20

	TestPool, err = pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	bo := backoff.NewExponentialBackOff()

	bo.MaxElapsedTime = 15 * time.Second
	if err := backoff.Retry(func() error { return TestPool.Ping(ctx) }, backoff.WithContext(bo, ctx)); err != nil {
		return nil, fmt.Errorf("external db ping failed: %w", err)
	}

	// Redis Setup
	redisHost := testutil.GetEnv("REDIS_HOST", "redis")
	redisPort := testutil.GetEnv("REDIS_PORT", "6379")
	redisPassword := testutil.GetEnv("REDIS_PASSWORD", "")

	TestRedis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
	})

	if err := TestRedis.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("external redis ping failed: %w", err)
	}

	return func() {
		TestPool.Close()
		_ = TestRedis.Close()
	}, nil
}

// Server setup
