package e2e

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
	httpMiddleware "github.com/skr1ms/CTFBoard/internal/controller/http/middleware"
	v1 "github.com/skr1ms/CTFBoard/internal/controller/http/v1"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	TestDB     *sql.DB
	TestRedis  *redis.Client
	testServer *http.Server
	testRouter chi.Router
	testPort   string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	var mariadbContainer *mariadb.MariaDBContainer
	var redisC *redisContainer.RedisContainer
	var err error

	if os.Getenv("USE_EXTERNAL_DB") == "true" {
		// MariaDB Setup (CI)
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

		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true&timeout=30s&readTimeout=30s&writeTimeout=30s",
			user, password, host, port, dbname,
		)
		TestDB, err = sql.Open("mysql", dsn)
		if err != nil {
			fmt.Printf("failed to open db connection: %s\n", err)
			os.Exit(1)
		}

		// Redis Setup (CI)
		redisHost := os.Getenv("REDIS_HOST")
		redisPort := os.Getenv("REDIS_PORT")
		redisPassword := os.Getenv("REDIS_PASSWORD")

		if redisHost == "" {
			redisHost = "redis"
		}
		if redisPort == "" {
			redisPort = "6379"
		}
		// Redis client handles empty password fine if not required, but here we likely need it

		TestRedis = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
			Password: redisPassword,
		})

	} else {
		// MariaDB Setup (Testcontainers)
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
			fmt.Printf("failed to start mariadb container: %s\n", err)
			os.Exit(1)
		}

		// --- Redis Setup (Testcontainers) ---
		redisC, err = redisContainer.Run(ctx, "redis:alpine")
		if err != nil {
			fmt.Printf("failed to start redis container: %s\n", err)
			os.Exit(1)
		}

		// MariaDB Connection
		host, err := mariadbContainer.Host(ctx)
		if err != nil {
			fmt.Printf("failed to get mariadb host: %s\n", err)
			os.Exit(1)
		}
		port, err := mariadbContainer.MappedPort(ctx, "3306/tcp")
		if err != nil {
			fmt.Printf("failed to get mariadb port: %s\n", err)
			os.Exit(1)
		}
		dsn := fmt.Sprintf(
			"user:password@tcp(%s:%s)/test?parseTime=true&multiStatements=true&timeout=30s&readTimeout=30s&writeTimeout=30s",
			host, port.Port(),
		)
		TestDB, err = sql.Open("mysql", dsn)
		if err != nil {
			fmt.Printf("failed to open db connection: %s\n", err)
			os.Exit(1)
		}

		// Redis Connection
		redisURI, err := redisC.ConnectionString(ctx)
		if err != nil {
			fmt.Printf("failed to get redis connection string: %s\n", err)
			os.Exit(1)
		}
		opts, err := redis.ParseURL(redisURI)
		if err != nil {
			fmt.Printf("failed to parse redis url: %s\n", err)
			os.Exit(1)
		}
		TestRedis = redis.NewClient(opts)
	}

	// Ping Redis
	if err := TestRedis.Ping(ctx).Err(); err != nil {
		fmt.Printf("failed to ping redis: %s\n", err)
		os.Exit(1)
	}

	// Ping DB
	var pingErr error
	for i := 0; i < 20; i++ {
		pingErr = TestDB.PingContext(ctx)
		if pingErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if pingErr != nil {
		fmt.Printf("failed to ping db after 20 attempts: %s\n", pingErr)
		os.Exit(1)
	}

	// Migrations
	migrationsPath := filepath.Join("..", "migrations", "000001_init.up.sql")
	migrationSQL, err := os.ReadFile(migrationsPath)
	if err != nil {
		fmt.Printf("failed to read migration file: %s\n", err)
		os.Exit(1)
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
		if _, err := TestDB.ExecContext(ctx, stmt); err != nil {
			errStr := err.Error()
			if !strings.Contains(errStr, "already exists") &&
				!strings.Contains(errStr, "Duplicate key") &&
				!strings.Contains(errStr, "Duplicate column") {
				fmt.Printf("failed to execute migration: %s\n", err)
			}
		}
	}
	_, _ = TestDB.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'user'")

	l := logger.New("error", "test")

	userRepo := persistent.NewUserRepo(TestDB)
	challengeRepo := persistent.NewChallengeRepo(TestDB)
	solveRepo := persistent.NewSolveRepo(TestDB)
	teamRepo := persistent.NewTeamRepo(TestDB)

	validatorService := validator.New()

	jwtService := jwt.NewJWTService(
		"test-access-secret",
		"test-refresh-secret",
		1*24*time.Hour,
		3*24*time.Hour,
	)

	userUC := usecase.NewUserUseCase(userRepo, teamRepo, solveRepo, jwtService)
	challengeUC := usecase.NewChallengeUseCase(challengeRepo, solveRepo, TestRedis)
	solveUC := usecase.NewSolveUseCase(solveRepo, TestRedis)
	teamUC := usecase.NewTeamUseCase(teamRepo, userRepo)

	testRouter = chi.NewRouter()
	testRouter.Use(middleware.RequestID)
	testRouter.Use(middleware.RealIP)
	testRouter.Use(httpMiddleware.Logger(l))
	testRouter.Use(middleware.Recoverer)
	testRouter.Use(middleware.Timeout(60 * time.Second))

	testRouter.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	v1.NewRouter(
		testRouter,
		userUC,
		challengeUC,
		solveUC,
		teamUC,
		jwtService,
		validatorService,
		l,
	)

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		fmt.Printf("failed to create listener: %s\n", err)
		os.Exit(1)
	}
	testPort = fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)

	testServer = &http.Server{
		Handler:      testRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 100 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := testServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("server error: %s\n", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	code := m.Run()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := testServer.Shutdown(ctx); err != nil {
		fmt.Printf("failed to shutdown server: %s\n", err)
	}
	_ = TestDB.Close()
	_ = TestRedis.Close()

	if mariadbContainer != nil {
		if err := mariadbContainer.Terminate(ctx); err != nil {
			fmt.Printf("failed to terminate mariadb: %s\n", err)
		}
	}
	if redisC != nil {
		if err := redisC.Terminate(ctx); err != nil {
			fmt.Printf("failed to terminate redis: %s\n", err)
		}
	}

	os.Exit(code)
}

func setupE2E(t *testing.T) *httpexpect.Expect {
	if err := TestRedis.FlushAll(context.Background()).Err(); err != nil {
		t.Fatalf("failed to flush redis: %v", err)
	}

	baseURL := fmt.Sprintf("http://localhost:%s", testPort)
	return httpexpect.Default(t, baseURL)
}
