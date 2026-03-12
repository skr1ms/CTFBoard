package e2e_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	v1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	wsV1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
)

type teamBracketGetter struct {
	r repo.TeamRepository
}

func (g *teamBracketGetter) GetTeamBracketID(ctx context.Context, teamID uuid.UUID) (*uuid.UUID, error) {
	team, err := g.r.GetByID(ctx, teamID)
	if err != nil || team == nil {
		return nil, err
	}
	return team.BracketID, nil
}

var (
	TestPool  *pgxpool.Pool
	TestRedis *redis.Client
	testPort  string
)

// Mocks
type noOpMailer struct{}

func (m *noOpMailer) Send(context.Context, mailer.Message) error {
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

	// One-time clean slate for parallel tests (no per-test truncation).
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

func GetTestBaseURL() string {
	return fmt.Sprintf("http://localhost:%s", testPort)
}

//nolint:thelper // test helper: truncates and re-seeds all tables; t can be nil so t.Helper() is guarded
func truncateE2EDB(ctx context.Context, t *testing.T) error {
	if t != nil {
		t.Helper()
	}
	truncateAndSeed := func() error {
		if TestPool == nil {
			return fmt.Errorf("TestPool is not initialized")
		}
		_, err := TestPool.Exec(ctx, `TRUNCATE TABLE
			configs, comments, api_tokens,
			field_values, fields, brackets, pages, user_notifications, notifications,
			submissions, challenge_tags, tags, audit_logs, team_audit_log, app_settings,
			solutions, files, verification_tokens, awards, hint_unlocks, hints, solves,
			challenges, teams, users, competition
			RESTART IDENTITY CASCADE`)
		if err != nil {
			return err
		}
		_, err = TestPool.Exec(ctx, `INSERT INTO competition (id, name, is_paused, is_public, mode, allow_team_switch, min_team_size, max_team_size, start_time, end_time)
			VALUES (1, 'CTF Competition', FALSE, TRUE, 'flexible', TRUE, 1, 10, NULL, NULL)
			ON CONFLICT (id) DO UPDATE set name = EXCLUDED.name, is_paused = EXCLUDED.is_paused, is_public = EXCLUDED.is_public, mode = EXCLUDED.mode, allow_team_switch = EXCLUDED.allow_team_switch, min_team_size = EXCLUDED.min_team_size, max_team_size = EXCLUDED.max_team_size, start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, updated_at = NOW()`)
		if err != nil {
			return err
		}
		_, err = TestPool.Exec(ctx, `INSERT INTO app_settings (
				id, app_name, verify_emails, frontend_url, cors_origins,
				resend_enabled, resend_from_email, resend_from_name,
				verify_ttl_hours, reset_ttl_hours, submit_limit_per_user, submit_limit_duration_min,
				scoreboard_visible, registration_open,
				rate_limit_login_per_minute, rate_limit_register_per_minute,
				rate_limit_forgot_password_per_minute, rate_limit_reset_password_per_minute,
				rate_limit_logout_per_minute, rate_limit_refresh_per_minute,
				rate_limit_scoreboard_per_minute, rate_limit_general_ip_per_minute,
				rate_limit_verify_email_per_minute, rate_limit_oauth_callback_per_minute,
				updated_at
			) VALUES (
				1, 'AstroCTFb', TRUE, 'http://localhost:3000', 'http://localhost:3000,http://localhost:5173',
				FALSE, 'noreply@astroctfb.local', 'AstroCTFb',
				24, 1, 10, 1,
				'public', TRUE,
				1000, 1000,
				1000, 1000,
				1000, 1000,
				1000, 1000,
				1000, 1000,
				now()
			) ON CONFLICT (id) DO UPDATE SET
				rate_limit_login_per_minute = 1000,
				rate_limit_register_per_minute = 1000,
				rate_limit_forgot_password_per_minute = 1000,
				rate_limit_reset_password_per_minute = 1000,
				rate_limit_logout_per_minute = 1000,
				rate_limit_refresh_per_minute = 1000,
				rate_limit_scoreboard_per_minute = 1000,
				rate_limit_general_ip_per_minute = 1000,
				rate_limit_verify_email_per_minute = 1000,
				rate_limit_oauth_callback_per_minute = 1000,
				updated_at = NOW()`)
		return err
	}
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 50 * time.Millisecond
	bo.MaxElapsedTime = 10 * time.Second
	err := backoff.Retry(func() error {
		err := truncateAndSeed()
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "40P01" {
				return err // retry on deadlock
			}
			return backoff.Permanent(err)
		}
		return nil
	}, backoff.WithContext(bo, ctx))
	if err != nil {
		if t != nil {
			t.Fatalf("truncate db: %v", err)
		}
		return err
	}
	return nil
}

