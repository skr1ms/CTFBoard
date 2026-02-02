package wire

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/config"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	v1 "github.com/skr1ms/CTFBoard/internal/controller/restapi/v1"
	wsController "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/storage"
	challenge "github.com/skr1ms/CTFBoard/internal/usecase/challenge"
	competition "github.com/skr1ms/CTFBoard/internal/usecase/competition"
	email "github.com/skr1ms/CTFBoard/internal/usecase/email"
	team "github.com/skr1ms/CTFBoard/internal/usecase/team"
	user "github.com/skr1ms/CTFBoard/internal/usecase/user"
	"github.com/skr1ms/CTFBoard/pkg/crypto"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	"github.com/skr1ms/CTFBoard/pkg/validator"
	pkgWS "github.com/skr1ms/CTFBoard/pkg/websocket"
	httpSwagger "github.com/swaggo/http-swagger"
)

func ProvideUserRepo(pool *pgxpool.Pool) *persistent.UserRepo {
	return persistent.NewUserRepo(pool)
}

func ProvideChallengeRepo(pool *pgxpool.Pool) *persistent.ChallengeRepo {
	return persistent.NewChallengeRepo(pool)
}

func ProvideSolveRepo(pool *pgxpool.Pool) *persistent.SolveRepo {
	return persistent.NewSolveRepo(pool)
}

func ProvideTeamRepo(pool *pgxpool.Pool) *persistent.TeamRepo {
	return persistent.NewTeamRepo(pool)
}

func ProvideCompetitionRepo(pool *pgxpool.Pool) *persistent.CompetitionRepo {
	return persistent.NewCompetitionRepo(pool)
}

func ProvideHintRepo(pool *pgxpool.Pool) *persistent.HintRepo {
	return persistent.NewHintRepo(pool)
}

func ProvideHintUnlockRepo(pool *pgxpool.Pool) *persistent.HintUnlockRepo {
	return persistent.NewHintUnlockRepo(pool)
}

func ProvideAwardRepo(pool *pgxpool.Pool) *persistent.AwardRepo {
	return persistent.NewAwardRepo(pool)
}

func ProvideAuditLogRepo(pool *pgxpool.Pool) *persistent.AuditLogRepo {
	return persistent.NewAuditLogRepo(pool)
}

func ProvideStatisticsRepo(pool *pgxpool.Pool) *persistent.StatisticsRepository {
	return persistent.NewStatisticsRepository(pool)
}

func ProvideFileRepo(pool *pgxpool.Pool) *persistent.FileRepository {
	return persistent.NewFileRepository(pool)
}

func ProvideTxRepo(pool *pgxpool.Pool) *persistent.TxRepo {
	return persistent.NewTxRepo(pool)
}

func ProvideBackupRepo(pool *pgxpool.Pool) *persistent.BackupRepo {
	return persistent.NewBackupRepo(pool)
}

func ProvideAppSettingsRepo(pool *pgxpool.Pool) *persistent.AppSettingsRepo {
	return persistent.NewAppSettingsRepo(pool)
}

func ProvideVerificationTokenRepo(pool *pgxpool.Pool) *persistent.VerificationTokenRepo {
	return persistent.NewVerificationTokenRepo(pool)
}

func ProvideValidator() validator.Validator {
	return validator.New()
}

func ProvideCrypto(cfg *config.Config) (crypto.Service, error) {
	if cfg.FlagEncryptionKey == "" {
		return nil, nil
	}
	return crypto.NewCryptoService(cfg.FlagEncryptionKey)
}

func ProvideUserUseCase(
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	solveRepo repo.SolveRepository,
	txRepo repo.TxRepository,
	jwtService *jwt.JWTService,
) *user.UserUseCase {
	return user.NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService)
}

func ProvideTeamUseCase(
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	compRepo repo.CompetitionRepository,
	txRepo repo.TxRepository,
	redis *redis.Client,
) *team.TeamUseCase {
	return team.NewTeamUseCase(teamRepo, userRepo, compRepo, txRepo, redis)
}

func ProvideAwardUseCase(
	awardRepo repo.AwardRepository,
	txRepo repo.TxRepository,
	redis *redis.Client,
) *team.AwardUseCase {
	return team.NewAwardUseCase(awardRepo, txRepo, redis)
}

func ProvideChallengeUseCase(
	challengeRepo repo.ChallengeRepository,
	solveRepo repo.SolveRepository,
	txRepo repo.TxRepository,
	compRepo repo.CompetitionRepository,
	teamRepo repo.TeamRepository,
	redis *redis.Client,
	hub *pkgWS.Hub,
	auditLogRepo repo.AuditLogRepository,
	cryptoService crypto.Service,
) *challenge.ChallengeUseCase {
	return challenge.NewChallengeUseCase(challengeRepo, solveRepo, txRepo, compRepo, teamRepo, redis, hub, auditLogRepo, cryptoService)
}

