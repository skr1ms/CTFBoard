package e2e_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	v1 "github.com/skr1ms/CTFBoard/internal/controller/restapi/v1"
	wsV1 "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/skr1ms/CTFBoard/internal/usecase/challenge"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/internal/usecase/email"
	"github.com/skr1ms/CTFBoard/internal/usecase/settings"
	"github.com/skr1ms/CTFBoard/internal/usecase/team"
	"github.com/skr1ms/CTFBoard/internal/usecase/user"
	"github.com/skr1ms/CTFBoard/pkg/crypto"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/skr1ms/CTFBoard/pkg/validator"
	"github.com/skr1ms/CTFBoard/pkg/websocket"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	TestPool  *pgxpool.Pool
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
	if err := runMigrations(ctx, TestPool); err != nil {
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
	t.Helper()
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
	// PostgreSQL
	postgresC, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername(entity.RoleUser),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(120*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	// Redis
	redisC, err := redisContainer.Run(ctx, "redis:alpine")
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	// PostgreSQL Connection
	connStr, err := postgresC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to get db connection string: %w", err)
	}

	TestPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify DB Connection
	if err := TestPool.Ping(ctx); err != nil {
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
		TestPool.Close()
		_ = TestRedis.Close()
		if err := postgresC.Terminate(ctx); err != nil {
			fmt.Printf("postgres terminate: %v\n", err)
		}
		if err := redisC.Terminate(ctx); err != nil {
			fmt.Printf("redis terminate: %v\n", err)
		}
	}
	return cleanup, nil
}

func setupExternalInfra(ctx context.Context) (func(), error) {
	// PostgreSQL Setup
	dbUser := getEnv("POSTGRES_USER", "test_user")
	dbPass := getEnv("POSTGRES_PASSWORD", "test_password")
	dbHost := getEnv("POSTGRES_HOST", "postgres")
	dbPort := getEnv("POSTGRES_PORT", "5432")
	dbName := getEnv("POSTGRES_DB", "test_board")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	var err error
	TestPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}

	// Retry Ping DB
	for i := 0; i < 20; i++ {
		if err := TestPool.Ping(ctx); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err := TestPool.Ping(ctx); err != nil {
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
		TestPool.Close()
		_ = TestRedis.Close()
	}, nil
}

// Migrations

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
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

		if _, err := pool.Exec(ctx, string(content)); err != nil {
			if !isIgnorableDBError(err) {
				fmt.Printf("Warn: migration error in %s: %v\n", f.Name(), err)
			}
		}
	}

	if _, err := TestPool.Exec(ctx, "UPDATE competition SET start_time = $1 WHERE ID = 1", time.Now().Add(-24*time.Hour)); err != nil {
		return fmt.Errorf("update competition start_time: %w", err)
	}
	return nil
}

// Server setup

type testDeps struct {
	logger    logger.Logger
	validator validator.Validator
	jwt       *jwt.JWTService
	crypto    *crypto.CryptoService
}

type testUseCases struct {
	user        *user.UserUseCase
	team        *team.TeamUseCase
	award       *team.AwardUseCase
	email       *email.EmailUseCase
	challenge   *challenge.ChallengeUseCase
	hint        *challenge.HintUseCase
	file        *challenge.FileUseCase
	solve       *competition.SolveUseCase
	competition *competition.CompetitionUseCase
	backup      *competition.BackupUseCase
	stats       *competition.StatisticsUseCase
	settings    *settings.SettingsUseCase
	ws          *wsV1.Controller
}

func startTestServer() (func(), error) {
	deps, err := initTestDeps()
	if err != nil {
		return nil, err
	}

	useCases, tempStorageDir, err := initTestUseCases(deps)
	if err != nil {
		return nil, err
	}

	r := setupTestRouter(deps.logger, useCases, deps.validator, deps.jwt, tempStorageDir)

	ctx := context.Background()
	ls := net.ListenConfig{}
	listener, err := ls.Listen(ctx, "tcp", ":0")
	if err != nil {
		return nil, err
	}
	testPort = fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port) //nolint:errcheck // type asserted

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
		if err := srv.Shutdown(ctx); err != nil {
			fmt.Printf("server shutdown: %v\n", err)
		}
		_ = os.RemoveAll(tempStorageDir)
	}, nil
}

// Deps (logger, validator, jwt, crypto)
func initTestDeps() (*testDeps, error) {
	l := logger.New(&logger.Options{
		Level:  logger.ErrorLevel,
		Output: logger.ConsoleOutput,
	})
	validatorService := validator.New()
	jwtService := jwt.NewJWTService("test-access-secret", "test-refresh-secret", 24*time.Hour, 72*time.Hour)
	dummyCrypto, err := crypto.NewCryptoService("1234567890123456789012345678901212345678901234567890123456789012")
	if err != nil {
		return nil, fmt.Errorf("failed to init crypto service: %w", err)
	}

	return &testDeps{
		logger:    l,
		validator: validatorService,
		jwt:       jwtService,
		crypto:    dummyCrypto,
	}, nil
}

