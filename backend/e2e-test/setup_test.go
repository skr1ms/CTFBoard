package e2e_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-wskit"

	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	v1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	wsV1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
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

	"github.com/wahrwelt-kit/go-pgkit/migrator/goose"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/testutil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
)

// teamBracketGetter adapts TeamRepository for scoreboard cache (bracket by team).
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
	TestPool           *pgxpool.Pool
	TestRedis          *redis.Client
	testPort           string
	e2eConnStr         string
	testRateLimitCache *restapimiddleware.RateLimitConfigCache
)

// Mocks
type noOpMailer struct{}

// Send is a no-op for e2e tests (no real email sent).
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

// GetTestBaseURL returns the base URL of the test server (e.g. http://localhost:PORT).
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
			ratings, challenges, teams, users, competition
			RESTART IDentity CASCADE`)
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
				24, 1, 500000, 1,
				'public', TRUE,
				10000, 10000,
				100000, 10000,
				10000, 10000,
				10000, 10000,
				10000, 10000,
				now()
			) ON CONFLICT (id) DO UPDATE SET
				submit_limit_per_user = 500000,
				submit_limit_duration_min = 1,
				rate_limit_login_per_minute = 10000,
				rate_limit_register_per_minute = 10000,
				rate_limit_forgot_password_per_minute = 100000,
				rate_limit_reset_password_per_minute = 10000,
				rate_limit_logout_per_minute = 10000,
				rate_limit_refresh_per_minute = 10000,
				rate_limit_scoreboard_per_minute = 10000,
				rate_limit_general_ip_per_minute = 10000,
				rate_limit_verify_email_per_minute = 10000,
				rate_limit_oauth_callback_per_minute = 10000,
				updated_at = NOW()`)
		if err != nil {
			return err
		}
		if TestRedis != nil {
			_ = TestRedis.Del(ctx, cache.KeyAppSettings)
			for _, pattern := range []string{"limiter:*", "e2e:*"} {
				var cursor uint64
				for {
					keys, next, scanErr := TestRedis.Scan(ctx, cursor, pattern, 100).Result()
					if scanErr != nil {
						break
					}
					if len(keys) > 0 {
						_ = TestRedis.Del(ctx, keys...)
					}
					cursor = next
					if cursor == 0 {
						break
					}
				}
			}
		}
		return nil
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

