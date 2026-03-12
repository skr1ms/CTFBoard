package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
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
	if err := pingPool(ctx, globalPool); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping pool: %v\n", err)
		os.Exit(1)
	}
	if err := runMigrations(ctx, globalPool); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run migrations: %v\n", err)
		os.Exit(1)
	}
	if err := truncateTablesCtx(ctx, globalPool); err != nil {
		fmt.Fprintf(os.Stderr, "failed to truncate: %v\n", err)
		os.Exit(1)
	}
	if err := seedCompetition(ctx, globalPool); err != nil {
		fmt.Fprintf(os.Stderr, "failed to seed competition: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func SetupTestPool(t *testing.T) *TestPool {
	t.Helper()
	return &TestPool{Pool: globalPool}
}

func startPostgresContainer(ctx context.Context) (*postgres.PostgresContainer, string, error) {
	container, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername(string(entity.RoleUser)),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, "", err
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", err
	}

	return container, connStr, nil
}

func getExternalConnStr() string {
	host := getEnv("POSTGRES_HOST", "postgres")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "test_user")
	password := getEnv("POSTGRES_PASSWORD", "test_password")
	dbName := getEnv("POSTGRES_DB", "test_board")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbName)
}

func pingPool(ctx context.Context, Pool *pgxpool.Pool) error {
	var err error
	for i := 0; i < 10; i++ {
		if err = Pool.Ping(ctx); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return err
}

func runMigrations(ctx context.Context, Pool *pgxpool.Pool) error {
	migrationsDir := filepath.Join("..", "migrations")
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}

		raw, err := os.ReadFile(filepath.Join(migrationsDir, f.Name()))
		if err != nil {
			return err
		}

		if _, err := Pool.Exec(ctx, extractGooseUp(string(raw))); err != nil {
			if !isIgnorableError(err) {
				return fmt.Errorf("migration error in %s: %w", f.Name(), err)
			}
		}
	}
	return nil
}

func extractGooseUp(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inUp := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "-- +goose Up" {
			inUp = true
			continue
		}
		if trimmed == "-- +goose Down" {
			break
		}
		if strings.HasPrefix(trimmed, "-- +goose") {
			continue
		}
		if inUp {
			// CONCURRENTLY cannot run in a transaction; tests use an isolated
			// container with no concurrent load, so a regular index is fine.
			result = append(result, strings.ReplaceAll(line, " CONCURRENTLY", ""))
		}
	}
	return strings.Join(result, "\n")
}

func truncateTablesCtx(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{
		"user_notifications",
		"oauth_accounts",
		"challenge_requirements",
		"challenge_tags",
		"submissions",
		"comments",
		"field_values",
		"api_tokens",
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
	}

	for _, table := range tables {
		if _, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}
	return nil
}

func seedCompetition(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `INSERT INTO competition (id, name, start_time, end_time) VALUES (1, 'CTF Competition', now() - INTERVAL '1 hour', now() + INTERVAL '24 hours') ON CONFLICT (id) DO UPDATE SET start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, updated_at = NOW()`)
	return err
}

func getEnv(key, fallback string) string {
	if v, exists := os.LookupEnv(key); exists {
		return v
	}
	return fallback
}

func isIgnorableError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "already exists") || strings.Contains(s, "duplicate")
}

const (
	seaweedS3Port    = "8333"
	seaweedAccessKey = "admin"
	seaweedSecretKey = "admin"
	seaweedBucket    = "astroctfb"
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
