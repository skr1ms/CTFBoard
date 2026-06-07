package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-pgkit/postgres"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/wire"
)

const (
	deletedTeamsRetention = 30 * 24 * time.Hour
	oldTrackingRetention  = 90 * 24 * time.Hour
	tasksStoragePrefix    = "tasks/"
)

func main() {
	l, err := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger initialization failed: %v\n", err)
		os.Exit(1)
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

	storageProvider, err := wire.ProvideStorage(ctx, cfg, l)
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

	cleanupUC, err := wire.InitializeCleanup(pool, storageProvider)
	if err != nil {
		l.WithError(err).Fatal("failed to initialize cleanup graph")
	}

	l.Info("Starting cleanup of teams deleted more than 30 days ago", map[string]any{"duration": deletedTeamsRetention})

	if err := cleanupUC.CleanupDeletedTeams(ctx, deletedTeamsRetention); err != nil {
		l.WithError(err).Fatal("CleanupDeletedTeams failed")
	}

	l.Info("Starting cleanup of orphaned storage files in tasks/")

	deletedTasks, err := cleanupUC.CleanupOrphanedStorageFiles(ctx, tasksStoragePrefix)
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

	deletedTracking, err := cleanupUC.CleanupOldTracking(ctx, oldTrackingRetention)
	if err != nil {
		l.WithError(err).Fatal("CleanupOldTracking failed")
	}

	l.Info("Old tracking data cleanup completed", logkit.Fields{
		"tracking_deleted":        deletedTracking.TrackingDeleted,
		"challenge_opens_deleted": deletedTracking.ChallengeOpensDeleted,
	})

	l.Info("Cleanup completed successfully")
}
