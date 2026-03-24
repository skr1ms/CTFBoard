package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wahrwelt-kit/go-logkit"

	"github.com/wahrwelt-kit/go-pgkit/postgres"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
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
		URL:      cfg.DB.URL,
		MaxConns: cfg.DB.MaxConns,
		MinConns: cfg.DB.MinConns,
	})
	if err != nil {
		l.WithError(err).Fatal("failed to connect to database")
	}
	defer pool.Close()
	teamRepo := persistent.NewTeamRepo(pool)
	cleanupUC := usecase.NewCleanupUseCase(usecase.CleanupDeps{TeamRepo: teamRepo})

	duration := 30 * 24 * time.Hour
	l.Info("Starting cleanup of teams deleted more than 30 days ago", map[string]any{"duration": duration})

	if err := cleanupUC.CleanupDeletedTeams(ctx, duration); err != nil {
		l.WithError(err).Fatal("Cleanup failed")
	}

	l.Info("Cleanup completed successfully")
	os.Exit(0)
}