// resetCompetitionToActive sets competition id=1 to active (start in past, end in future, not paused, no freeze).
// Use in t.Cleanup for tests that mutate global competition state.
func resetCompetitionToActive() {
	ctx := context.Background()
	now := time.Now().UTC()
	_, err := TestPool.Exec(ctx, `UPDATE competition SET is_paused = FALSE, start_time = $1, end_time = $2, freeze_time = NULL, updated_at = now() WHERE id = 1`,
		now.Add(-1*time.Hour), now.Add(24*time.Hour))
	if err != nil {
		panic("resetCompetitionToActive: " + err.Error())
	}
	_ = TestRedis.Del(ctx, "competition")
}

// setCompetitionTimes sets competition id=1 times directly in DB, bypassing API restrictions.
// Use when parallel tests may have activated the competition between resetCompetitionToNotStarted
// and the API PUT call. Pass nil for freezeTime to clear it.
func setCompetitionTimes(startTime, endTime time.Time, freezeTime *time.Time) {
	ctx := context.Background()
	_, err := TestPool.Exec(ctx, `UPDATE competition SET start_time = $1, end_time = $2, freeze_time = $3, is_paused = FALSE, updated_at = now() WHERE id = 1`,
		startTime, endTime, freezeTime)
	if err != nil {
		panic("setCompetitionTimes: " + err.Error())
	}
	_ = TestRedis.Del(ctx, "competition")
}

func setCompetitionPaused(paused bool) {
	ctx := context.Background()
	var err error
	if paused {
		_, err = TestPool.Exec(ctx, `UPDATE competition SET is_paused = TRUE, paused_at = NOW(), updated_at = now() WHERE id = 1`)
	} else {
		_, err = TestPool.Exec(ctx, `UPDATE competition SET is_paused = FALSE, paused_at = NULL, updated_at = now() WHERE id = 1`)
	}
	if err != nil {
		panic("setCompetitionPaused: " + err.Error())
	}
	_ = TestRedis.Del(ctx, "competition")
}

func invalidateScoreboardCache(ctx context.Context) {
	if TestRedis == nil {
		return
	}
	c := cache.New(TestRedis)
	if err := c.Del(ctx, cache.KeyScoreboard); err != nil {
		return
	}
	if err := c.DeleteByPrefix(ctx, cache.KeyScoreboardFrozenPrefix); err != nil {
		return
	}
	if err := c.DeleteByPrefix(ctx, cache.KeyScoreboardBracketPrefix); err != nil {
		return
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
		postgres.WithUsername(string(entity.RoleUser)),
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
		if !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}

		raw, err := os.ReadFile(filepath.Join(migrationsDir, f.Name()))
		if err != nil {
			return err
		}

		if _, err := pool.Exec(ctx, extractGooseUp(string(raw))); err != nil {
			if !isIgnorableDBError(err) {
				fmt.Printf("Warn: migration error in %s: %v\n", f.Name(), err)
			}
		}
	}

	if _, err := TestPool.Exec(ctx, "UPDATE competition SET start_time = $1 WHERE id = 1", time.Now().Add(-24*time.Hour)); err != nil {
		return fmt.Errorf("update competition start_time: %w", err)
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
			result = append(result, strings.ReplaceAll(line, " CONCURRENTLY", ""))
		}
	}
	return strings.Join(result, "\n")
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
	SettingsRepo     *persistent.SettingsRepo
	auditLogRepo     *persistent.AuditLogRepo
	awardRepo        *persistent.AwardRepo
	backupRepo       *persistent.BackupRepo
	bracketRepo      *persistent.BracketRepo
	challengeRepo    *persistent.ChallengeRepo
	commentRepo      *persistent.CommentRepo
	compRepo         *persistent.CompetitionRepo
	paramRepo        *persistent.CompetitionParamRepo
	fieldRepo        *persistent.FieldRepo
	fieldValueRepo   *persistent.FieldValueRepo
	fileRepo         *persistent.FileRepo
	hintRepo         *persistent.HintRepo
	notificationRepo *persistent.NotificationRepo
	pageRepo         *persistent.PageRepo
	solveRepo        *persistent.SolveRepo
	statsRepo        *persistent.StatisticsRepo
	submissionRepo   *persistent.SubmissionRepo
	tagRepo          *persistent.TagRepo
	teamRepo         *persistent.TeamRepo
	tokenRepo        *persistent.VerificationTokenRepo
	trackingRepo     *persistent.TrackingRepo
	tm               repo.TransactionManager
	userRepo         *persistent.UserRepo
}

