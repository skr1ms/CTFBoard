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
	defaultPostgresStartupTimeout = 60 * time.Second
)

type PostgresOption func(*postgresOpts)

type postgresOpts struct {
	database string
	user     string
	password string
	timeout  time.Duration
	cmd      []string
}

func PostgresWithDatabase(db string) PostgresOption {
	return func(o *postgresOpts) { o.database = db }
}

func PostgresWithUser(user string) PostgresOption {
	return func(o *postgresOpts) { o.user = user }
}

func PostgresWithPassword(pass string) PostgresOption {
	return func(o *postgresOpts) { o.password = pass }
}

func PostgresWithStartupTimeout(d time.Duration) PostgresOption {
	return func(o *postgresOpts) { o.timeout = d }
}

func PostgresWithCmd(cmd ...string) PostgresOption {
	return func(o *postgresOpts) { o.cmd = cmd }
}

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
				WithOccurrence(2).
				WithStartupTimeout(o.timeout),
		),
	}
	if len(o.cmd) > 0 {
		runOpts = append(runOpts, testcontainers.WithCmd(o.cmd...))
	}

	container, err := postgres.Run(ctx, "postgres:18-alpine", runOpts...)
	if err != nil {
		return nil, "", fmt.Errorf("postgres.Run: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", fmt.Errorf("postgres.ConnectionString: %w", err)
	}

	return container, connStr, nil
}

func StartRedis(ctx context.Context) (string, func(), error) {
	redisC, err := redisModule.Run(ctx, "redis:alpine")
	if err != nil {
		return "", nil, fmt.Errorf("redis.Run: %w", err)
	}

	redisURI, err := redisC.ConnectionString(ctx)
	if err != nil {
		termErr := redisC.Terminate(ctx)
		if termErr != nil {
			fmt.Printf("redis terminate: %v\n", termErr)
		}

		return "", nil, fmt.Errorf("redis.ConnectionString: %w", err)
	}

	cleanup := func() {
		err := redisC.Terminate(ctx)
		if err != nil {
			fmt.Printf("redis terminate: %v\n", err)
		}
	}

	return redisURI, cleanup, nil
}

func StartRedisClient(ctx context.Context) (*redis.Client, func(), error) {
	redisURI, cleanup, err := StartRedis(ctx)
	if err != nil {
		return nil, nil, err
	}

	opts, err := redis.ParseURL(redisURI)
	if err != nil {
		cleanup()

		return nil, nil, fmt.Errorf("redis.ParseURL: %w", err)
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()

		cleanup()

		return nil, nil, fmt.Errorf("redis.Ping: %w", err)
	}

	fullCleanup := func() {
		_ = client.Close()

		cleanup()
	}

	return client, fullCleanup, nil
}
