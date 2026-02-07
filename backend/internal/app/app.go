package app

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/skr1ms/CTFBoard/config"
	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/skr1ms/CTFBoard/internal/wire"
	"github.com/skr1ms/CTFBoard/pkg/cache"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/skr1ms/CTFBoard/pkg/migrator"
	"github.com/skr1ms/CTFBoard/pkg/postgres"
	"github.com/skr1ms/CTFBoard/pkg/seed"
	pkgWS "github.com/skr1ms/CTFBoard/pkg/websocket"
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

	redisClient, err := cache.NewRedisClient(cfg.Host, cfg.Redis.Port, cfg.Redis.Password)
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

	storageProvider, err := provideStorage(cfg, l)
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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	jwtService := jwt.NewJWTService(cfg.AccessSecret, cfg.RefreshSecret, cfg.AccessTTL, cfg.RefreshTTL)
	wsHub := pkgWS.NewHub(redisClient, "scoreboard:updates")
	go wsHub.Run(ctx)
	go wsHub.SubscribeToRedis(ctx)

	resendMailer := mailer.New(mailer.Config{APIKey: cfg.APIKey, FromEmail: cfg.FromEmail, FromName: cfg.FromName})
	asyncMailer := mailer.NewAsyncMailer(resendMailer, 100, 2, l)
	asyncMailer.Start()
	defer asyncMailer.Stop()

	app, err := wire.InitializeApp(cfg, l, pool, redisClient, storageProvider, jwtService, wsHub, asyncMailer)
	if err != nil {
		l.WithError(err).Error("failed to initialize app")
		return
	}

	runSeed(cfg, app, l)
	runServerUntilShutdown(ctx, app.Server, cfg.HTTP.Port, l)
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

func runServerUntilShutdown(ctx context.Context, server *http.Server, port string, l logger.Logger) {
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := server.Shutdown(shutdownCtx); err != nil {
			l.WithError(err).Error("Server forced to shutdown")
			_ = server.Close()
		}
		cancel()
	}
}

func provideStorage(cfg *config.Config, l logger.Logger) (storage.Provider, error) {
	if cfg.Provider == "s3" {
		s3Provider, err := storage.NewS3Provider(
			cfg.S3Endpoint,
			cfg.S3PublicEndpoint,
			cfg.S3AccessKey,
			cfg.S3SecretKey,
			cfg.S3Bucket,
			cfg.S3UseSSL,
		)
		if err != nil {
			return nil, err
		}
		if err := s3Provider.EnsureBucket(context.Background()); err != nil {
			return nil, err
		}
		l.Info("Using S3 storage provider", map[string]any{"endpoint": cfg.S3Endpoint, "bucket": cfg.S3Bucket})
		return s3Provider, nil
	}
	fsProvider, err := storage.NewFilesystemProvider(cfg.LocalPath)
	if err != nil {
		return nil, err
	}
	l.Info("Using filesystem storage provider", map[string]any{"path": cfg.LocalPath})
	return fsProvider, nil
}
