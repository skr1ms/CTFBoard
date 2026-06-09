package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	redisModule "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultPostgresUser           = "user"
	defaultPostgresPassword       = "password"
	defaultPostgresDB             = "test"
	defaultPostgresImage          = "postgres:18.4-alpine3.23"
	defaultRedisImage             = "redis:8.8.0-alpine3.23"
	defaultPostgresStartupTimeout = 60 * time.Second
	defaultPostgresReadyLogs      = 2
)

// PostgresOption is a functional option for configuring the test Postgres container.
type PostgresOption func(*postgresOpts)

type postgresOpts struct {
	database string
	user     string
	password string
	timeout  time.Duration
	cmd      []string
}

// PostgresWithDatabase overrides the default test database name.
func PostgresWithDatabase(db string) PostgresOption {
	return func(o *postgresOpts) { o.database = db }
}

// PostgresWithUser overrides the default test database user.
func PostgresWithUser(user string) PostgresOption {
	return func(o *postgresOpts) { o.user = user }
}

// PostgresWithPassword overrides the default test database password.
func PostgresWithPassword(pass string) PostgresOption {
	return func(o *postgresOpts) { o.password = pass }
}

// PostgresWithStartupTimeout overrides the default container startup timeout.
func PostgresWithStartupTimeout(d time.Duration) PostgresOption {
	return func(o *postgresOpts) { o.timeout = d }
}

// PostgresWithCmd appends extra command-line arguments to the postgres server process.
func PostgresWithCmd(cmd ...string) PostgresOption {
	return func(o *postgresOpts) { o.cmd = cmd }
}

// StartPostgres starts a throwaway Postgres container using testcontainers and returns the container,
// a ready-to-use connection string, and any startup error.
func StartPostgres(ctx context.Context, opts ...PostgresOption) (*postgres.PostgresContainer, string, error) {
	o := &postgresOpts{
		database: defaultPostgresDB,
		user:     defaultPostgresUser,
		password: defaultPostgresPassword,
		timeout:  defaultPostgresStartupTimeout,
	}

	for _, f := range opts {
		f(o)
	}

	runOpts := []testcontainers.ContainerCustomizer{
		postgres.WithDatabase(o.database),
		postgres.WithUsername(o.user),
		postgres.WithPassword(o.password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(defaultPostgresReadyLogs).
				WithStartupTimeout(o.timeout),
		),
	}
	if len(o.cmd) > 0 {
		runOpts = append(runOpts, testcontainers.WithCmd(o.cmd...))
	}

	container, err := postgres.Run(ctx, defaultPostgresImage, runOpts...)
	if err != nil {
		return nil, "", fmt.Errorf("StartPostgres - Run: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", fmt.Errorf("StartPostgres - ConnectionString: %w", err)
	}

	return container, connStr, nil
}

// StartRedis starts a throwaway Redis container and returns its connection URI, a cleanup function,
// and any startup error.
func StartRedis(ctx context.Context) (string, func(), error) {
	redisC, err := redisModule.Run(ctx, defaultRedisImage)
	if err != nil {
		return "", nil, fmt.Errorf("StartRedis - Run: %w", err)
	}

	redisURI, err := redisC.ConnectionString(ctx)
	if err != nil {
		termErr := redisC.Terminate(ctx)
		if termErr != nil {
			fmt.Printf("redis terminate: %v\n", termErr)
		}

		return "", nil, fmt.Errorf("StartRedis - ConnectionString: %w", err)
	}

	cleanup := func() {
		err := redisC.Terminate(ctx)
		if err != nil {
			fmt.Printf("redis terminate: %v\n", err)
		}
	}

	return redisURI, cleanup, nil
}

// StartRedisClient starts a throwaway Redis container, connects a client, verifies connectivity with
// PING, and returns the client, a cleanup function that closes both the client and container, and any error.
func StartRedisClient(ctx context.Context) (*redis.Client, func(), error) {
	redisURI, cleanup, err := StartRedis(ctx)
	if err != nil {
		return nil, nil, err
	}

	opts, err := redis.ParseURL(redisURI)
	if err != nil {
		cleanup()

		return nil, nil, fmt.Errorf("StartRedisClient - ParseURL: %w", err)
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()

		cleanup()

		return nil, nil, fmt.Errorf("StartRedisClient - Ping: %w", err)
	}

	fullCleanup := func() {
		_ = client.Close()

		cleanup()
	}

	return client, fullCleanup, nil
}
