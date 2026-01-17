package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/skr1ms/CTFBoard/config"
	httpMiddleware "github.com/skr1ms/CTFBoard/internal/controller/http/middleware"
	v1 "github.com/skr1ms/CTFBoard/internal/controller/http/v1"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/redis"

	// For PostgreSQL: uncomment postgres import and comment mariadb import
	// "github.com/skr1ms/CTFBoard/pkg/postgres"
	"github.com/skr1ms/CTFBoard/pkg/mariadb"
	"github.com/skr1ms/CTFBoard/pkg/migrator"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

func Run(cfg *config.Config, l *logger.Logger) {
	l.Info("Application initialized", nil, map[string]interface{}{
		"mode":      cfg.ChiMode,
		"log_level": cfg.LogLevel,
		"version":   cfg.Version,
	})

	// For PostgreSQL: comment mariadb.New and uncomment postgres.New
	// Also update config.go to use PostgreSQL connection string format
	db, err := mariadb.New(&cfg.DB)
	if err != nil {
		l.Error("failed to connect to database", err)
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			l.Error("failed to close database connection", err)
		}
	}()

	redisClient, err := redis.New(cfg.Host, cfg.Redis.Port, cfg.Password)
	if err != nil {
		l.Error("failed to connect to redis", err)
		return
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			l.Error("failed to close redis connection", err)
		}
	}()

	// For PostgreSQL: uncomment this block and comment mariadb.New above
	// Also add postgres service to docker-compose.yml (see comments there)
	// db, err := postgres.New(&cfg.DB)
	// if err != nil {
	// 	l.Error("failed to connect to database", err)
	// 	return
	// }
	// defer func() {
	// 	if err := db.Close(); err != nil {
	// 		l.Error("failed to close database connection", err)
	// 	}
	// }()

	if err := migrator.Run(&cfg.DB); err != nil {
		l.Error("failed to run migrations", err)
		return
	}

	userRepo := persistent.NewUserRepo(db)
	challengeRepo := persistent.NewChallengeRepo(db)
	solveRepo := persistent.NewSolveRepo(db)
	teamRepo := persistent.NewTeamRepo(db)
	competitionRepo := persistent.NewCompetitionRepo(db)

	validator := validator.New()

	jwtService := jwt.NewJWTService(
		cfg.AccessSecret,
		cfg.RefreshSecret,
		cfg.AccessTTL,
		cfg.RefreshTTL,
	)

	userUC := usecase.NewUserUseCase(userRepo, teamRepo, solveRepo, jwtService)
	challengeUC := usecase.NewChallengeUseCase(challengeRepo, solveRepo, redisClient)
	solveUC := usecase.NewSolveUseCase(solveRepo, competitionRepo, redisClient)
	teamUC := usecase.NewTeamUseCase(teamRepo, userRepo)
	competitionUC := usecase.NewCompetitionUseCase(competitionRepo, redisClient)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	if cfg.ChiMode == "production" {
		router.Use(httpMiddleware.Logger(l))
	} else {
		router.Use(middleware.Logger)
	}
	router.Use(httpMiddleware.Metrics)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(httprate.LimitByIP(100, 1*time.Minute))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			l.Error("failed to write health check response", err)
		}
	})

	router.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	v1.NewRouter(
		router,
		userUC,
		challengeUC,
		solveUC,
		teamUC,
		competitionUC,
		jwtService,
		validator,
		l,
		cfg.RateLimit.SubmitFlag,
		cfg.RateLimit.SubmitFlagDuration,
	)

	server := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 100 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		l.Info("Starting HTTP server", nil, map[string]interface{}{"port": cfg.HTTP.Port})
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			l.Error("HTTP server error", err)
		}
	case <-shutdown:
		l.Info("Shutting down server", nil)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		if err := server.Shutdown(ctx); err != nil {
			l.Error("Server forced to shutdown", err)
			if err := server.Close(); err != nil {
				l.Error("failed to close server", err)
			}
		}
		cancel()
	}
}
