package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/wahrwelt-kit/go-pgkit/migrator/goose"
)

// TestMain: entry point for e2e test suite.
func TestMain(m *testing.M) {
	fmt.Println("E2E TestMain: starting...")

	ctx := context.Background()

	fmt.Println("E2E TestMain: setting up infrastructure (containers, DB, Redis)...")
	// Setup Infrastructure
	cleanup, err := setupInfrastructure(ctx)
	if err != nil {
		fmt.Printf("Infrastructure setup failed: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	_, thisFile, _, _ := runtime.Caller(0)
	backendDir := filepath.Dir(filepath.Dir(thisFile))
	oldWd, _ := os.Getwd()

	if err := os.Chdir(backendDir); err != nil {
		fmt.Printf("chdir to backend: %v\n", err)
		os.Exit(1)
	}

	defer func() { _ = os.Chdir(oldWd) }()

	if err := goose.Run(context.Background(), e2eConnStr, "migrations"); err != nil {
		fmt.Printf("Migrations failed: %v\n", err)
		os.Exit(1)
	}

	if _, err := TestPool.Exec(ctx, "UPDATE competition SET start_time = $1 WHERE id = 1", time.Now().Add(-24*time.Hour)); err != nil {
		fmt.Printf("Update competition start_time: %v\n", err)
		os.Exit(1)
	}

	// One-time clean slate for parallel tests (no per-test truncation)
	if err := truncateE2EDB(ctx, nil); err != nil {
		fmt.Printf("Initial truncate failed: %v\n", err)
		os.Exit(1)
	}

	// Start Application Server
	shutdownServer, err := startTestServer()
	if err != nil {
		fmt.Printf("Server start failed: %v\n", err)
		os.Exit(1)
	}
	defer shutdownServer()

	fmt.Printf("Test environment ready. Server running on port %s\n", testPort)

	// Run Tests
	code := m.Run()
	os.Exit(code)
}

// GetTestBaseURL returns the base URL of the test server (e.g. http://localhost:PORT)
func GetTestBaseURL() string {
	return fmt.Sprintf("http://localhost:%s", testPort)
}