func resetAppSettings() {
	ctx := context.Background()
	_, err := TestPool.Exec(ctx, `UPDATE app_settings SET
		submit_limit_per_user = 500000, submit_limit_duration_min = 1,
		rate_limit_login_per_minute = 10000, rate_limit_register_per_minute = 10000,
		rate_limit_forgot_password_per_minute = 100000, rate_limit_reset_password_per_minute = 10000,
		rate_limit_logout_per_minute = 10000, rate_limit_refresh_per_minute = 10000,
		rate_limit_scoreboard_per_minute = 10000, rate_limit_general_ip_per_minute = 10000,
		rate_limit_verify_email_per_minute = 10000, rate_limit_oauth_callback_per_minute = 10000,
		updated_at = NOW() WHERE id = 1`)
	if err != nil {
		panic("resetAppSettings: " + err.Error())
	}
	if TestRedis != nil {
		_ = TestRedis.Del(ctx, "app_settings")
	}
	if testRateLimitCache != nil {
		testRateLimitCache.Invalidate()
	}
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

func WithCompetitionTimes(t *testing.T, start, end time.Time, freeze *time.Time) {
	t.Helper()
	t.Cleanup(resetCompetitionToActive)
	setCompetitionTimes(start, end, freeze)
}

// setCompetitionPaused sets competition id=1 is_paused in DB and clears competition cache.
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

// invalidateScoreboardCache clears scoreboard and frozen scoreboard keys in Redis.
func invalidateScoreboardCache(ctx context.Context) {
	if TestRedis == nil {
		return
	}
	c := cachekit.New(TestRedis)
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

// setupInfrastructure starts DB and Redis (testcontainers or external) and returns cleanup.
func setupInfrastructure(ctx context.Context) (func(), error) {
	if os.Getenv("USE_EXTERNAL_DB") == "true" {
		fmt.Println("Using EXTERNAL infrastructure (CI mode)...")
		return setupExternalInfra(ctx)
	}
	fmt.Println("Using TESTCONTAINERS infrastructure...")
	return setupTestContainers(ctx)
}

func setupTestContainers(ctx context.Context) (func(), error) {
	postgresC, connStr, err := testutil.StartPostgres(ctx,
		testutil.PostgresWithUser(string(domain.RoleUser)),
		testutil.PostgresWithPassword("password"),
		testutil.PostgresWithStartupTimeout(120*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}
	e2eConnStr = connStr

	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}
	poolCfg.MaxConns = 20
	TestPool, err = pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	if err := TestPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	var redisCleanup func()
	TestRedis, redisCleanup, err = testutil.StartRedisClient(ctx)
	if err != nil {
		if termErr := postgresC.Terminate(ctx); termErr != nil {
			fmt.Printf("postgres terminate on cleanup: %v\n", termErr)
		}
		return nil, fmt.Errorf("failed to start redis: %w", err)
	}

	cleanup := func() {
		fmt.Println("Cleaning up containers...")
		TestPool.Close()
		redisCleanup()
		if err := postgresC.Terminate(ctx); err != nil {
			fmt.Printf("postgres terminate: %v\n", err)
		}
	}
	return cleanup, nil
}

// setupExternalInfra connects to existing Postgres and Redis from env (CI) and returns cleanup.
func setupExternalInfra(ctx context.Context) (func(), error) {
	// PostgreSQL Setup
	dbUser := testutil.GetEnv("POSTGRES_USER", "test_user")
	dbPass := testutil.GetEnv("POSTGRES_PASSWORD", "test_password")
	dbHost := testutil.GetEnv("POSTGRES_HOST", "postgres")
	dbPort := testutil.GetEnv("POSTGRES_PORT", "5432")
	dbName := testutil.GetEnv("POSTGRES_DB", "test_board")

	e2eConnStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	poolCfg, err := pgxpool.ParseConfig(e2eConnStr)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = 20
	TestPool, err = pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 15 * time.Second
	if err := backoff.Retry(func() error { return TestPool.Ping(ctx) }, backoff.WithContext(bo, ctx)); err != nil {
		return nil, fmt.Errorf("external db ping failed: %w", err)
	}

	// Redis Setup
	redisHost := testutil.GetEnv("REDIS_HOST", "redis")
	redisPort := testutil.GetEnv("REDIS_PORT", "6379")
	redisPassword := testutil.GetEnv("REDIS_PASSWORD", "")

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

// Server setup

// testDeps holds logger, validator, JWT, crypto for e2e server.
type testDeps struct {
	logger    logkit.Logger
	validator validator.Validator
	jwt       *jwtkit.JWTService
	crypto    *crypto.CryptoService
}

// testRepos holds all persistent repositories and transaction manager for e2e.
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
	ratingRepo       *persistent.RatingRepo
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

// testUseCases holds all application use cases and settings repo for e2e server.
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
	ratingUC           *challenge.RatingUseCase
	bracketUC          *competition.BracketUseCase
	notifUC            usecase.NotificationUseCase
	apiTokenUC         usecase.APITokenUseCase
	competitionParamUC *competition.CompetitionParamUseCase
	commentUC          *challenge.CommentUseCase
	trackingUC         *user.TrackingUseCase
	SettingsRepo       repo.SettingsRepository
}

// startTestServer builds deps, use cases, router, starts HTTP server on random port; returns shutdown and temp dir cleanup.
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

	baseURL := "http://localhost:" + testPort
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/api/v1/competition/status", nil)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

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
	l, err := logkit.New(logkit.WithLevel(logkit.ErrorLevel), logkit.WithOutput(logkit.ConsoleOutput))
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}
	validatorService, err := validator.New()
	if err != nil {
		panic("e2e: failed to create validator: " + err.Error())
	}
	jwtRevoker := jwtkit.NewRedisRevocationStore(TestRedis)
	jwtService, err := jwtkit.NewJWTService(jwtkit.Config{
		AccessKeys:  []jwtkit.KeyEntry{{Kid: "0", Secret: []byte("test-access-secret-min-32-bytes!")}},
		RefreshKeys: []jwtkit.KeyEntry{{Kid: "0", Secret: []byte("test-refresh-secret-min32-bytes!")}},
		AccessTTL:   24 * time.Hour,
		RefreshTTL:  72 * time.Hour,
		Issuer:      "e2e-issuer",
		Revoker:     jwtRevoker,
	})
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

// initTestRepos creates all persistent repos and transaction manager from TestPool.
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
		ratingRepo:       persistent.NewRatingRepo(TestPool),
		bracketRepo:      persistent.NewBracketRepo(TestPool),
		notificationRepo: persistent.NewNotificationRepo(TestPool),
		apiTokenRepo:     persistent.NewAPITokenRepo(TestPool),
		paramRepo:        persistent.NewCompetitionParamRepo(TestPool),
		commentRepo:      persistent.NewCommentRepo(TestPool),
		trackingRepo:     persistent.NewTrackingRepo(TestPool),
	}
}

func initTestStorageAndHub() (string, storage.Provider, *wskit.Hub, error) {
	tempStorageDir, err := os.MkdirTemp("", "astroctfb-e2e-storage")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp storage dir: %w", err)
	}
	fileStorage, err := storage.NewFilesystemProvider(tempStorageDir)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create storage provider: %w", err)
	}
	ctx := context.Background()
	hub := wskit.NewHub(
		wskit.WithRedis(TestRedis, "astroctfb:events"),
		wskit.WithOnConnect(func(c *wskit.Client) {
			data, err := json.Marshal(wskit.NewEvent(websocket.EventTypeConnected, nil))
			if err == nil {
				c.Send(data)
			}
		}),
	)
	go hub.Run(ctx)
	go hub.SubscribeToRedis(ctx)
	return tempStorageDir, fileStorage, hub, nil
}

