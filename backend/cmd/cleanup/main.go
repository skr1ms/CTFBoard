package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-pgkit/postgres"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func main() {
	l, err := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	if err != nil {
		panic(err)
	}

	cfg, err := config.New()
	if err != nil {
		l.WithError(err).Fatal("failed to load config")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.New(ctx, &postgres.Config{
		URL:      cfg.URL,
		MaxConns: cfg.MaxConns,
		MinConns: cfg.MinConns,
	})
	if err != nil {
		l.WithError(err).Fatal("failed to connect to database")
	}
	defer pool.Close()

	storageProvider, err := provideStorage(ctx, cfg, l)
	if err != nil {
		l.WithError(err).Fatal("failed to initialize storage provider")
	}

	if closer, ok := storageProvider.(interface{ Close() error }); ok {
		defer func() {
			closeErr := closer.Close()
			if closeErr != nil {
				l.WithError(closeErr).Warn("failed to close storage provider")
			}
		}()
	}

	teamRepo := persistent.NewTeamRepo(pool)
	userRepo := persistent.NewUserRepo(pool)
	fileRepo := persistent.NewFileRepo(pool)
	trackingRepo := persistent.NewTrackingRepo(pool)
	cleanupUC := usecase.NewCleanupUseCase(usecase.CleanupDeps{
		UserRepo:     userRepo,
		TeamRepo:     teamRepo,
		FileRepo:     fileRepo,
		Storage:      storageProvider,
		TrackingRepo: trackingRepo,
	})

	duration := 30 * 24 * time.Hour
	l.Info("Starting cleanup of teams deleted more than 30 days ago", map[string]any{"duration": duration})

	if err := cleanupUC.CleanupDeletedTeams(ctx, duration); err != nil {
		l.WithError(err).Fatal("CleanupDeletedTeams failed")
	}

	l.Info("Starting cleanup of orphaned storage files in tasks/")

	deletedTasks, err := cleanupUC.CleanupOrphanedStorageFiles(ctx, "tasks/")
	if err != nil {
		l.WithError(err).Fatal("CleanupOrphanedStorageFiles(tasks/) failed")
	}

	l.Info("Orphaned task files cleanup completed", logkit.Fields{"deleted": deletedTasks})

	l.Info("Starting cleanup of orphaned avatar files")

	deletedAvatars, err := cleanupUC.CleanupOrphanedAvatars(ctx)
	if err != nil {
		l.WithError(err).Fatal("CleanupOrphanedAvatars failed")
	}

	l.Info("Orphaned avatar files cleanup completed", logkit.Fields{"deleted": deletedAvatars})

	l.Info("Starting cleanup of old tracking data (older than 90 days)")

	trackingDuration := 90 * 24 * time.Hour
	if err := cleanupUC.CleanupOldTracking(ctx, trackingDuration); err != nil {
		l.WithError(err).Fatal("CleanupOldTracking failed")
	}

	l.Info("Old tracking data cleanup completed")

	l.Info("Cleanup completed successfully")
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
			return nil, fmt.Errorf("provideStorage - NewS3Provider: %w", err)
		}

		if err := s3Provider.EnsureBucket(ctx); err != nil {
			return nil, fmt.Errorf("provideStorage - EnsureBucket: %w", err)
		}

		l.Info("Using S3 storage provider", logkit.Fields{"endpoint": cfg.S3Endpoint, "bucket": cfg.S3Bucket})

		return s3Provider, nil
	}

	fsProvider, err := storage.NewFilesystemProvider(cfg.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("provideStorage - NewFilesystemProvider: %w", err)
	}

	l.Info("Using filesystem storage provider", logkit.Fields{"path": cfg.LocalPath})

	return fsProvider, nil
}
