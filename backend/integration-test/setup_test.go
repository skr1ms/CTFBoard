package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/wahrwelt-kit/go-pgkit/migrator/goose"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/testutil"
)

var (
	_              *postgres.PostgresContainer
	globalConnStr  string
	containerOnce  sync.Once
	containerErr   error
	globalPool     *pgxpool.Pool
	globalPoolOnce sync.Once
)

type TestPool struct {
	Pool *pgxpool.Pool
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	if os.Getenv("USE_EXTERNAL_DB") != "true" {
		containerOnce.Do(func() {
			_, globalConnStr, containerErr = startPostgresContainer(ctx)
		})

		if containerErr != nil {
			fmt.Fprintf(os.Stderr, "failed to start container: %v\n", containerErr)
			os.Exit(1)
		}
	} else {
		globalConnStr = getExternalConnStr()
	}

	globalPoolOnce.Do(func() {
		var err error

		globalPool, err = pgxpool.New(ctx, globalConnStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create pool: %v\n", err)
			os.Exit(1)
		}
	})

	err := pingPool(ctx, globalPool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping pool: %v\n", err)
		os.Exit(1)
	}

	_, thisFile, _, _ := runtime.Caller(0)
	backendDir := filepath.Dir(filepath.Dir(thisFile))
	oldWd, _ := os.Getwd()

	err = os.Chdir(backendDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to chdir to backend: %v\n", err)
		os.Exit(1)
	}

	defer func() { _ = os.Chdir(oldWd) }()

	err = goose.Run(context.Background(), globalConnStr, "migrations")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	err = truncateTablesCtx(ctx, globalPool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to truncate: %v\n", err)
		os.Exit(1)
	}

	err = seedCompetition(ctx, globalPool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to seed competition: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func SetupTestPool(t *testing.T) *TestPool {
	t.Helper()

	return &TestPool{Pool: globalPool}
}

func SetupTestFixture(t *testing.T) *TestFixture {
	t.Helper()

	return NewTestFixture(SetupTestPool(t).Pool)
}

func startPostgresContainer(ctx context.Context) (*postgres.PostgresContainer, string, error) {
	return testutil.StartPostgres(ctx,
		testutil.PostgresWithUser(string(domain.RoleUser)),
		testutil.PostgresWithPassword("password"),
	)
}

func getExternalConnStr() string {
	host := testutil.GetEnv("POSTGRES_HOST", "postgres")
	port := testutil.GetEnv("POSTGRES_PORT", "5432")
	user := testutil.GetEnv("POSTGRES_USER", "test_user")
	password := testutil.GetEnv("POSTGRES_PASSWORD", "test_password")
	dbName := testutil.GetEnv("POSTGRES_DB", "test_board")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbName)
}

func pingPool(ctx context.Context, pool *pgxpool.Pool) error {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 200 * time.Millisecond
	bo.MaxElapsedTime = 10 * time.Second

	return backoff.Retry(func() error { return pool.Ping(ctx) }, backoff.WithContext(bo, ctx))
}

func truncateTablesCtx(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{
		"user_notifications",
		"oauth_accounts",
		"challenge_requirements",
		"challenge_topics",
		"challenge_tags",
		"submissions",
		"comments",
		"field_values",
		"api_tokens",
		"topics",
		"tags",
		"notifications",
		"fields",
		"brackets",
		"pages",
		"configs",
		"hint_unlocks",
		"awards",
		"solves",
		"hints",
		"solutions",
		"files",
		"challenges",
		"verification_tokens",
		"users",
		"teams",
		"competition",
		"ban_appeals",
	}

	for _, table := range tables {
		if _, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}

	return nil
}

func seedCompetition(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `INSERT INTO competition (id, name, start_time, end_time, mode, allow_team_switch) VALUES (1, 'CTF Competition', now() - INTERVAL '1 hour', now() + INTERVAL '24 hours', 'teams_only', true) ON CONFLICT (id) DO UPDATE SET start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, mode = EXCLUDED.mode, allow_team_switch = true, updated_at = NOW()`)

	return err
}

const (
	seaweedS3Port    = "8333"
	seaweedAccessKey = "admin"
	seaweedSecretKey = "admin"
	seaweedBucket    = "ctf-platform"
)

var (
	seaweedContainer testcontainers.Container
	seaweedOnce      sync.Once
	seaweedErr       error
	seaweedEndpoint  string
)

func SetupSeaweedFS(t *testing.T) (endpoint, accessKey, secretKey, bucket string) {
	t.Helper()

	ctx := context.Background()

	seaweedOnce.Do(func() {
		s3ConfigPath := findS3ConfigPath(t)
		req := testcontainers.ContainerRequest{
			Image:        "chrislusf/seaweedfs:latest",
			Cmd:          []string{"server", "-s3", "-s3.config=/etc/seaweedfs/s3.json"},
			ExposedPorts: []string{seaweedS3Port + "/tcp"},
			WaitingFor:   wait.ForListeningPort(seaweedS3Port + "/tcp").WithStartupTimeout(30 * time.Second),
			Files: []testcontainers.ContainerFile{
				{HostFilePath: s3ConfigPath, ContainerFilePath: "/etc/seaweedfs/s3.json", FileMode: 0o644},
			},
		}

		seaweedContainer, seaweedErr = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if seaweedErr != nil {
			return
		}

		host, err := seaweedContainer.Host(ctx)
		if err != nil {
			seaweedErr = err

			return
		}

		port, err := seaweedContainer.MappedPort(ctx, seaweedS3Port)
		if err != nil {
			seaweedErr = err

			return
		}

		seaweedEndpoint = host + ":" + port.Port()
	})

	if seaweedErr != nil {
		t.Fatalf("seaweedfs container: %v", seaweedErr)
	}

	return seaweedEndpoint, seaweedAccessKey, seaweedSecretKey, seaweedBucket
}

func findS3ConfigPath(t *testing.T) string {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "deployment", "seaweedfs", "s3.json"),
		filepath.Join("deployment", "seaweedfs", "s3.json"),
		filepath.Join("..", "..", "deployment", "seaweedfs", "s3.json"),
	}
	for _, p := range candidates {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}

		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}

	t.Fatal("s3.json not found (run tests from backend or repo root)")

	return ""
}
