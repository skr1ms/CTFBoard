package load_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-pgkit/migrator/goose"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/testutil"
)

var (
	testDBPool      *pgxpool.Pool
	testRedisClient *redis.Client
	testDBConnStr   string
	testBaseURL     string
	Fixture         *TestFixture
)

func TestMain(m *testing.M) {
	fmt.Println("[load-test] starting environment setup...")

	ctx := context.Background()

	cleanup, err := setupInfra(ctx)
	if err != nil {
		fmt.Printf("[load-test] infra setup failed: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	if err := goose.Run(context.Background(), testDBConnStr, filepath.Join("..", "migrations")); err != nil {
		fmt.Printf("[load-test] migrations failed: %v\n", err)
		os.Exit(1)
	}

	if _, err := testDBPool.Exec(ctx, "UPDATE competition SET start_time = $1 WHERE id = 1", time.Now().Add(-24*time.Hour)); err != nil {
		fmt.Printf("[load-test] set competition start_time: %v\n", err)
		os.Exit(1)
	}

	if err := seedAppSettings(ctx, testDBPool); err != nil {
		fmt.Printf("[load-test] app_settings seed failed: %v\n", err)
		os.Exit(1)
	}

	baseURL, shutdownServer, err := startLoadTestServer(testDBPool, testRedisClient)
	if err != nil {
		fmt.Printf("[load-test] server start failed: %v\n", err)
		os.Exit(1)
	}
	defer shutdownServer()

	testBaseURL = baseURL
	fmt.Printf("[load-test] server ready at %s\n", testBaseURL)

	fmt.Println("[load-test] seeding fixture data...")

	Fixture, err = seedLoadTestData(ctx, testBaseURL, testDBPool)
	if err != nil {
		fmt.Printf("[load-test] seed failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[load-test] fixture ready: %d users, %d challenges\n", len(Fixture.Users), len(Fixture.ChallengeIDs))

	pipe := testRedisClient.Pipeline()

	for _, u := range Fixture.Users {
		pipe.Del(ctx, "user:"+u.UserID)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		fmt.Printf("[load-test] warn: cache flush: %v\n", err)
	}

	code := m.Run()

	FlushReports()
	os.Exit(code)
}

func setupInfra(ctx context.Context) (func(), error) {
	if os.Getenv("USE_EXTERNAL_DB") == "true" {
		return setupExternalInfra(ctx)
	}

	return setupContainerInfra(ctx)
}

func setupContainerInfra(ctx context.Context) (func(), error) {
	pgC, connStr, err := testutil.StartPostgres(ctx,
		testutil.PostgresWithDatabase("loadtest"),
		testutil.PostgresWithUser(string(domain.RoleUser)),
		testutil.PostgresWithPassword("password"),
		testutil.PostgresWithCmd("postgres", "-c", "max_connections=400"),
		testutil.PostgresWithStartupTimeout(120*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	testDBConnStr = connStr

	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse pg config: %w", err)
	}

	cfg.MaxConns = 200
	cfg.MinConns = 50
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	testDBPool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	if err := testDBPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	redisURI, redisCleanup, err := testutil.StartRedis(ctx)
	if err != nil {
		termErr := pgC.Terminate(ctx)
		if termErr != nil {
			fmt.Printf("postgres terminate on cleanup: %v\n", termErr)
		}

		return nil, fmt.Errorf("start redis: %w", err)
	}

	opts, err := redis.ParseURL(redisURI)
	if err != nil {
		redisCleanup()

		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	opts.PoolSize = 200
	opts.MinIdleConns = 20
	opts.PoolFIFO = true
	opts.PoolTimeout = 5 * time.Second

	testRedisClient = redis.NewClient(opts)
	if err := testRedisClient.Ping(ctx).Err(); err != nil {
		_ = testRedisClient.Close()

		redisCleanup()

		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return func() {
		testDBPool.Close()
		_ = testRedisClient.Close()

		redisCleanup()

		_ = pgC.Terminate(ctx) //nolint:errcheck
	}, nil
}

func setupExternalInfra(ctx context.Context) (func(), error) {
	dbUser := testutil.GetEnv("POSTGRES_USER", "test_user")
	dbPass := testutil.GetEnv("POSTGRES_PASSWORD", "test_password")
	dbHost := testutil.GetEnv("POSTGRES_HOST", "postgres")
	dbPort := testutil.GetEnv("POSTGRES_PORT", "5432")
	dbName := testutil.GetEnv("POSTGRES_DB", "loadtest")
	testDBConnStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)

	cfg, err := pgxpool.ParseConfig(testDBConnStr)
	if err != nil {
		return nil, fmt.Errorf("parse pg config: %w", err)
	}

	cfg.MaxConns = 200
	cfg.MinConns = 50
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	testDBPool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	bo := backoff.NewExponentialBackOff()

	bo.MaxElapsedTime = 15 * time.Second
	if err := backoff.Retry(func() error { return testDBPool.Ping(ctx) }, backoff.WithContext(bo, ctx)); err != nil {
		return nil, fmt.Errorf("ping external db: %w", err)
	}

	redisHost := testutil.GetEnv("REDIS_HOST", "redis")
	redisPort := testutil.GetEnv("REDIS_PORT", "6379")
	redisPassword := testutil.GetEnv("REDIS_PASSWORD", "")

	testRedisClient = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password:     redisPassword,
		PoolSize:     200,
		MinIdleConns: 20,
		PoolFIFO:     true,
		PoolTimeout:  5 * time.Second,
	})
	if err := testRedisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping external redis: %w", err)
	}

	return func() {
		testDBPool.Close()
		_ = testRedisClient.Close()
	}, nil
}

func seedAppSettings(ctx context.Context, pool *pgxpool.Pool) error {
	retry := func() error {
		_, err := pool.Exec(ctx, `
			TRUNCATE TABLE
				configs, comments, api_tokens,
				field_values, fields, brackets, pages, user_notifications, notifications,
				submissions, challenge_tags, tags, audit_logs, team_audit_log, app_settings,
				solutions, files, verification_tokens, awards, hint_unlocks, hints, solves,
				challenges, teams, users, competition
			RESTART IDentity CASCADE
		`)
		if err != nil {
			return err
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO competition (id, name, is_paused, is_public, mode, allow_team_switch, min_team_size, max_team_size, start_time, end_time)
			VALUES (1, 'Load Test CTF', false, true, 'flexible', true, 1, 100, $1, $2)
			ON CONFLICT (id) DO UPDATE
				SET name = EXCLUDED.name, is_paused = EXCLUDED.is_paused,
				    start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time,
				    updated_at = NOW()
		`, time.Now().Add(-2*time.Hour), time.Now().Add(48*time.Hour))
		if err != nil {
			return err
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO app_settings (
				id, app_name, verify_emails, frontend_url, cors_origins,
				resend_enabled, resend_from_email, resend_from_name,
				verify_ttl_hours, reset_ttl_hours,
				submit_limit_per_user, submit_limit_duration_min,
				scoreboard_visible, registration_open,
				rate_limit_login_per_minute, rate_limit_register_per_minute,
				rate_limit_forgot_password_per_minute, rate_limit_reset_password_per_minute,
				rate_limit_logout_per_minute, rate_limit_refresh_per_minute,
				rate_limit_scoreboard_per_minute, rate_limit_general_ip_per_minute,
				rate_limit_verify_email_per_minute, rate_limit_oauth_callback_per_minute,
				updated_at
			) VALUES (
				1, 'Load Test CTF', false, 'http://localhost:3000', 'http://localhost:3000',
				false, 'noreply@lt.local', 'LoadTestCTF',
				24, 1,
				100000, 1,
				'public', true,
				100000, 100000,
				100000, 100000,
				100000, 100000,
				100000, 100000,
				100000, 100000,
				NOW()
			) ON CONFLICT (id) DO NOTHING
		`)

		return err
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 50 * time.Millisecond
	bo.MaxElapsedTime = 10 * time.Second

	return backoff.Retry(func() error {
		err := retry()
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "40P01" {
				return err
			}

			return backoff.Permanent(err)
		}

		return nil
	}, backoff.WithContext(bo, context.Background()))
}
