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

	"github.com/cenkalti/backoff/v4"
	"github.com/gavv/httpexpect/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	v1 "github.com/skr1ms/CTFBoard/internal/controller/restapi/v1"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	wsV1 "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/internal/usecase/challenge"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/internal/usecase/email"
	"github.com/skr1ms/CTFBoard/internal/usecase/notification"
	"github.com/skr1ms/CTFBoard/internal/usecase/page"
	"github.com/skr1ms/CTFBoard/internal/usecase/settings"
	"github.com/skr1ms/CTFBoard/internal/usecase/team"
	"github.com/skr1ms/CTFBoard/internal/usecase/user"
	"github.com/skr1ms/CTFBoard/pkg/cache"
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

func GetTestBaseURL() string {
	return fmt.Sprintf("http://localhost:%s", testPort)
}

func setupE2E(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if err := TestRedis.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("failed to flush redis: %v", err)
	}
	truncateE2EDB(ctx, t)
	if err := TestRedis.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("failed to flush redis after truncate: %v", err)
	}
	_ = httpexpect.Default(t, GetTestBaseURL())
}

func truncateE2EDB(ctx context.Context, t *testing.T) {
	t.Helper()
	_, err := TestPool.Exec(ctx, `TRUNCATE TABLE
		global_ratings, team_ratings, ctf_events, configs, comments, api_tokens,
		field_values, fields, brackets, pages, user_notifications, notifications,
		submissions, challenge_tags, tags, audit_logs, team_audit_log, app_settings,
		files, verification_tokens, awards, hint_unlocks, hints, solves,
		challenges, teams, users, competition
		RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate db: %v", err)
	}
	_, err = TestPool.Exec(ctx, `INSERT INTO competition (id, name, is_paused, is_public, mode, allow_team_switch, min_team_size, max_team_size, start_time, end_time)
		VALUES (1, 'CTF Competition', false, true, 'flexible', true, 1, 10, NOW() - INTERVAL '1 hour', NOW() + INTERVAL '24 hours')
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, is_paused = EXCLUDED.is_paused, is_public = EXCLUDED.is_public, mode = EXCLUDED.mode, allow_team_switch = EXCLUDED.allow_team_switch, min_team_size = EXCLUDED.min_team_size, max_team_size = EXCLUDED.max_team_size, start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, updated_at = NOW()`)
	if err != nil {
		t.Fatalf("insert competition: %v", err)
	}
	_, err = TestPool.Exec(ctx, `INSERT INTO app_settings (id, app_name, verify_emails, frontend_url, cors_origins, resend_enabled, resend_from_email, resend_from_name, verify_ttl_hours, reset_ttl_hours, submit_limit_per_user, submit_limit_duration_min, scoreboard_visible, registration_open, updated_at)
		VALUES (1, 'CTFBoard', true, 'http://localhost:3000', 'http://localhost:3000,http://localhost:5173', false, 'noreply@ctfboard.local', 'CTFBoard', 24, 1, 10, 1, 'public', true, NOW())
		ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		t.Fatalf("insert app_settings: %v", err)
	}
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

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 15 * time.Second
	if err := backoff.Retry(func() error { return TestPool.Ping(ctx) }, backoff.WithContext(bo, ctx)); err != nil {
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

type testRepos struct {
	apiTokenRepo     *persistent.APITokenRepo
	appSettingsRepo  *persistent.AppSettingsRepo
	auditLogRepo     *persistent.AuditLogRepo
	awardRepo        *persistent.AwardRepo
	backupRepo       *persistent.BackupRepo
	bracketRepo      *persistent.BracketRepo
	challengeRepo    *persistent.ChallengeRepo
	commentRepo      *persistent.CommentRepo
	compRepo         *persistent.CompetitionRepo
	configRepo       *persistent.ConfigRepo
	fieldRepo        *persistent.FieldRepo
	fieldValueRepo   *persistent.FieldValueRepo
	fileRepo         *persistent.FileRepository
	hintRepo         *persistent.HintRepo
	hintUnlockRepo   *persistent.HintUnlockRepo
	notificationRepo *persistent.NotificationRepo
	pageRepo         *persistent.PageRepo
	ratingRepo       *persistent.RatingRepo
	solveRepo        *persistent.SolveRepo
	statsRepo        *persistent.StatisticsRepository
	submissionRepo   *persistent.SubmissionRepo
	tagRepo          *persistent.TagRepo
	teamRepo         *persistent.TeamRepo
	tokenRepo        *persistent.VerificationTokenRepo
	txRepo           *persistent.TxRepo
	userRepo         *persistent.UserRepo
}

type testUseCases struct {
	user            *user.UserUseCase
	team            *team.TeamUseCase
	award           *team.AwardUseCase
	email           *email.EmailUseCase
	challenge       *challenge.ChallengeUseCase
	hint            *challenge.HintUseCase
	file            *challenge.FileUseCase
	solve           *competition.SolveUseCase
	competition     *competition.CompetitionUseCase
	backup          *competition.BackupUseCase
	stats           *competition.StatisticsUseCase
	settings        *settings.SettingsUseCase
	ws              *wsV1.Controller
	submissionUC    *competition.SubmissionUseCase
	tagUC           *challenge.TagUseCase
	fieldUC         *settings.FieldUseCase
	pageUC          *page.PageUseCase
	bracketUC       *competition.BracketUseCase
	ratingUC        *competition.RatingUseCase
	notifUC         usecase.NotificationUseCase
	apiTokenUC      usecase.APITokenUseCase
	dynamicConfigUC *competition.DynamicConfigUseCase
	commentUC       *challenge.CommentUseCase
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

func initTestRepos() *testRepos {
	return &testRepos{
		userRepo:         persistent.NewUserRepo(TestPool),
		challengeRepo:    persistent.NewChallengeRepo(TestPool),
		solveRepo:        persistent.NewSolveRepo(TestPool),
		teamRepo:         persistent.NewTeamRepo(TestPool),
		compRepo:         persistent.NewCompetitionRepo(TestPool),
		hintRepo:         persistent.NewHintRepo(TestPool),
		hintUnlockRepo:   persistent.NewHintUnlockRepo(TestPool),
		awardRepo:        persistent.NewAwardRepo(TestPool),
		txRepo:           persistent.NewTxRepo(TestPool),
		tokenRepo:        persistent.NewVerificationTokenRepo(TestPool),
		auditLogRepo:     persistent.NewAuditLogRepo(TestPool),
		statsRepo:        persistent.NewStatisticsRepository(TestPool),
		fileRepo:         persistent.NewFileRepository(TestPool),
		backupRepo:       persistent.NewBackupRepo(TestPool),
		appSettingsRepo:  persistent.NewAppSettingsRepo(TestPool),
		tagRepo:          persistent.NewTagRepo(TestPool),
		fieldRepo:        persistent.NewFieldRepo(TestPool),
		fieldValueRepo:   persistent.NewFieldValueRepo(TestPool),
		submissionRepo:   persistent.NewSubmissionRepo(TestPool),
		pageRepo:         persistent.NewPageRepo(TestPool),
		bracketRepo:      persistent.NewBracketRepo(TestPool),
		ratingRepo:       persistent.NewRatingRepo(TestPool),
		notificationRepo: persistent.NewNotificationRepo(TestPool),
		apiTokenRepo:     persistent.NewAPITokenRepo(TestPool),
		configRepo:       persistent.NewConfigRepo(TestPool),
		commentRepo:      persistent.NewCommentRepo(TestPool),
	}
}

func initTestStorageAndHub() (string, storage.Provider, *websocket.Hub, error) {
	tempStorageDir, err := os.MkdirTemp("", "ctfboard-e2e-storage")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp storage dir: %w", err)
	}
	fileStorage, err := storage.NewFilesystemProvider(tempStorageDir)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create storage provider: %w", err)
	}
	ctx := context.Background()
	hub := websocket.NewHub(TestRedis, "ctfboard:events")
	go hub.Run(ctx)
	go hub.SubscribeToRedis(ctx)
	return tempStorageDir, fileStorage, hub, nil
}

func buildTestUseCases(deps *testDeps, repos *testRepos, fileStorage storage.Provider, hub *websocket.Hub) *testUseCases {
	fieldValidator := settings.NewFieldValidator(repos.fieldRepo)
	broadcaster := websocket.NewBroadcaster(hub)
	userUC := user.NewUserUseCase(repos.userRepo, repos.teamRepo, repos.solveRepo, repos.txRepo, deps.jwt, fieldValidator, repos.fieldValueRepo)
	compUC := competition.NewCompetitionUseCase(repos.compRepo, repos.auditLogRepo, TestRedis)
	challengeUC := challenge.NewChallengeUseCase(
		repos.challengeRepo,
		challenge.WithTagRepo(repos.tagRepo),
		challenge.WithSolveRepo(repos.solveRepo),
		challenge.WithTxRepo(repos.txRepo),
		challenge.WithCompetitionRepo(repos.compRepo),
		challenge.WithTeamRepo(repos.teamRepo),
		challenge.WithRedis(TestRedis),
		challenge.WithBroadcaster(broadcaster),
		challenge.WithAuditLogRepo(repos.auditLogRepo),
		challenge.WithCrypto(deps.crypto),
	)
	testCache := cache.New(TestRedis)
	solveUC := competition.NewSolveUseCase(repos.solveRepo, repos.challengeRepo, repos.compRepo, repos.userRepo, repos.teamRepo, repos.txRepo, testCache, broadcaster)
	teamUC := team.NewTeamUseCase(repos.teamRepo, repos.userRepo, repos.compRepo, repos.txRepo, TestRedis)
	hintUC := challenge.NewHintUseCase(repos.hintRepo, repos.hintUnlockRepo, repos.awardRepo, repos.txRepo, repos.solveRepo, TestRedis)
	awardUC := team.NewAwardUseCase(repos.awardRepo, repos.txRepo, TestRedis)
	emailUC := email.NewEmailUseCase(repos.userRepo, repos.tokenRepo, &noOpMailer{}, 24*time.Hour, 1*time.Hour, "http://localhost:3000", true)
	statsUC := competition.NewStatisticsUseCase(repos.statsRepo, testCache)
	submissionUC := competition.NewSubmissionUseCase(repos.submissionRepo)
	tagUC := challenge.NewTagUseCase(repos.tagRepo)
	fieldUC := settings.NewFieldUseCase(repos.fieldRepo)
	pageUC := page.NewPageUseCase(repos.pageRepo)
	bracketUC := competition.NewBracketUseCase(repos.bracketRepo)
	ratingUC := competition.NewRatingUseCase(repos.ratingRepo, repos.solveRepo, repos.teamRepo)
	notifUC := notification.NewNotificationUseCase(repos.notificationRepo)
	apiTokenUC := user.NewAPITokenUseCase(repos.apiTokenRepo)
	backupUC := competition.NewBackupUseCase(repos.compRepo, repos.challengeRepo, repos.hintRepo, repos.teamRepo, repos.userRepo, repos.awardRepo, repos.solveRepo, repos.fileRepo, repos.backupRepo, fileStorage, repos.txRepo, deps.logger)
	settingsUC := settings.NewSettingsUseCase(repos.appSettingsRepo, repos.auditLogRepo, TestRedis)
	dynamicConfigUC := competition.NewDynamicConfigUseCase(repos.configRepo, repos.auditLogRepo)
	commentUC := challenge.NewCommentUseCase(repos.commentRepo, repos.challengeRepo)
	ws := wsV1.NewController(hub, deps.logger, []string{"*"})
	fileUC := challenge.NewFileUseCase(repos.fileRepo, fileStorage, 1*time.Hour)
	return &testUseCases{
		user: userUC, challenge: challengeUC, solve: solveUC, team: teamUC, competition: compUC,
		hint: hintUC, award: awardUC, email: emailUC, file: fileUC, stats: statsUC, backup: backupUC,
		settings: settingsUC, ws: ws, submissionUC: submissionUC, tagUC: tagUC, fieldUC: fieldUC,
		pageUC: pageUC, bracketUC: bracketUC, ratingUC: ratingUC, notifUC: notifUC, apiTokenUC: apiTokenUC,
		dynamicConfigUC: dynamicConfigUC, commentUC: commentUC,
	}
}

func initTestUseCases(deps *testDeps) (*testUseCases, string, error) {
	repos := initTestRepos()
	tempStorageDir, fileStorage, hub, err := initTestStorageAndHub()
	if err != nil {
		return nil, "", err
	}
	uc := buildTestUseCases(deps, repos, fileStorage, hub)
	return uc, tempStorageDir, nil
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

	deps := &helper.ServerDeps{
		Challenge: helper.ChallengeDeps{
			ChallengeUC: uc.challenge, HintUC: uc.hint, FileUC: uc.file, TagUC: uc.tagUC, CommentUC: uc.commentUC,
		},
		Team:  helper.TeamDeps{TeamUC: uc.team, AwardUC: uc.award},
		User:  helper.UserDeps{UserUC: uc.user, EmailUC: uc.email, APITokenUC: uc.apiTokenUC},
		Comp:  helper.CompetitionDeps{CompetitionUC: uc.competition, SolveUC: uc.solve, StatsUC: uc.stats, SubmissionUC: uc.submissionUC, BracketUC: uc.bracketUC, RatingUC: uc.ratingUC},
		Admin: helper.AdminDeps{BackupUC: uc.backup, SettingsUC: uc.settings, DynamicConfigUC: uc.dynamicConfigUC, FieldUC: uc.fieldUC, PageUC: uc.pageUC, NotifUC: uc.notifUC},
		Infra: helper.InfraDeps{JWTService: jwtService, RedisClient: TestRedis, WSController: uc.ws, Validator: validatorService, Logger: l},
	}
	r.Route("/api/v1", func(apiRouter chi.Router) {
		v1.NewRouter(apiRouter, deps, 100, 1*time.Minute, false)

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
