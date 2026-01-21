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
	httpMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	v1 "github.com/skr1ms/CTFBoard/internal/controller/restapi/v1"
	wsController "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/skr1ms/CTFBoard/pkg/migrator"
	"github.com/skr1ms/CTFBoard/pkg/postgres"
	"github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/skr1ms/CTFBoard/pkg/validator"
	pkgWS "github.com/skr1ms/CTFBoard/pkg/websocket"
	httpSwagger "github.com/swaggo/http-swagger"
)

func Run(cfg *config.Config, l *logger.Logger) {
	l.Info("Application initialized", nil, map[string]interface{}{
		"mode":      cfg.ChiMode,
		"log_level": cfg.LogLevel,
		"version":   cfg.Version,
	})

	pool, err := postgres.New(&cfg.DB)
	if err != nil {
		l.Error("failed to connect to database", err)
		return
	}
	defer pool.Close()

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

	if err := migrator.Run(&cfg.DB); err != nil {
		l.Error("failed to run migrations", err)
		return
	}

	userRepo := persistent.NewUserRepo(pool)
	challengeRepo := persistent.NewChallengeRepo(pool)
	solveRepo := persistent.NewSolveRepo(pool)
	teamRepo := persistent.NewTeamRepo(pool)
	competitionRepo := persistent.NewCompetitionRepo(pool)
	hintRepo := persistent.NewHintRepo(pool)
	hintUnlockRepo := persistent.NewHintUnlockRepo(pool)
	awardRepo := persistent.NewAwardRepo(pool)
	txRepo := persistent.NewTxRepo(pool)

	validator := validator.New()

	jwtService := jwt.NewJWTService(
		cfg.AccessSecret,
		cfg.RefreshSecret,
		cfg.AccessTTL,
		cfg.RefreshTTL,
	)

	wsHub := pkgWS.NewHub(redisClient, "scoreboard:updates")
	go wsHub.Run()
	go wsHub.SubscribeToRedis(context.Background())

	userUC := usecase.NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)
	challengeUC := usecase.NewChallengeUseCase(challengeRepo, solveRepo, txRepo, redisClient, wsHub)
	solveUC := usecase.NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, redisClient, wsHub)
	teamUC := usecase.NewTeamUseCase(teamRepo, userRepo)
	competitionUC := usecase.NewCompetitionUseCase(competitionRepo, redisClient)
	hintUC := usecase.NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)
	var storageProvider storage.Provider
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
			l.Error("failed to create S3 storage provider", err)
			return
		}
		if err := s3Provider.EnsureBucket(context.Background()); err != nil {
			l.Error("failed to ensure S3 bucket exists", err)
			return
		}
		storageProvider = s3Provider
		l.Info("Using S3 storage provider", nil, map[string]interface{}{"endpoint": cfg.S3Endpoint, "bucket": cfg.S3Bucket})
	} else {
		fsProvider, err := storage.NewFilesystemProvider(cfg.LocalPath)
		if err != nil {
			l.Error("failed to create filesystem storage provider", err)
			return
		}

		defer func() {
			if err := fsProvider.Close(); err != nil {
				l.Error("failed to close filesystem provider", err)
			}
		}()

		storageProvider = fsProvider
		l.Info("Using filesystem storage provider", nil, map[string]interface{}{"path": cfg.LocalPath})
	}

	fileRepo := persistent.NewFileRepository(pool)
	fileUC := usecase.NewFileUseCase(fileRepo, storageProvider, cfg.PresignedExpiry)

	verificationTokenRepo := persistent.NewVerificationTokenRepo(pool)
	resendMailer := mailer.New(mailer.Config{
		APIKey:    cfg.APIKey,
		FromEmail: cfg.FromEmail,
		FromName:  cfg.FromName,
	})

	asyncMailer := mailer.NewAsyncMailer(resendMailer, 100, 2, l)
	asyncMailer.Start()
	defer asyncMailer.Stop()

	emailUC := usecase.NewEmailUseCase(
		userRepo,
		verificationTokenRepo,
		asyncMailer,
		cfg.VerifyTTL,
		cfg.ResetTTL,
		cfg.FrontendURL,
		cfg.Enabled,
	)

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

	router.Get("/swagger/*", httpSwagger.Handler())

	wsCtrl := wsController.NewController(wsHub, l, cfg.CORSOrigins)

	router.Route("/api/v1", func(r chi.Router) {
		wsCtrl.RegisterRoutes(r)

		v1.NewRouter(
			r,
			userUC,
			challengeUC,
			solveUC,
			teamUC,
			competitionUC,
			hintUC,
			emailUC,
			fileUC,
			jwtService,
			redisClient,
			validator,
			l,
			cfg.SubmitFlag,
			cfg.SubmitFlagDuration,
		)
	})

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