type testUseCases struct {
	user               *user.UserUseCase
	team               *team.TeamUseCase
	award              *team.AwardUseCase
	email              *email.EmailUseCase
	challenge          *challenge.ChallengeUseCase
	hint               *challenge.HintUseCase
	file               *challenge.FileUseCase
	solve              *competition.SolveUseCase
	competition        *competition.CompetitionUseCase
	backup             *backup.BackupUseCase
	stats              *competition.StatisticsUseCase
	settings           *settings.SettingsUseCase
	ws                 *wsV1.Controller
	submissionUC       *competition.SubmissionUseCase
	tagUC              *challenge.TagUseCase
	fieldUC            *settings.FieldUseCase
	pageUC             *page.PageUseCase
	bracketUC          *competition.BracketUseCase
	notifUC            usecase.NotificationUseCase
	apiTokenUC         usecase.APITokenUseCase
	competitionParamUC *competition.CompetitionParamUseCase
	commentUC          *challenge.CommentUseCase
	trackingUC         *user.TrackingUseCase
	SettingsRepo       repo.SettingsRepository
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

	ctx := context.Background()
	r := setupTestRouter(ctx, deps.logger, useCases, deps.validator, deps.jwt, tempStorageDir)

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
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	validatorService, err := validator.New()
	if err != nil {
		panic("e2e: failed to create validator: " + err.Error())
	}
	jwtRevoker := jwt.NewRedisRevocationStore(TestRedis)
	jwtService, err := jwt.NewJWTService(
		[]jwt.KeyEntry{{Kid: "0", Secret: "test-access-secret-min-32-bytes!"}},
		[]jwt.KeyEntry{{Kid: "0", Secret: "test-refresh-secret-min32-bytes!"}},
		24*time.Hour, 72*time.Hour, jwtRevoker, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init JWT service: %w", err)
	}
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
	tm := persistent.NewTransactionManager(TestPool)
	return &testRepos{
		userRepo:         persistent.NewUserRepo(TestPool),
		challengeRepo:    persistent.NewChallengeRepo(TestPool),
		solveRepo:        persistent.NewSolveRepo(TestPool),
		teamRepo:         persistent.NewTeamRepo(TestPool),
		compRepo:         persistent.NewCompetitionRepo(TestPool),
		hintRepo:         persistent.NewHintRepo(TestPool),
		awardRepo:        persistent.NewAwardRepo(TestPool),
		tm:               tm,
		tokenRepo:        persistent.NewVerificationTokenRepo(TestPool),
		auditLogRepo:     persistent.NewAuditLogRepo(TestPool),
		statsRepo:        persistent.NewStatisticsRepo(TestPool),
		fileRepo:         persistent.NewFileRepo(TestPool),
		backupRepo:       persistent.NewBackupRepo(TestPool),
		SettingsRepo:     persistent.NewSettingsRepo(TestPool),
		tagRepo:          persistent.NewTagRepo(TestPool),
		fieldRepo:        persistent.NewFieldRepo(TestPool),
		fieldValueRepo:   persistent.NewFieldValueRepo(TestPool),
		submissionRepo:   persistent.NewSubmissionRepo(TestPool),
		pageRepo:         persistent.NewPageRepo(TestPool),
		bracketRepo:      persistent.NewBracketRepo(TestPool),
		notificationRepo: persistent.NewNotificationRepo(TestPool),
		apiTokenRepo:     persistent.NewAPITokenRepo(TestPool),
		paramRepo:        persistent.NewCompetitionParamRepo(TestPool),
		commentRepo:      persistent.NewCommentRepo(TestPool),
		trackingRepo:     persistent.NewTrackingRepo(TestPool),
	}
}

