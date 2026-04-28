package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/run"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-pgkit/migrator/goose"
	"github.com/wahrwelt-kit/go-pgkit/postgres"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/seed"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
	"github.com/TakuyaYagam1/AstroCTFb/internal/wire"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
)

const (
	asyncMailerQueueSize = 100
	asyncMailerWorkers   = 2
	shutdownGraceTimeout = 5 * time.Second
)

// Run is the application entry point. It initialises infrastructure in order:
// PostgreSQL pool -> Redis -> goose migrations -> signal context -> storage
// provider -> JWT service -> WebSocket hub -> async mailer -> wire DI graph.
// After wiring, it configures the JWT role-lookup callback, seeds the default
// admin when credentials are present, then runs an oklog/run group that
// combines the HTTP server and a signal-cancel actor with graceful shutdown
// (draining in-flight rate-limit audits and avatar goroutines).
func Run(cfg *config.Config, l logkit.Logger) {
	l.Info("Application initialized", map[string]any{
		"structured_logger": cfg.StructuredLogger,
		"secure_cookies":    cfg.SecureCookies,
		"log_level":         cfg.LogLevel,
		"version":           cfg.Version,
	})

	ctx := context.Background()

	pool, err := postgres.New(ctx, &postgres.Config{
		URL:      cfg.URL,
		MaxConns: cfg.MaxConns,
		MinConns: cfg.MinConns,
	})
	if err != nil {
		l.WithError(err).Error("failed to connect to database")

		return
	}
	defer pool.Close()

	redisClient, err := cache.NewRedisClient(ctx, &cfg.Redis)
	if err != nil {
		l.WithError(err).Error("failed to connect to redis")

		return
	}

	defer func() {
		err := redisClient.Close()
		if err != nil {
			l.WithError(err).Error("failed to close redis connection")
		}
	}()

	if err := goose.Run(ctx, cfg.URL, cfg.MigrationsPath); err != nil {
		l.WithError(err).Error("failed to run migrations")

		return
	}

	reconcileSettings(ctx, cfg, pool, l)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	storageProvider, err := provideStorage(ctx, cfg, l)
	if err != nil {
		l.WithError(err).Error("failed to create storage provider")

		return
	}

	if closer, ok := storageProvider.(interface{ Close() error }); ok {
		defer func() {
			err := closer.Close()
			if err != nil {
				l.WithError(err).Error("failed to close storage provider")
			}
		}()
	}

	jwtRevoker := jwtkit.NewRedisRevocationStore(redisClient)

	accessKeys := make([]jwtkit.KeyEntry, len(cfg.AccessKeys))
	for i, k := range cfg.AccessKeys {
		accessKeys[i] = jwtkit.KeyEntry{Kid: k.Kid, Secret: []byte(k.Secret)}
	}

	refreshKeys := make([]jwtkit.KeyEntry, len(cfg.RefreshKeys))
	for i, k := range cfg.RefreshKeys {
		refreshKeys[i] = jwtkit.KeyEntry{Kid: k.Kid, Secret: []byte(k.Secret)}
	}

	jwtService, err := jwtkit.NewJWTService(jwtkit.Config{
		AccessKeys:  accessKeys,
		RefreshKeys: refreshKeys,
		AccessTTL:   cfg.AccessTTL,
		RefreshTTL:  cfg.RefreshTTL,
		Issuer:      cfg.Issuer,
		Revoker:     jwtRevoker,
	})
	if err != nil {
		l.WithError(err).Error("failed to create JWT service")

		return
	}

	wsHub := wskit.NewHub(
		wskit.WithRedis(redisClient, cache.PubSubScoreboard),
		wskit.WithOnTimeout(func(op string) { l.Warn("websocket hub operation timed out", logkit.Fields{"op": op}) }),
		wskit.WithOnConnect(func(sub wskit.Subscriber) {
			client, ok := sub.(*wskit.Client)
			if !ok {
				return
			}

			data, err := json.Marshal(wskit.NewEvent(websocket.EventTypeConnected, nil))
			if err == nil {
				client.Send(data)
			}
		}),
	)
	go wsHub.Run(ctx)
	go wsHub.SubscribeToRedis(ctx)

	resendMailer := mailer.New(mailer.Config{APIKey: cfg.APIKey, FromEmail: cfg.FromEmail, FromName: cfg.FromName})
	asyncMailer := mailer.NewAsyncMailer(resendMailer, asyncMailerQueueSize, asyncMailerWorkers, l)

	asyncMailer.Start()
	defer asyncMailer.Stop()

	app, err := wire.InitializeApp(ctx, cfg, l, pool, redisClient, storageProvider, jwtService, wsHub, asyncMailer)
	if err != nil {
		l.WithError(err).Error("failed to initialize app")

		return
	}

	jwtService.SetUserRoleLookup(func(ctx context.Context, userID uuid.UUID) (string, error) {
		u, err := app.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return "", fmt.Errorf("SetUserRoleLookup - GetByID: %w", err)
		}

		// Banned users are still allowed to refresh so the frontend can load
		// /auth/me, read ban_status, and redirect to /banned with the ban reason.
		// The AuthGuard redirects all banned users to /banned; individual usecases
		// enforce the ban for CTF operations (submit, hint unlock, team ops, etc.).
		if u.WasInBannedTeam && u.Role != domain.RoleAdmin {
			return "", apperr.ErrUserBanned
		}

		return string(u.Role), nil
	})

	if app.SubmissionBatcher != nil {
		defer app.SubmissionBatcher.Stop()
	}

	if app.SolveUseCase != nil {
		defer app.SolveUseCase.StopLocalScoreboardCache()
	}

	// Skip the legacy seed on fresh deploys - the setup wizard handles admin creation.
	if isSetupComplete(ctx, pool, l) {
		runSeed(cfg, app, l)
	}

	var g run.Group
	g.Add(func() error {
		l.Info("Starting HTTP server", map[string]any{"port": cfg.HTTP.Port})

		return app.Server.ListenAndServe()
	}, func(err error) {
		if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, context.Canceled) {
			l.WithError(err).Error("HTTP server error")
		}

		l.Info("Shutting down server")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()

		if err := app.Server.Shutdown(shutdownCtx); err != nil {
			l.WithError(err).Error("Server forced to shutdown")

			_ = app.Server.Close()
		}

		if app.RatelimitAuditWG != nil {
			waitDone := make(chan struct{})

			go func() {
				app.RatelimitAuditWG.Wait()
				close(waitDone)
			}()

			select {
			case <-waitDone:
			case <-time.After(shutdownGraceTimeout):
				l.Warn("ratelimit audit wait group timeout")
			}
		}

		if app.AvatarUC != nil {
			app.AvatarUC.Wait()
		}

		if app.Broadcaster != nil {
			app.Broadcaster.Wait()
		}
	})
	g.Add(func() error {
		<-ctx.Done()

		return nil
	}, func(_ error) {
		cancel()
	})

	if err := g.Run(); err != nil && !errors.Is(err, context.Canceled) {
		l.WithError(err).Error("run group error")
	}
}

