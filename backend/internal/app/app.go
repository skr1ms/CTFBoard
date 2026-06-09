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
	"github.com/oklog/run"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-pgkit/migrator/goose"
	"github.com/wahrwelt-kit/go-pgkit/postgres"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/internal/seed"
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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

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

	settingsRepo := persistent.NewSettingsRepo(pool)
	paramRepo := persistent.NewCompetitionParamRepo(pool)

	reconcileSettings(ctx, cfg, settingsRepo, paramRepo, l)

	storageProvider, err := wire.ProvideStorage(ctx, cfg, l)
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
	asyncMailer := mailer.NewAsyncMailer(ctx, resendMailer, asyncMailerQueueSize, asyncMailerWorkers, l)

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
		// The ban middleware and usecase guards enforce blocked CTF operations.

		return string(u.Role), nil
	})

	if app.SolveUseCase != nil {
		defer app.SolveUseCase.StopLocalScoreboardCache()
	}

	// Skip the legacy seed on fresh deploys - the setup wizard handles admin creation.
	if isSetupComplete(ctx, paramRepo, l) {
		runSeed(ctx, cfg, app, l)
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

		if app.BackupUC != nil {
			waitDone := make(chan struct{})

			go func() {
				app.BackupUC.Wait()
				close(waitDone)
			}()

			select {
			case <-waitDone:
			case <-time.After(shutdownGraceTimeout):
				l.Warn("backup import wait group timeout")
			}
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

type setupCompleteReader interface {
	IsSetupComplete(ctx context.Context) (bool, error)
}

// isSetupComplete checks whether the setup wizard has been completed. Returns
// true on any DB error (fail-open to avoid blocking an already-deployed platform
// due to a transient query failure).
func isSetupComplete(ctx context.Context, reader setupCompleteReader, l logkit.Logger) bool {
	complete, err := reader.IsSetupComplete(ctx)
	if err != nil {
		l.WithError(err).Warn("app: could not read setup_complete from DB, assuming complete (fail-open)")

		return true
	}

	return complete
}

// runSeed creates the default admin account when all three credentials
// (username, email, password) are set in cfg. A missing credential is treated
// as intentional and logged as info rather than an error.
func runSeed(ctx context.Context, cfg *config.Config, app *wire.App, l logkit.Logger) {
	adminUsername, adminEmail, adminPassword := cfg.Username, cfg.Email, cfg.Admin.Password
	if adminUsername == "" || adminEmail == "" || adminPassword == "" {
		l.Info("Admin credentials not provided, skipping default admin creation")

		return
	}

	err := seed.CreateDefaultAdmin(ctx, app.UserRepo, adminUsername, adminEmail, adminPassword, l, 0)
	if err != nil {
		l.WithError(err).Error("Failed to seed default admin")
	}
}

type startupSettingsReconciler interface {
	ReconcileStartupDefaults(
		ctx context.Context,
		appName string,
		fromName string,
		fromEmail string,
		resendEnabled bool,
		githubEnabled bool,
		googleEnabled bool,
		defaultAppName string,
	) error
}

type ctfNameDefaultReconciler interface {
	ReconcileCTFNameDefault(ctx context.Context, appName, defaultAppName string) error
}

// reconcileSettings syncs brand-related env vars (APP_NAME, RESEND_FROM_NAME,
// RESEND_FROM_EMAIL) into the DB settings rows, but only when the DB still holds
// the generic migration defaults. This allows a fresh fork install to display the
// operator's custom CTF name without manual admin-UI intervention, while
// preserving any admin edits made after first boot.
func reconcileSettings(
	ctx context.Context,
	cfg *config.Config,
	settingsRepo startupSettingsReconciler,
	paramRepo ctfNameDefaultReconciler,
	l logkit.Logger,
) {
	const (
		defaultAppName   = "CTF Platform"
		defaultFromEmail = "noreply@ctf-platform.local"
	)

	appName := cfg.Name
	fromName := cfg.FromName
	fromEmail := cfg.FromEmail
	resendEnabled := cfg.Enabled
	githubEnabled := cfg.GitHub.IsConfigured()
	googleEnabled := cfg.Google.IsConfigured()

	if appName == "" {
		appName = defaultAppName
	}

	if fromName == "" {
		fromName = defaultAppName
	}

	if fromEmail == "" {
		fromEmail = defaultFromEmail
	}

	err := settingsRepo.ReconcileStartupDefaults(ctx,
		appName, fromName, fromEmail, resendEnabled, githubEnabled, googleEnabled, defaultAppName,
	)
	if err != nil {
		l.WithError(err).Warn("reconcileSettings - app_settings update skipped")
	}

	err = paramRepo.ReconcileCTFNameDefault(ctx, appName, defaultAppName)
	if err != nil {
		l.WithError(err).Warn("reconcileSettings - configs update skipped")
	}
}
