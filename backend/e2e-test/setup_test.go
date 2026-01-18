package e2e_test

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/gavv/httpexpect/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	httpMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	v1 "github.com/skr1ms/CTFBoard/internal/controller/restapi/v1"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

var (
	TestDB    *sql.DB
	TestRedis *redis.Client
	testPort  string
)

// Mocks
type noOpMailer struct{}

func (m *noOpMailer) Send(ctx context.Context, msg mailer.Message) error {
	return nil
}

// Entry point
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Setup Infrastructure
	cleanup, err := setupInfrastructure(ctx)
	if err != nil {
		fmt.Printf("Infrastructure setup failed: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Run Migrations
	if err := runMigrations(ctx, TestDB); err != nil {
		fmt.Printf("Migrations failed: %v\n", err)
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
// E2E Setup
func setupE2E(t *testing.T) *httpexpect.Expect {
	if err := TestRedis.FlushAll(context.Background()).Err(); err != nil {
		t.Fatalf("failed to flush redis: %v", err)
	}

	baseURL := fmt.Sprintf("http://localhost:%s", testPort)

	return httpexpect.Default(t, baseURL)
}

// Infrastructure Setup

func setupInfrastructure(ctx context.Context) (func(), error) {
	if os.Getenv("USE_EXTERNAL_DB") == "true" {
		fmt.Println("Using EXTERNAL infrastructure (CI mode)...")
		return setupExternalInfra(ctx)
	}
	fmt.Println("Using TESTCONTAINERS infrastructure...")
	return setupTestContainers(ctx)
}

func setupTestContainers(ctx context.Context) (func(), error) {
	// MariaDB
	mariadbC, err := mariadb.Run(ctx,
		"mariadb:latest",
		mariadb.WithDatabase("test"),
		mariadb.WithUsername("user"),
		mariadb.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForSQL("3306/tcp", "mysql", func(host string, port nat.Port) string {
				return fmt.Sprintf("user:password@tcp(%s:%s)/test?parseTime=true", host, port.Port())
			}).WithStartupTimeout(120*time.Second).WithQuery("SELECT 1"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start mariadb container: %w", err)
	}

	// Redis
	redisC, err := redisContainer.Run(ctx, "redis:alpine")
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	// MariaDB Connection
	connStr, err := mariadbC.ConnectionString(ctx, "parseTime=true&multiStatements=true&timeout=30s")
	if err != nil {
		return nil, fmt.Errorf("failed to get db connection string: %w", err)
	}
	
	TestDB, err = sql.Open("mysql", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open db connection: %w", err)
	}

	// Verify DB Connection
	if err := TestDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	// Redis Connection
	redisURI, err := redisC.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get redis connection string: %w", err)
	}
	
	opts, err := redis.ParseURL(redisURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}
	TestRedis = redis.NewClient(opts)

	// Verify Redis Connection
	if err := TestRedis.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	// Cleanup func
	cleanup := func() {
		fmt.Println("Cleaning up containers...")
		_ = TestDB.Close()
		_ = TestRedis.Close()
		_ = mariadbC.Terminate(ctx)
		_ = redisC.Terminate(ctx)
	}
	return cleanup, nil
}

func setupExternalInfra(ctx context.Context) (func(), error) {
	// MariaDB Setup
	dbUser := getEnv("MARIADB_USER", "test_user")
	dbPass := getEnv("MARIADB_PASSWORD", "test_password")
	dbHost := getEnv("MARIADB_HOST", "mariadb")
	dbPort := getEnv("MARIADB_PORT", "3306")
	dbName := getEnv("MARIADB_DB", "test_board")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true&timeout=30s", dbUser, dbPass, dbHost, dbPort, dbName)
	var err error
	TestDB, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Retry Ping DB
	for i := 0; i < 20; i++ {
		if err := TestDB.PingContext(ctx); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err := TestDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("external db ping failed: %w", err)
	}

	// Redis Setup
	redisHost := getEnv("REDIS_HOST", "redis")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	TestRedis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
	})

	if err := TestRedis.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("external redis ping failed: %w", err)
	}

	return func() {
		_ = TestDB.Close()
		_ = TestRedis.Close()
	}, nil
}

// Migrations

func runMigrations(ctx context.Context, db *sql.DB) error {
	migrationsDir := filepath.Join("..", "migrations")
	
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir '%s': %w", migrationsDir, err)
	}

	fmt.Printf("Running migrations from %s...\n", migrationsDir)

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".up.sql") {
			continue
		}

		path := filepath.Join(migrationsDir, f.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		statements := strings.Split(string(content), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.HasPrefix(stmt, "--") || strings.HasPrefix(stmt, "#") {
				continue
			}

			if _, err := db.ExecContext(ctx, stmt); err != nil {
				if !isIgnorableDBError(err) {
					fmt.Printf("Warn: migration error in %s: %v\n", f.Name(), err)
				}
			}
		}
	}

	_, _ = db.ExecContext(ctx, "UPDATE competition SET start_time = ? WHERE id = 1", time.Now().Add(-24*time.Hour))

	return nil
}

// Server Setup

func startTestServer() (func(), error) {
	// Dependencies
	l := logger.New("error", "test")
	validatorService := validator.New()
	jwtService := jwt.NewJWTService("test-access-secret", "test-refresh-secret", 24*time.Hour, 72*time.Hour)

	// Repositories
	userRepo := persistent.NewUserRepo(TestDB)
	challengeRepo := persistent.NewChallengeRepo(TestDB)
	solveRepo := persistent.NewSolveRepo(TestDB)
	teamRepo := persistent.NewTeamRepo(TestDB)
	compRepo := persistent.NewCompetitionRepo(TestDB)
	hintRepo := persistent.NewHintRepo(TestDB)
	hintUnlockRepo := persistent.NewHintUnlockRepo(TestDB)
	awardRepo := persistent.NewAwardRepo(TestDB)
	txRepo := persistent.NewTxRepo(TestDB)
	tokenRepo := persistent.NewVerificationTokenRepo(TestDB)

	// UseCases
	userUC := usecase.NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)
	compUC := usecase.NewCompetitionUseCase(compRepo, TestRedis)
	challUC := usecase.NewChallengeUseCase(challengeRepo, solveRepo, txRepo, TestRedis, nil)
	solveUC := usecase.NewSolveUseCase(solveRepo, challengeRepo, compRepo, txRepo, TestRedis, nil)
	teamUC := usecase.NewTeamUseCase(teamRepo, userRepo)
	hintUC := usecase.NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, TestRedis)
	emailUC := usecase.NewEmailUseCase(userRepo, tokenRepo, &noOpMailer{}, 24*time.Hour, 1*time.Hour, "http://localhost:3000", true)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Timeout(60*time.Second))
	r.Use(httpMiddleware.Logger(l)) // Раскомментируйте для отладки HTTP запросов

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	v1.NewRouter(r, userUC, challUC, solveUC, teamUC, compUC, hintUC, emailUC, jwtService, TestRedis, validatorService, l, 100, 1*time.Minute)

	// Listener on random port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}
	testPort = fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)

	srv := &http.Server{
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 100 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}, nil
}

// Utils

func getEnv(key, fallback string) string {
	if v, exists := os.LookupEnv(key); exists {
		return v
	}
	return fallback
}

func isIgnorableDBError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "Duplicate key") ||
		strings.Contains(msg, "Duplicate column")
}
