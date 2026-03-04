package main

import (
	"context"
	"os"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/postgres"
)

func main() {
	l := logger.New(&logger.Options{
		Level:  logger.InfoLevel,
		Output: logger.ConsoleOutput,
	})

	cfg, err := config.New()
	if err != nil {
		l.WithError(err).Error("failed to load config")
		os.Exit(1)
	}

	pool, err := postgres.New(&cfg.DB)
	if err != nil {
		l.WithError(err).Error("failed to connect to database")
		os.Exit(1)
	}
	defer pool.Close()

	ctx := context.Background()
	teamRepo := persistent.NewTeamRepo(pool)
	cleanupUC := usecase.NewCleanupUseCase(usecase.CleanupDeps{TeamRepo: teamRepo})

	duration := 30 * 24 * time.Hour
	l.Info("Starting cleanup of teams deleted more than 30 days ago", map[string]any{"duration": duration})

	if err := cleanupUC.CleanupDeletedTeams(ctx, duration); err != nil {
		l.WithError(err).Error("Cleanup failed")
		os.Exit(1)
	}

	l.Info("Cleanup completed successfully")
}
