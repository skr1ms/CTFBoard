package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	globalDBContainer *mariadb.MariaDBContainer
	globalDSN         string
	containerOnce     sync.Once
	containerErr      error
)

type TestDB struct {
	DB *sql.DB
}

func SetupTestDB(t *testing.T) *TestDB {
	ctx := context.Background()

	if os.Getenv("USE_EXTERNAL_DB") != "true" {
		containerOnce.Do(func() {
			globalDBContainer, globalDSN, containerErr = startMariaDBContainer(ctx)
		})
		if containerErr != nil {
			t.Fatalf("failed to start global db container: %s", containerErr)
		}
	} else {
		globalDSN = getExternalDSN()
	}

	db, err := sql.Open("mysql", globalDSN)
	if err != nil {
		t.Fatalf("failed to open db connection: %s", err)
	}

	if err := pingDB(ctx, db); err != nil {
		t.Fatalf("failed to ping db: %s", err)
	}

	if err := runMigrations(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %s", err)
	}

	truncateTables(t, db)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return &TestDB{DB: db}
}

func startMariaDBContainer(ctx context.Context) (*mariadb.MariaDBContainer, string, error) {
	container, err := mariadb.Run(ctx,
		"mariadb:latest",
		mariadb.WithDatabase("test"),
		mariadb.WithUsername("user"),
		mariadb.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForSQL("3306/tcp", "mysql", func(host string, port nat.Port) string {
				return fmt.Sprintf("user:password@tcp(%s:%s)/test?parseTime=true", host, port.Port())
			}).WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, "", err
	}

	connStr, err := container.ConnectionString(ctx, "parseTime=true&multiStatements=true")
	if err != nil {
		return nil, "", err
	}

	return container, connStr, nil
}

func getExternalDSN() string {
	host := getEnv("MARIADB_HOST", "mariadb")
	port := getEnv("MARIADB_PORT", "3306")
	user := getEnv("MARIADB_USER", "test_user")
	password := getEnv("MARIADB_PASSWORD", "test_password")
	dbname := getEnv("MARIADB_DB", "test_board")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true", user, password, host, port, dbname)
}

func pingDB(ctx context.Context, db *sql.DB) error {
	var err error
	for i := 0; i < 10; i++ {
		if err = db.PingContext(ctx); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return err
}

func runMigrations(ctx context.Context, db *sql.DB) error {
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

		statements := strings.Split(string(content), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.HasPrefix(stmt, "--") {
				continue
			}

			if _, err := db.ExecContext(ctx, stmt); err != nil {
				if !isIgnorableError(err) {
					return fmt.Errorf("migration error in %s: %w", f.Name(), err)
				}
			}
		}
	}
	return nil
}

func truncateTables(t *testing.T, db *sql.DB) {
	tables := []string{
		"hint_unlocks",
		"awards",
		"solves",
		"hints",
		"challenges",
		"verification_tokens",
		"teams",
		"users",
		"competition",
	}

	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table)); err != nil {
			t.Logf("failed to truncate table %s: %v", table, err)
		}
	}
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	_, _ = db.Exec("INSERT INTO competition (id, name) VALUES (1, 'CTF Competition') ON DUPLICATE KEY UPDATE id=id")
}

func getEnv(key, fallback string) string {
	if v, exists := os.LookupEnv(key); exists {
		return v
	}
	return fallback
}

func isIgnorableError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "already exists") || strings.Contains(s, "Duplicate")
}
