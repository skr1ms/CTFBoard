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
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/wire"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/seed"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
)

const (
	asyncMailerQueueSize = 100
	asyncMailerWorkers   = 2
)

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
		wskit.WithOnConnect(func(client *wskit.Client) {
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
			return "", fmt.Errorf("app - SetUserRoleLookup - GetByID: %w", err)
		}

		if u.IsBanned {
			return "", httperr.ErrUserBanned
		}

		if u.WasInBannedTeam && u.Role != domain.RoleAdmin {
			return "", httperr.ErrUserBanned
		}

		return string(u.Role), nil
	})

	if app.SubmissionBatcher != nil {
		defer app.SubmissionBatcher.Stop()
	}

	if app.SolveUseCase != nil {
		defer app.SolveUseCase.StopLocalScoreboardCache()
	}

	runSeed(cfg, app, l)

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
			case <-time.After(5 * time.Second):
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

func runSeed(cfg *config.Config, app *wire.App, l logkit.Logger) {
	adminUsername, adminEmail, adminPassword := cfg.Username, cfg.Email, cfg.Admin.Password
	if adminUsername == "" || adminEmail == "" || adminPassword == "" {
		l.Info("Admin credentials not provided, skipping default admin creation")

		return
	}

	err := seed.CreateDefaultAdmin(context.Background(), app.UserRepo, adminUsername, adminEmail, adminPassword, l)
	if err != nil {
		l.WithError(err).Error("Failed to seed default admin")
	}
}

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