func buildTestUseCases(deps *testDeps, repos *testRepos, fileStorage storage.Provider, hub *wskit.Hub) *testUseCases {
	fieldValidator := settings.NewFieldValidator(repos.fieldRepo)
	broadcaster := websocket.NewBroadcaster(hub)
	testCache := cachekit.New(TestRedis)
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
		ChallengeRepo:   repos.challengeRepo,
		ScoreboardCache: scoreboardCache,
		Logger:          deps.logger,
	})
	compUC := competition.NewCompetitionUseCase(competition.CompetitionDeps{
		CompetitionRepo: repos.compRepo,
		AuditLogRepo:    repos.auditLogRepo,
		TM:              repos.tm,
		Redis:           &cachekit.RedisKeyValueStore{Client: TestRedis},
		ScoreboardCache: scoreboardCache,
		Logger:          deps.logger,
	})
	challengeUC := challenge.NewChallengeUseCase(challenge.ChallengeDeps{
		ChallengeRepo:   repos.challengeRepo,
		TagRepo:         repos.tagRepo,
		SolveRepo:       repos.solveRepo,
		SubmissionRepo:  repos.submissionRepo,
		TM:              repos.tm,
		CompRepo:        repos.compRepo,
		TeamRepo:        repos.teamRepo,
		UserRepo:        repos.userRepo,
		ScoreboardCache: scoreboardCache,
		Broadcaster:     broadcaster,
		AuditLogRepo:    repos.auditLogRepo,
		Crypto:          deps.crypto,
	})
	ratingUC := challenge.NewRatingUseCase(challenge.RatingDeps{
		ChallengeRepo: repos.challengeRepo,
		SolveRepo:     repos.solveRepo,
		RatingRepo:    repos.ratingRepo,
		TM:            repos.tm,
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
		Redis:        &cachekit.RedisKeyValueStore{Client: TestRedis},
		CompRepo:     repos.compRepo,
		Logger:       deps.logger,
	})
	competitionParamUC := competition.NewCompetitionParamUseCase(context.Background(), competition.CompetitionParamDeps{
		Repo:         repos.paramRepo,
		AuditLogRepo: repos.auditLogRepo,
		TM:           repos.tm,
		Logger:       deps.logger,
	})
	commentUC := challenge.NewCommentUseCase(challenge.CommentDeps{CommentRepo: repos.commentRepo, ChallengeRepo: repos.challengeRepo, TM: repos.tm})
	trackingUC := user.NewTrackingUseCase(user.TrackingDeps{TrackingRepo: repos.trackingRepo})
	ws := wsV1.NewController(hub, deps.logger, []string{"localhost:*"})
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
		pageUC: pageUC, ratingUC: ratingUC, bracketUC: bracketUC, notifUC: notifUC, apiTokenUC: apiTokenUC,
		competitionParamUC: competitionParamUC, commentUC: commentUC, trackingUC: trackingUC,
		SettingsRepo: repos.SettingsRepo,
	}
}

// initTestUseCases initializes repos, storage, hub, and builds all use cases; returns temp dir path.
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
func setupTestRouter(ctx context.Context, l logkit.Logger, uc *testUseCases, validatorService validator.Validator, jwtService *jwtkit.JWTService, _ string) *chi.Mux {
	r := chi.NewRouter()
	timeoutMW := kitMiddleware.Timeout(60 * time.Second)
	r.Use(kitMiddleware.RequestID(), middleware.RealIP, kitMiddleware.Recoverer(l))
	r.Use(func(next http.Handler) http.Handler {
		withTimeout := timeoutMW(next)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/ws") {
				next.ServeHTTP(w, req)
				return
			}
			withTimeout.ServeHTTP(w, req)
		})
	})
	r.Use(kitMiddleware.Logger(l, nil))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK")) //nolint:errcheck // best-effort health
	})

	forgotLimiter, err := restapimiddleware.NewPerKeyRateLimiter(TestRedis, "e2e:forgot", 20, 24*time.Hour)
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
			ChallengeUC: uc.challenge, HintUC: uc.hint, FileUC: uc.file, TagUC: uc.tagUC, CommentUC: uc.commentUC, RatingUC: uc.ratingUC,
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
			StructuredLogger:              false,
			DebugEnabled:                  false,
			ForgotPasswordRateLimiter:     forgotLimiter,
			ResendVerificationRateLimiter: resendLimiter,
			ResetPasswordTokenRateLimiter: resetTokenLimiter,
		},
	}
	testRateLimitCache = restapimiddleware.NewRateLimitConfigCache(1 * time.Second)
	deps.Infra.RateLimitConfigCache = testRateLimitCache
	r.Route("/api/v1", func(apiRouter chi.Router) {
		v1.NewRouter(ctx, apiRouter, deps, false, testRateLimitCache)
	})

	return r
}

// Utils

// getEnv returns os.LookupEnv(key) or fallback.
