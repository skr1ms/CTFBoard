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
	restapiMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	v1 "github.com/skr1ms/CTFBoard/internal/controller/restapi/v1"
	wsController "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/crypto"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/skr1ms/CTFBoard/pkg/migrator"
	"github.com/skr1ms/CTFBoard/pkg/postgres"
	"github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/skr1ms/CTFBoard/pkg/seed"
	"github.com/skr1ms/CTFBoard/pkg/validator"
	pkgWS "github.com/skr1ms/CTFBoard/pkg/websocket"
	httpSwagger "github.com/swaggo/http-swagger"
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

	//nolint:staticcheck
	redisClient, err := redis.New(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password)
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

	userRepo := persistent.NewUserRepo(pool)
	challengeRepo := persistent.NewChallengeRepo(pool)
	solveRepo := persistent.NewSolveRepo(pool)
	teamRepo := persistent.NewTeamRepo(pool)
	competitionRepo := persistent.NewCompetitionRepo(pool)
	hintRepo := persistent.NewHintRepo(pool)
	hintUnlockRepo := persistent.NewHintUnlockRepo(pool)
	awardRepo := persistent.NewAwardRepo(pool)
	auditLogRepo := persistent.NewAuditLogRepo(pool)
	statsRepo := persistent.NewStatisticsRepository(pool)
	txRepo := persistent.NewTxRepo(pool)

	// Seed default admin if configured
	//nolint:staticcheck
	adminUsername := cfg.Admin.Username
	//nolint:staticcheck
	adminEmail := cfg.Admin.Email
	//nolint:staticcheck
	adminPassword := cfg.Admin.Password

	if adminUsername != "" && adminEmail != "" && adminPassword != "" {
		if err := seed.CreateDefaultAdmin(context.Background(), *userRepo, adminUsername, adminEmail, adminPassword, l); err != nil {
			l.WithError(err).Error("Failed to seed default admin")
		}
	} else {
		l.Info("Admin credentials not provided, skipping default admin creation")
	}

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

	var cryptoService *crypto.CryptoService
	if cfg.FlagEncryptionKey != "" {
		c, err := crypto.NewCryptoService(cfg.FlagEncryptionKey)
		if err != nil {
			l.WithError(err).Error("failed to initialize crypto service")
			return
		}
		cryptoService = c
	} else {
		l.Warn("FlagEncryptionKey not provided, regex challenges will fail")
	}

	userUC := usecase.NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)
	challengeUC := usecase.NewChallengeUseCase(challengeRepo, solveRepo, txRepo, competitionRepo, redisClient, wsHub, auditLogRepo, cryptoService)
	solveUC := usecase.NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, userRepo, txRepo, redisClient, wsHub)
	teamUC := usecase.NewTeamUseCase(teamRepo, userRepo, competitionRepo, txRepo)
	competitionUC := usecase.NewCompetitionUseCase(competitionRepo, auditLogRepo, redisClient)
	hintUC := usecase.NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)
	awardUC := usecase.NewAwardUseCase(awardRepo, redisClient)
	statsUC := usecase.NewStatisticsUseCase(statsRepo, redisClient)
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
			l.WithError(err).Error("failed to create S3 storage provider")
			return
		}
		if err := s3Provider.EnsureBucket(context.Background()); err != nil {
			l.WithError(err).Error("failed to ensure S3 bucket exists")
			return
		}
		storageProvider = s3Provider
		l.Info("Using S3 storage provider", map[string]any{"endpoint": cfg.S3Endpoint, "bucket": cfg.S3Bucket})
	} else {
		fsProvider, err := storage.NewFilesystemProvider(cfg.LocalPath)
		if err != nil {
			l.WithError(err).Error("failed to create filesystem storage provider")
			return
		}

		defer func() {
			if err := fsProvider.Close(); err != nil {
				l.WithError(err).Error("failed to close filesystem provider")
			}
		}()

		storageProvider = fsProvider
		l.Info("Using filesystem storage provider", map[string]any{"path": cfg.LocalPath})
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
		router.Use(restapiMiddleware.Logger(l))
	} else {
		router.Use(middleware.Logger)
	}
	router.Use(restapiMiddleware.Metrics)
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
			l.WithError(err).Error("failed to write health check response")
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
			awardUC,
			statsUC,
			jwtService,
			redisClient,
			validator,
			l,
			cfg.SubmitFlag,
			cfg.SubmitFlagDuration,
			cfg.VerifyEmails,
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
		l.Info("Starting HTTP server", map[string]any{"port": cfg.HTTP.Port})
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			l.WithError(err).Error("HTTP server error")
		}
	case <-shutdown:
		l.Info("Shutting down server")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		if err := server.Shutdown(ctx); err != nil {
			l.WithError(err).Error("Server forced to shutdown")
			if err := server.Close(); err != nil {
				l.WithError(err).Error("failed to close server")
			}
		}
		cancel()
	}
}
