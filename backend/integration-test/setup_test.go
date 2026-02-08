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
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	globalPoolContainer *postgres.PostgresContainer
	globalConnStr       string
	containerOnce       sync.Once
	containerErr        error
)

type TestPool struct {
	Pool *pgxpool.Pool
}

func SetupTestPool(t *testing.T) *TestPool {
	t.Helper()
	ctx := context.Background()

	if os.Getenv("USE_EXTERNAL_Pool") != "true" {
		containerOnce.Do(func() {
			globalPoolContainer, globalConnStr, containerErr = startPostgresContainer(ctx)
		})
		if containerErr != nil {
			t.Fatalf("failed to start global Pool container: %s", containerErr)
		}
	} else {
		globalConnStr = getExternalConnStr()
	}

	Pool, err := pgxpool.New(ctx, globalConnStr)
	if err != nil {
		t.Fatalf("failed to create connection Pool: %s", err)
	}

	if err := pingPool(ctx, Pool); err != nil {
		t.Fatalf("failed to ping Pool: %s", err)
	}

	if err := runMigrations(ctx, Pool); err != nil {
		t.Fatalf("failed to run migrations: %s", err)
	}

	truncateTables(t, Pool)

	t.Cleanup(func() {
		Pool.Close()
	})

	return &TestPool{Pool: Pool}
}

func startPostgresContainer(ctx context.Context) (*postgres.PostgresContainer, string, error) {
	container, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername(entity.RoleUser),
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
	Poolname := getEnv("POSTGRES_Pool", "test_board")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, Poolname)
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
		if !strings.HasSuffix(f.Name(), ".up.sql") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, f.Name()))
		if err != nil {
			return err
		}

		if _, err := Pool.Exec(ctx, string(content)); err != nil {
			if !isIgnorableError(err) {
				return fmt.Errorf("migration error in %s: %w", f.Name(), err)
			}
		}
	}
	return nil
}

func truncateTables(t *testing.T, Pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	tables := []string{
		"user_notifications",
		"challenge_tags",
		"submissions",
		"comments",
		"team_ratings",
		"global_ratings",
		"field_values",
		"api_tokens",
		"tags",
		"notifications",
		"fields",
		"ctf_events",
		"brackets",
		"pages",
		"configs",
		"hint_unlocks",
		"awards",
		"solves",
		"hints",
		"files",
		"challenges",
		"verification_tokens",
		"users",
		"teams",
		"competition",
	}

	for _, table := range tables {
		if _, err := Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			t.Logf("failed to truncate table %s: %v", table, err)
		}
	}

	_, err := Pool.Exec(ctx, "INSERT INTO competition (ID, name) VALUES (1, 'CTF Competition') ON CONFLICT (ID) DO NOTHING")
	if err != nil {
		t.Logf("insert competition: %v", err)
	}
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
	seaweedBucket    = "ctfboard"
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