// Repositories, storage, hub, usecases
func initTestUseCases(deps *testDeps) (*testUseCases, string, error) {
	// Repositories
	userRepo := persistent.NewUserRepo(TestPool)
	challengeRepo := persistent.NewChallengeRepo(TestPool)
	solveRepo := persistent.NewSolveRepo(TestPool)
	teamRepo := persistent.NewTeamRepo(TestPool)
	compRepo := persistent.NewCompetitionRepo(TestPool)
	hintRepo := persistent.NewHintRepo(TestPool)
	hintUnlockRepo := persistent.NewHintUnlockRepo(TestPool)
	awardRepo := persistent.NewAwardRepo(TestPool)
	txRepo := persistent.NewTxRepo(TestPool)
	tokenRepo := persistent.NewVerificationTokenRepo(TestPool)
	auditLogRepo := persistent.NewAuditLogRepo(TestPool)
	statsRepo := persistent.NewStatisticsRepository(TestPool)
	fileRepo := persistent.NewFileRepository(TestPool)
	backupRepo := persistent.NewBackupRepo(TestPool)
	appSettingsRepo := persistent.NewAppSettingsRepo(TestPool)

	// Storage
	tempStorageDir, err := os.MkdirTemp("", "ctfboard-e2e-storage")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp storage dir: %w", err)
	}
	fileStorage, err := storage.NewFilesystemProvider(tempStorageDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create storage provider: %w", err)
	}

	// WebSocket hub
	hub := websocket.NewHub(TestRedis, "ctfboard:events")
	go hub.Run()
	go hub.SubscribeToRedis(context.Background())

	// Usecases
	userUC := user.NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, deps.jwt)
	compUC := competition.NewCompetitionUseCase(compRepo, auditLogRepo, TestRedis)
	challengeUC := challenge.NewChallengeUseCase(challengeRepo, solveRepo, txRepo, compRepo, TestRedis, hub, auditLogRepo, deps.crypto)
	solveUC := competition.NewSolveUseCase(solveRepo, challengeRepo, compRepo, userRepo, txRepo, TestRedis, hub)
	teamUC := team.NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo)
	hintUC := challenge.NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, TestRedis)
	awardUC := team.NewAwardUseCase(awardRepo, txRepo, TestRedis)
	emailUC := email.NewEmailUseCase(userRepo, tokenRepo, &noOpMailer{}, 24*time.Hour, 1*time.Hour, "http://localhost:3000", true)
	statsUC := competition.NewStatisticsUseCase(statsRepo, TestRedis)
	backupUC := competition.NewBackupUseCase(compRepo, challengeRepo, hintRepo, teamRepo, userRepo, awardRepo, solveRepo, fileRepo, backupRepo, fileStorage, txRepo, deps.logger)
	settingsUC := settings.NewSettingsUseCase(appSettingsRepo, auditLogRepo, TestRedis)

	ws := wsV1.NewController(hub, deps.logger, []string{"*"})

	fileUC := challenge.NewFileUseCase(fileRepo, fileStorage, 1*time.Hour)

	return &testUseCases{
		user:        userUC,
		challenge:   challengeUC,
		solve:       solveUC,
		team:        teamUC,
		competition: compUC,
		hint:        hintUC,
		award:       awardUC,
		email:       emailUC,
		file:        fileUC,
		stats:       statsUC,
		backup:      backupUC,
		settings:    settingsUC,
		ws:          ws,
	}, tempStorageDir, nil
}

// Router (chi, middleware, api v1 routes)
func setupTestRouter(l logger.Logger, uc *testUseCases, validatorService validator.Validator, jwtService *jwt.JWTService, tempStorageDir string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Timeout(60*time.Second))
	r.Use(restapimiddleware.Logger(l))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK")) //nolint:errcheck // best-effort health
	})

	r.Route("/api/v1", func(apiRouter chi.Router) {
		v1.NewRouter(apiRouter, uc.user, uc.challenge, uc.solve, uc.team, uc.competition, uc.hint, uc.email, uc.file, uc.award, uc.stats, uc.backup, uc.settings, jwtService, TestRedis, uc.ws, validatorService, l, 100, 1*time.Minute, false)

		// Static routes for E2E Filesystem
		apiRouter.Get("/files/download/*", func(w http.ResponseWriter, r *http.Request) {
			fs := http.StripPrefix("/api/v1/files/download/", http.FileServer(http.Dir(tempStorageDir)))
			fs.ServeHTTP(w, r)
		})
	})

	return r
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
		strings.Contains(msg, "duplicate key")
}