func initTestStorageAndHub() (string, storage.Provider, *websocket.Hub, error) {
	tempStorageDir, err := os.MkdirTemp("", "astroctfb-e2e-storage")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp storage dir: %w", err)
	}
	fileStorage, err := storage.NewFilesystemProvider(tempStorageDir)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create storage provider: %w", err)
	}
	ctx := context.Background()
	hub := websocket.NewHub(TestRedis, "astroctfb:events")
	go hub.Run(ctx)
	go hub.SubscribeToRedis(ctx)
	return tempStorageDir, fileStorage, hub, nil
}

func buildTestUseCases(deps *testDeps, repos *testRepos, fileStorage storage.Provider, hub *websocket.Hub) *testUseCases {
	fieldValidator := settings.NewFieldValidator(repos.fieldRepo)
	broadcaster := websocket.NewBroadcaster(hub)
	testCache := cache.New(TestRedis)
	scoreboardCache := cache.NewScoreboardCacheService(testCache, &teamBracketGetter{repos.teamRepo})
	competitionGuard := competition.NewGuard(repos.compRepo)
	userCacheSvc := cache.NewUserCacheService(testCache)
	teamUC := team.NewTeamUseCase(team.TeamDeps{
		TeamRepo:           repos.teamRepo,
		UserRepo:           repos.userRepo,
		SolveRepo:          repos.solveRepo,
		SubmissionRepo:     repos.submissionRepo,
		AwardRepo:          repos.awardRepo,
		CompRepo:           repos.compRepo,
		SettingsGetter:     repos.SettingsRepo,
		ChallengeRepo:      repos.challengeRepo,
		TM:                 repos.tm,
		Guard:              competitionGuard,
		ScoreboardCache:    scoreboardCache,
		UserCache:          userCacheSvc,
		HintRepo:           repos.hintRepo,
		DefaultMaxTeamSize: 10,
	})
	emailUC := email.NewEmailUseCase(email.EmailDeps{
		UserRepo: repos.userRepo, TokenRepo: repos.tokenRepo, TM: repos.tm, Mailer: &noOpMailer{},
		VerifyTTL: 24 * time.Hour, ResetTTL: 1 * time.Hour, FrontendURL: "http://localhost:3000", Enabled: true,
	})

	userUC := user.NewUserUseCase(user.UserDeps{
		UserRepo:        repos.userRepo,
		TeamRepo:        repos.teamRepo,
		SolveRepo:       repos.solveRepo,
		SubmissionRepo:  repos.submissionRepo,
		AwardRepo:       repos.awardRepo,
		TM:              repos.tm,
		JWTService:      deps.jwt,
		FieldValidator:  fieldValidator,
		FieldValueRepo:  repos.fieldValueRepo,
		SettingsRepo:    repos.SettingsRepo,
		EmailSender:     emailUC,
		FailedLogin:     nil,
		CompRepo:        repos.compRepo,
		SoloTeamCreator: teamUC,
		UserCache:       userCacheSvc,
	})
	compUC := competition.NewCompetitionUseCase(competition.CompetitionDeps{
		CompetitionRepo: repos.compRepo,
		AuditLogRepo:    repos.auditLogRepo,
		TM:              repos.tm,
		Redis:           &cache.RedisKeyValueStore{Client: TestRedis},
		ScoreboardCache: scoreboardCache,
		Logger:          deps.logger,
	})
	challengeUC := challenge.NewChallengeUseCase(challenge.ChallengeDeps{
		ChallengeRepo:   repos.challengeRepo,
		TagRepo:         repos.tagRepo,
		SolveRepo:       repos.solveRepo,
		TM:              repos.tm,
		CompRepo:        repos.compRepo,
		TeamRepo:        repos.teamRepo,
		ScoreboardCache: scoreboardCache,
		Broadcaster:     broadcaster,
		AuditLogRepo:    repos.auditLogRepo,
		Crypto:          deps.crypto,
	})
	solveUC := competition.NewSolveUseCase(competition.SolveDeps{
		SolveRepo: repos.solveRepo, ChallengeRepo: repos.challengeRepo, CompetitionRepo: repos.compRepo,
		CompetitionUC: compUC,
		UserRepo:      repos.userRepo, TeamRepo: repos.teamRepo, TM: repos.tm,
		Cache: testCache, ScoreboardCache: scoreboardCache, Broadcaster: broadcaster,
	})
	hintUC := challenge.NewHintUseCase(challenge.HintDeps{
		HintRepo: repos.hintRepo, AwardRepo: repos.awardRepo,
		TM: repos.tm, SolveRepo: repos.solveRepo, CompRepo: repos.compRepo, CompGetter: compUC,
		TeamRepo: repos.teamRepo, UserRepo: repos.userRepo,
		ChallengeRepo: repos.challengeRepo, ScoreboardCache: scoreboardCache,
	})
	awardUC := team.NewAwardUseCase(team.AwardDeps{AwardRepo: repos.awardRepo, TeamRepo: repos.teamRepo, TM: repos.tm, ScoreboardCache: scoreboardCache, CompRepo: repos.compRepo})
	statsUC := competition.NewStatisticsUseCase(competition.StatisticsDeps{StatsRepo: repos.statsRepo, Cache: testCache})
	submissionUC := competition.NewSubmissionUseCase(competition.SubmissionDeps{
		SubmissionRepo: repos.submissionRepo,
		CompGetter:     compUC,
	})
	tagUC := challenge.NewTagUseCase(challenge.TagDeps{TagRepo: repos.tagRepo, ChallengeRepo: repos.challengeRepo})
	fieldUC := settings.NewFieldUseCase(settings.FieldDeps{FieldRepo: repos.fieldRepo})
	pageUC := page.NewPageUseCase(page.PageDeps{PageRepo: repos.pageRepo})
	bracketUC := competition.NewBracketUseCase(competition.BracketDeps{BracketRepo: repos.bracketRepo, TM: repos.tm})
	notifUC := notification.NewNotificationUseCase(notification.NotificationDeps{NotifRepo: repos.notificationRepo, Broadcaster: broadcaster})
	apiTokenUC := user.NewAPITokenUseCase(user.APITokenDeps{Repo: repos.apiTokenRepo})
	backupUC := backup.NewBackupUseCase(backup.BackupDeps{
		CompetitionRepo: repos.compRepo, ChallengeRepo: repos.challengeRepo, TagRepo: repos.tagRepo, HintRepo: repos.hintRepo,
		TeamRepo: repos.teamRepo, UserRepo: repos.userRepo, AwardRepo: repos.awardRepo,
		SolveRepo: repos.solveRepo, SubmissionRepo: repos.submissionRepo, FileRepo: repos.fileRepo,
		BackupRepo: repos.backupRepo, SettingsRepo: repos.SettingsRepo, AuditLogRepo: repos.auditLogRepo,
		BracketRepo: repos.bracketRepo, CommentRepo: repos.commentRepo, FieldRepo: repos.fieldRepo, FieldValueRepo: repos.fieldValueRepo,
		Storage: fileStorage, TM: repos.tm, Logger: deps.logger,
	})
	settingsUC := settings.NewSettingsUseCase(settings.SettingsDeps{
		Repo:         repos.SettingsRepo,
		AuditLogRepo: repos.auditLogRepo,
		TM:           repos.tm,
		Redis:        &cache.RedisKeyValueStore{Client: TestRedis},
		CompRepo:     repos.compRepo,
		Logger:       deps.logger,
	})
	competitionParamUC := competition.NewCompetitionParamUseCase(competition.CompetitionParamDeps{
		Repo:         repos.paramRepo,
		AuditLogRepo: repos.auditLogRepo,
		TM:           repos.tm,
		Logger:       deps.logger,
	})
	commentUC := challenge.NewCommentUseCase(challenge.CommentDeps{CommentRepo: repos.commentRepo, ChallengeRepo: repos.challengeRepo, TM: repos.tm})
	trackingUC := user.NewTrackingUseCase(user.TrackingDeps{TrackingRepo: repos.trackingRepo})
	ws := wsV1.NewController(hub, deps.logger, []string{"*"})
	fileUC := challenge.NewFileUseCase(challenge.FileDeps{
		FileRepo:       repos.fileRepo,
		ChallengeRepo:  repos.challengeRepo,
		SolveRepo:      repos.solveRepo,
		Storage:        fileStorage,
		Expiry:         1 * time.Hour,
		DownloadSecret: "test-download-secret",
		BaseURL:        "http://localhost:3000",
	})
	return &testUseCases{
		user: userUC, challenge: challengeUC, solve: solveUC, team: teamUC, competition: compUC,
		hint: hintUC, award: awardUC, email: emailUC, file: fileUC, stats: statsUC, backup: backupUC,
		settings: settingsUC, ws: ws, submissionUC: submissionUC, tagUC: tagUC, fieldUC: fieldUC,
		pageUC: pageUC, bracketUC: bracketUC, notifUC: notifUC, apiTokenUC: apiTokenUC,
		competitionParamUC: competitionParamUC, commentUC: commentUC, trackingUC: trackingUC,
		SettingsRepo: repos.SettingsRepo,
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
func setupTestRouter(ctx context.Context, l logger.Logger, uc *testUseCases, validatorService validator.Validator, jwtService *jwt.JWTService, tempStorageDir string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Timeout(60*time.Second))
	r.Use(restapimiddleware.Logger(l, nil))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK")) //nolint:errcheck // best-effort health
	})

	forgotLimiter, err := restapimiddleware.NewPerKeyRateLimiter(TestRedis, "e2e:forgot", 3, 24*time.Hour)
	if err != nil {
		panic("e2e: failed to create forgot-password rate limiter: " + err.Error())
	}
	resendLimiter, err := restapimiddleware.NewPerKeyRateLimiter(TestRedis, "e2e:resend", 10, 24*time.Hour)
	if err != nil {
		panic("e2e: failed to create resend-verification rate limiter: " + err.Error())
	}
	resetTokenLimiter, err := restapimiddleware.NewPerKeyRateLimiter(TestRedis, "e2e:reset-token", 20, time.Minute)
	if err != nil {
		panic("e2e: failed to create reset-password-token rate limiter: " + err.Error())
	}

	deps := &helper.ServerDeps{
		Challenge: helper.ChallengeDeps{
			ChallengeUC: uc.challenge, HintUC: uc.hint, FileUC: uc.file, TagUC: uc.tagUC, CommentUC: uc.commentUC,
		},
		Team:  helper.TeamDeps{TeamUC: uc.team, AwardUC: uc.award},
		User:  helper.UserDeps{UserUC: uc.user, EmailUC: uc.email, APITokenUC: uc.apiTokenUC, TrackingUC: uc.trackingUC},
		Comp:  helper.CompetitionDeps{CompetitionUC: uc.competition, SolveUC: uc.solve, StatsUC: uc.stats, SubmissionUC: uc.submissionUC, BracketUC: uc.bracketUC},
		Admin: helper.AdminDeps{BackupUC: uc.backup, SettingsUC: uc.settings, CompetitionParamUC: uc.competitionParamUC, FieldUC: uc.fieldUC, PageUC: uc.pageUC, NotifUC: uc.notifUC, SettingsRepo: uc.SettingsRepo},
		Infra: helper.InfraDeps{
			JWTService:                    jwtService,
			RedisClient:                   TestRedis,
			WSController:                  uc.ws,
			Validator:                     validatorService,
			Logger:                        l,
			TrustedProxyCIDRs:             nil,
			ForgotPasswordRateLimiter:     forgotLimiter,
			ResendVerificationRateLimiter: resendLimiter,
			ResetPasswordTokenRateLimiter: resetTokenLimiter,
		},
	}
	r.Route("/api/v1", func(apiRouter chi.Router) {
		rateLimitCache := helper.NewRateLimitConfigCache(30 * time.Second)
		v1.NewRouter(ctx, apiRouter, deps, false, rateLimitCache)

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