// isSetupComplete queries the configs table to check whether the setup wizard
// has been completed. Returns true on any DB error (fail-open to avoid blocking
// an already-deployed platform due to a transient query failure).
func isSetupComplete(ctx context.Context, pool *pgxpool.Pool, l logkit.Logger) bool {
	var value string

	err := pool.QueryRow(ctx, "SELECT value FROM configs WHERE key = 'setup_complete' LIMIT 1").Scan(&value)
	if err != nil {
		l.WithError(err).Warn("app: could not read setup_complete from DB, assuming complete (fail-open)")

		return true
	}

	return value == "true"
}

// runSeed creates the default admin account when all three credentials
// (username, email, password) are set in cfg. A missing credential is treated
// as intentional and logged as info rather than an error.
func runSeed(cfg *config.Config, app *wire.App, l logkit.Logger) {
	adminUsername, adminEmail, adminPassword := cfg.Username, cfg.Email, cfg.Admin.Password
	if adminUsername == "" || adminEmail == "" || adminPassword == "" {
		l.Info("Admin credentials not provided, skipping default admin creation")

		return
	}

	err := seed.CreateDefaultAdmin(context.Background(), app.UserRepo, adminUsername, adminEmail, adminPassword, l, 0)
	if err != nil {
		l.WithError(err).Error("Failed to seed default admin")
	}
}

