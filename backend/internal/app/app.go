package app

import (
	"context"
	"errors"
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
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	v1 "github.com/skr1ms/CTFBoard/internal/controller/restapi/v1"
	wsController "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/storage"
	challenge "github.com/skr1ms/CTFBoard/internal/usecase/challenge"
	competition "github.com/skr1ms/CTFBoard/internal/usecase/competition"
	email "github.com/skr1ms/CTFBoard/internal/usecase/email"
	"github.com/skr1ms/CTFBoard/internal/usecase/settings"
	team "github.com/skr1ms/CTFBoard/internal/usecase/team"
	user "github.com/skr1ms/CTFBoard/internal/usecase/user"
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

//nolint:gocognit,gocyclo,funlen
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
	fileRepo := persistent.NewFileRepository(pool)
	txRepo := persistent.NewTxRepo(pool)
	backupRepo := persistent.NewBackupRepo(pool)
	appSettingsRepo := persistent.NewAppSettingsRepo(pool)
	verificationTokenRepo := persistent.NewVerificationTokenRepo(pool)

	adminUsername, adminEmail, adminPassword := cfg.Admin.Username, cfg.Email, cfg.Admin.Password //nolint:staticcheck
	if adminUsername != "" && adminEmail != "" && adminPassword != "" {
		if err := seed.CreateDefaultAdmin(context.Background(), *userRepo, adminUsername, adminEmail, adminPassword, l); err != nil {
			l.WithError(err).Error("Failed to seed default admin")
		}
	} else {
		l.Info("Admin credentials not provided, skipping default admin creation")
	}

	validator := validator.New()
	jwtService := jwt.NewJWTService(cfg.AccessSecret, cfg.RefreshSecret, cfg.AccessTTL, cfg.RefreshTTL)
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

	resendMailer := mailer.New(mailer.Config{APIKey: cfg.APIKey, FromEmail: cfg.FromEmail, FromName: cfg.FromName})
	asyncMailer := mailer.NewAsyncMailer(resendMailer, 100, 2, l)
	asyncMailer.Start()
	defer asyncMailer.Stop()

	userUC := user.NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)
	teamUC := team.NewTeamUseCase(teamRepo, userRepo, competitionRepo, txRepo)
	awardUC := team.NewAwardUseCase(awardRepo, txRepo, redisClient)
	challengeUC := challenge.NewChallengeUseCase(challengeRepo, solveRepo, txRepo, competitionRepo, redisClient, wsHub, auditLogRepo, cryptoService)
	hintUC := challenge.NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redisClient)
	competitionUC := competition.NewCompetitionUseCase(competitionRepo, auditLogRepo, redisClient)
	solveUC := competition.NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, userRepo, txRepo, redisClient, wsHub)
	statsUC := competition.NewStatisticsUseCase(statsRepo, redisClient)
	fileUC := challenge.NewFileUseCase(fileRepo, storageProvider, cfg.PresignedExpiry)
	backupUC := competition.NewBackupUseCase(competitionRepo, challengeRepo, hintRepo, teamRepo, userRepo, awardRepo, solveRepo, fileRepo, backupRepo, storageProvider, txRepo, l)
	settingsUC := settings.NewSettingsUseCase(appSettingsRepo, auditLogRepo, redisClient)
	emailUC := email.NewEmailUseCase(userRepo, verificationTokenRepo, asyncMailer, cfg.VerifyTTL, cfg.ResetTTL, cfg.FrontendURL, cfg.Enabled)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	if cfg.ChiMode == "production" {
		router.Use(restapimiddleware.Logger(l))
	} else {
		router.Use(middleware.Logger)
	}
	router.Use(restapimiddleware.Metrics)
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

	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK")) //nolint:errcheck // best-effort health
	})

	router.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	router.Get("/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		swagger, err := openapi.GetSwagger()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonBytes, err := swagger.MarshalJSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonBytes) //nolint:errcheck // best-effort openapi json
	})

	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/openapi.json"),
	))

	wsCtrl := wsController.NewController(wsHub, l, cfg.CORSOrigins)

	router.Route("/api/v1", func(r chi.Router) {
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
			backupUC,
			settingsUC,
			jwtService,
			redisClient,
			wsCtrl,
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
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
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
