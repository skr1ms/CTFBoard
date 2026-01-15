package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDB struct {
	DB        *sql.DB
	Container *mariadb.MariaDBContainer
	DSN       string
}

func SetupTestDB(t *testing.T) *TestDB {
	ctx := context.Background()
	var db *sql.DB
	var mariadbContainer *mariadb.MariaDBContainer
	var dsn string
	var err error

	if os.Getenv("CI") == "true" {
		host := os.Getenv("MARIADB_HOST")
		port := os.Getenv("MARIADB_PORT")
		user := os.Getenv("MARIADB_USER")
		password := os.Getenv("MARIADB_PASSWORD")
		dbname := os.Getenv("MARIADB_DB")

		if host == "" {
			host = "mariadb"
		}
		if port == "" {
			port = "3306"
		}
		if user == "" {
			user = "test_user"
		}
		if password == "" {
			password = "test_password"
		}
		if dbname == "" {
			dbname = "test_board"
		}

		dsn = fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true&timeout=30s&readTimeout=30s&writeTimeout=30s",
			user, password, host, port, dbname,
		)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			t.Fatalf("failed to open db connection: %s", err)
		}
	} else {
		mariadbContainer, err = mariadb.Run(ctx,
			"mariadb:latest",
			mariadb.WithDatabase("test"),
			mariadb.WithUsername("user"),
			mariadb.WithPassword("password"),
			testcontainers.WithWaitStrategy(
				wait.ForSQL("3306/tcp", "mysql", func(host string, port nat.Port) string {
					return fmt.Sprintf("user:password@tcp(%s:%s)/test?parseTime=true", host, port.Port())
				}).
					WithStartupTimeout(120*time.Second).
					WithQuery("SELECT 1"),
			),
		)
		if err != nil {
			t.Fatalf("failed to start container: %s", err)
		}

		host, err := mariadbContainer.Host(ctx)
		if err != nil {
			t.Fatalf("failed to get container host: %s", err)
		}

		port, err := mariadbContainer.MappedPort(ctx, "3306/tcp")
		if err != nil {
			t.Fatalf("failed to get container port: %s", err)
		}

		dsn = fmt.Sprintf(
			"user:password@tcp(%s:%s)/test?parseTime=true&multiStatements=true&timeout=30s&readTimeout=30s&writeTimeout=30s",
			host, port.Port(),
		)

		db, err = sql.Open("mysql", dsn)
		if err != nil {
			t.Fatalf("failed to open db connection: %s", err)
		}
	}

	var pingErr error
	for i := 0; i < 20; i++ {
		pingErr = db.PingContext(ctx)
		if pingErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if pingErr != nil {
		t.Fatalf("failed to ping db after 20 attempts: %s", pingErr)
	}

	migrationsPath := filepath.Join("..", "migrations", "000001_init.up.sql")
	migrationSQL, err := os.ReadFile(migrationsPath)
	if err != nil {
		t.Fatalf("failed to read migration file: %s", err)
	}

	statements := strings.Split(string(migrationSQL), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		stmt = strings.TrimSuffix(stmt, "\n")
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			errStr := err.Error()
			if !strings.Contains(errStr, "already exists") &&
				!strings.Contains(errStr, "Duplicate key") &&
				!strings.Contains(errStr, "Duplicate column") {
				t.Fatalf("failed to execute migration: %s, error: %s", stmt[:min(100, len(stmt))], err)
			}
		}
	}

	t.Cleanup(func() {
		_ = db.Close()
		if mariadbContainer != nil {
			if err := mariadbContainer.Terminate(ctx); err != nil {
				t.Errorf("failed to terminate container: %s", err)
			}
		}
	})

	return &TestDB{
		DB:        db,
		Container: mariadbContainer,
		DSN:       dsn,
	}
}