// provideStorage constructs the appropriate storage.Provider based on
// cfg.Provider: "s3" initialises an S3-compatible backend (minio-go) and
// ensures the bucket exists; any other value uses the local filesystem backend.
func provideStorage(ctx context.Context, cfg *config.Config, l logkit.Logger) (storage.Provider, error) {
	if cfg.Provider == "s3" {
		s3Provider, err := storage.NewS3Provider(
			cfg.S3Endpoint,
			cfg.S3PublicEndpoint,
			cfg.S3AccessKey,
			cfg.S3SecretKey,
			cfg.S3Bucket,
			cfg.S3Region,
			cfg.S3UseSSL,
		)
		if err != nil {
			return nil, fmt.Errorf("provideStorage - NewS3Provider: %w", err)
		}

		if err := s3Provider.EnsureBucket(ctx); err != nil {
			return nil, fmt.Errorf("provideStorage - EnsureBucket: %w", err)
		}

		l.Info("Using S3 storage provider", map[string]any{"endpoint": cfg.S3Endpoint, "bucket": cfg.S3Bucket})

		return s3Provider, nil
	}

	fsProvider, err := storage.NewFilesystemProvider(cfg.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("provideStorage - NewFilesystemProvider: %w", err)
	}

	l.Info("Using filesystem storage provider", map[string]any{"path": cfg.LocalPath})

	return fsProvider, nil
}

// reconcileSettings syncs brand-related env vars (APP_NAME, RESEND_FROM_NAME,
// RESEND_FROM_EMAIL) into the DB settings rows, but only when the DB still holds
// the generic migration defaults. This allows a fresh fork install to display the
// operator's custom CTF name without manual admin-UI intervention, while
// preserving any admin edits made after first boot.
func reconcileSettings(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, l logkit.Logger) {
	const (
		defaultAppName   = "CTF Platform"
		defaultFromEmail = "noreply@ctf-platform.local"
	)

	appName := cfg.Name
	fromName := cfg.FromName
	fromEmail := cfg.FromEmail
	resendEnabled := cfg.Enabled && cfg.APIKey != ""
	githubEnabled := cfg.GitHub.ClientID != "" && cfg.GitHub.ClientSecret != "" && cfg.GitHub.RedirectURL != ""
	googleEnabled := cfg.Google.ClientID != "" && cfg.Google.ClientSecret != "" && cfg.Google.RedirectURL != ""

	if appName == "" {
		appName = defaultAppName
	}

	if fromName == "" {
		fromName = defaultAppName
	}

	if fromEmail == "" {
		fromEmail = defaultFromEmail
	}

	_, err := pool.Exec(ctx,
		`UPDATE app_settings
		    SET app_name=$1,
		        resend_from_name=$2,
		        resend_from_email=$3,
		        resend_enabled=$4,
		        oauth_github_enabled=$5,
		        oauth_google_enabled=$6
		  WHERE id=1
		    AND app_name=$7`,
		appName, fromName, fromEmail, resendEnabled, githubEnabled, googleEnabled, defaultAppName,
	)
	if err != nil {
		l.WithError(err).Warn("reconcileSettings - app_settings update skipped")
	}

	_, err = pool.Exec(ctx,
		`UPDATE configs SET value=$1 WHERE key='ctf_name' AND value=$2`,
		appName, defaultAppName,
	)
	if err != nil {
		l.WithError(err).Warn("reconcileSettings - configs update skipped")
	}
}
