package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/wire"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/migrator"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/postgres"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/seed"
	pkgWS "github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
)

const (
	asyncMailerQueueSize = 100
	asyncMailerWorkers   = 2
)

func Run(cfg *config.Config, l logger.Logger) {
	l.Info("Application initialized", map[string]any{
		"mode":      cfg.ChiMode,
		"log_level": cfg.LogLevel,
		"version":   cfg.Version,
	})

	pool, err := postgres.New(&cfg.DB)
	if err != nil {
		l.WithError(err).Error("failed to connect to database")
		return
	}
	defer pool.Close()

	redisClient, err := cache.NewRedisClient(&cfg.Redis)
	if err != nil {
		l.WithError(err).Error("failed to connect to redis")
		return
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			l.WithError(err).Error("failed to close redis connection")
		}
	}()

	if err := migrator.Run(&cfg.DB); err != nil {
		l.WithError(err).Error("failed to run migrations")
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	storageProvider, err := provideStorage(ctx, cfg, l)
	if err != nil {
		l.WithError(err).Error("failed to create storage provider")
		return
	}
	if closer, ok := storageProvider.(interface{ Close() error }); ok {
		defer func() {
			if err := closer.Close(); err != nil {
				l.WithError(err).Error("failed to close storage provider")
			}
		}()
	}

	jwtRevoker := jwt.NewRedisRevocationStore(redisClient)
	accessKeys := make([]jwt.KeyEntry, len(cfg.AccessKeys))
	for i, k := range cfg.AccessKeys {
		accessKeys[i] = jwt.KeyEntry{Kid: k.Kid, Secret: k.Secret}
	}
	refreshKeys := make([]jwt.KeyEntry, len(cfg.RefreshKeys))
	for i, k := range cfg.RefreshKeys {
		refreshKeys[i] = jwt.KeyEntry{Kid: k.Kid, Secret: k.Secret}
	}
	jwtService, err := jwt.NewJWTService(accessKeys, refreshKeys, cfg.AccessTTL, cfg.RefreshTTL, jwtRevoker, nil)
	if err != nil {
		l.WithError(err).Error("failed to create JWT service")
		return
	}
	wsHub := pkgWS.NewHub(redisClient, cache.PubSubScoreboard)
	wsHub.SetTimeoutLogger(func(op string) { l.Warn("websocket hub operation timed out", logger.Fields{"op": op}) })
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

	jwtService.SetUserRoleLookup(func(ctx context.Context, userID uuid.UUID) (string, string, string, error) {
		u, err := app.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return "", "", "", fmt.Errorf("app - SetUserRoleLookup - GetByID: %w", err)
		}
		if u.IsBanned {
			return "", "", "", httperr.ErrUserBanned
		}
		if u.WasInBannedTeam && u.Role != entity.RoleAdmin {
			return "", "", "", httperr.ErrUserBanned
		}
		return u.Email, u.Username, string(u.Role), nil
	})

	if app.SubmissionBatcher != nil {
		defer app.SubmissionBatcher.Stop()
	}

	runSeed(cfg, app, l)
	runServerUntilShutdown(ctx, app.Server, cfg.HTTP.Port, cfg.ShutdownTimeout, l)
}

func runSeed(cfg *config.Config, app *wire.App, l logger.Logger) {
	adminUsername, adminEmail, adminPassword := cfg.Username, cfg.Email, cfg.Admin.Password
	if adminUsername == "" || adminEmail == "" || adminPassword == "" {
		l.Info("Admin credentials not provided, skipping default admin creation")
		return
	}
	if err := seed.CreateDefaultAdmin(context.Background(), app.UserRepo, adminUsername, adminEmail, adminPassword, l); err != nil {
		l.WithError(err).Error("Failed to seed default admin")
	}
}

func runServerUntilShutdown(ctx context.Context, server *http.Server, port string, shutdownTimeout time.Duration, l logger.Logger) {
	serverErrors := make(chan error, 1)
	go func() {
		l.Info("Starting HTTP server", map[string]any{"port": port})
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.WithError(err).Error("HTTP server error")
		}
	case <-ctx.Done():
		l.Info("Shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			l.WithError(err).Error("Server forced to shutdown")
			_ = server.Close()
		}
	}
}

func provideStorage(ctx context.Context, cfg *config.Config, l logger.Logger) (storage.Provider, error) {
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
			return nil, fmt.Errorf("app - provideStorage - NewS3Provider: %w", err)
		}
		if err := s3Provider.EnsureBucket(ctx); err != nil {
			return nil, fmt.Errorf("app - provideStorage - EnsureBucket: %w", err)
		}
		l.Info("Using S3 storage provider", map[string]any{"endpoint": cfg.S3Endpoint, "bucket": cfg.S3Bucket})
		return s3Provider, nil
	}
	fsProvider, err := storage.NewFilesystemProvider(cfg.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("app - provideStorage - NewFilesystemProvider: %w", err)
	}
	l.Info("Using filesystem storage provider", map[string]any{"path": cfg.LocalPath})
	return fsProvider, nil
}