func ProvideHintUseCase(
	hintRepo repo.HintRepository,
	hintUnlockRepo repo.HintUnlockRepository,
	awardRepo repo.AwardRepository,
	txRepo repo.TxRepository,
	solveRepo repo.SolveRepository,
	redis *redis.Client,
) *challenge.HintUseCase {
	return challenge.NewHintUseCase(hintRepo, hintUnlockRepo, awardRepo, txRepo, solveRepo, redis)
}

func ProvideCompetitionUseCase(
	competitionRepo repo.CompetitionRepository,
	auditLogRepo repo.AuditLogRepository,
	redis *redis.Client,
) *competition.CompetitionUseCase {
	return competition.NewCompetitionUseCase(competitionRepo, auditLogRepo, redis)
}

func ProvideSolveUseCase(
	solveRepo repo.SolveRepository,
	challengeRepo repo.ChallengeRepository,
	competitionRepo repo.CompetitionRepository,
	userRepo repo.UserRepository,
	txRepo repo.TxRepository,
	redis *redis.Client,
	hub *pkgWS.Hub,
) *competition.SolveUseCase {
	return competition.NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, userRepo, txRepo, redis, hub)
}

func ProvideStatisticsUseCase(
	statsRepo repo.StatisticsRepository,
	redis *redis.Client,
) *competition.StatisticsUseCase {
	return competition.NewStatisticsUseCase(statsRepo, redis)
}

func ProvideFileUseCase(
	fileRepo repo.FileRepository,
	storageProvider storage.Provider,
	cfg *config.Config,
) *challenge.FileUseCase {
	return challenge.NewFileUseCase(fileRepo, storageProvider, cfg.PresignedExpiry)
}

func ProvideBackupUseCase(
	competitionRepo repo.CompetitionRepository,
	challengeRepo repo.ChallengeRepository,
	hintRepo repo.HintRepository,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	awardRepo repo.AwardRepository,
	solveRepo repo.SolveRepository,
	fileRepo repo.FileRepository,
	backupRepo repo.BackupRepository,
	storageProvider storage.Provider,
	txRepo repo.TxRepository,
	l logger.Logger,
) *competition.BackupUseCase {
	return competition.NewBackupUseCase(competitionRepo, challengeRepo, hintRepo, teamRepo, userRepo, awardRepo, solveRepo, fileRepo, backupRepo, storageProvider, txRepo, l)
}

func ProvideSettingsUseCase(
	appSettingsRepo repo.AppSettingsRepository,
	auditLogRepo repo.AuditLogRepository,
	redis *redis.Client,
) *competition.SettingsUseCase {
	return competition.NewSettingsUseCase(appSettingsRepo, auditLogRepo, redis)
}

func ProvideEmailUseCase(
	userRepo repo.UserRepository,
	tokenRepo repo.VerificationTokenRepository,
	mailer mailer.Mailer,
	cfg *config.Config,
) *email.EmailUseCase {
	return email.NewEmailUseCase(userRepo, tokenRepo, mailer, cfg.VerifyTTL, cfg.ResetTTL, cfg.FrontendURL, cfg.Enabled)
}

func ProvideWsController(wsHub *pkgWS.Hub, l logger.Logger, cfg *config.Config) *wsController.Controller {
	return wsController.NewController(wsHub, l, cfg.CORSOrigins)
}

func ProvideRouter(
	cfg *config.Config,
	l logger.Logger,
	userUC *user.UserUseCase,
	challengeUC *challenge.ChallengeUseCase,
	solveUC *competition.SolveUseCase,
	teamUC *team.TeamUseCase,
	competitionUC *competition.CompetitionUseCase,
	hintUC *challenge.HintUseCase,
	emailUC *email.EmailUseCase,
	fileUC *challenge.FileUseCase,
	awardUC *team.AwardUseCase,
	statsUC *competition.StatisticsUseCase,
	backupUC *competition.BackupUseCase,
	settingsUC *competition.SettingsUseCase,
	jwtService *jwt.JWTService,
	redisClient *redis.Client,
	wsCtrl *wsController.Controller,
	v validator.Validator,
) chi.Router {
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
		_, _ = w.Write([]byte("OK")) //nolint:errcheck
	})
	router.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	))
	router.Get("/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		swagger, err := openapi.GetSwagger()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonBytes, err := json.Marshal(swagger)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonBytes) //nolint:errcheck
	})
	router.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/openapi.json")))
	router.Route("/api/v1", func(r chi.Router) {
		v1.NewRouter(
			r,
			userUC, challengeUC, solveUC, teamUC, competitionUC, hintUC, emailUC, fileUC, awardUC, statsUC,
			backupUC, settingsUC, jwtService, redisClient, wsCtrl, v, l,
			cfg.SubmitFlag, cfg.SubmitFlagDuration, cfg.VerifyEmails,
		)
	})
	return router
}

func ProvideServer(router chi.Router, cfg *config.Config) *http.Server {
	return &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 100 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func ProvideApp(server *http.Server, userRepo repo.UserRepository) *App {
	return &App{Server: server, UserRepo: userRepo}
}
