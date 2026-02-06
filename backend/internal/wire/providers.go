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
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	wsController "github.com/skr1ms/CTFBoard/internal/controller/websocket/v1"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/storage"
	challenge "github.com/skr1ms/CTFBoard/internal/usecase/challenge"
	competition "github.com/skr1ms/CTFBoard/internal/usecase/competition"
	email "github.com/skr1ms/CTFBoard/internal/usecase/email"
	notification "github.com/skr1ms/CTFBoard/internal/usecase/notification"
	page "github.com/skr1ms/CTFBoard/internal/usecase/page"
	"github.com/skr1ms/CTFBoard/internal/usecase/settings"
	team "github.com/skr1ms/CTFBoard/internal/usecase/team"
	user "github.com/skr1ms/CTFBoard/internal/usecase/user"
	"github.com/skr1ms/CTFBoard/pkg/cache"
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

func ProvideSubmissionRepo(pool *pgxpool.Pool) *persistent.SubmissionRepo {
	return persistent.NewSubmissionRepo(pool)
}

func ProvideTagRepo(pool *pgxpool.Pool) *persistent.TagRepo {
	return persistent.NewTagRepo(pool)
}

func ProvideFieldRepo(pool *pgxpool.Pool) *persistent.FieldRepo {
	return persistent.NewFieldRepo(pool)
}

func ProvideFieldValueRepo(pool *pgxpool.Pool) *persistent.FieldValueRepo {
	return persistent.NewFieldValueRepo(pool)
}

func ProvideNotificationRepo(pool *pgxpool.Pool) *persistent.NotificationRepo {
	return persistent.NewNotificationRepo(pool)
}

func ProvidePageRepo(pool *pgxpool.Pool) *persistent.PageRepo {
	return persistent.NewPageRepo(pool)
}

func ProvideCommentRepo(pool *pgxpool.Pool) *persistent.CommentRepo {
	return persistent.NewCommentRepo(pool)
}

func ProvideAppSettingsRepo(pool *pgxpool.Pool) *persistent.AppSettingsRepo {
	return persistent.NewAppSettingsRepo(pool)
}

func ProvideConfigRepo(pool *pgxpool.Pool) *persistent.ConfigRepo {
	return persistent.NewConfigRepo(pool)
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
	fieldValidator *settings.FieldValidator,
	fieldValueRepo repo.FieldValueRepository,
) *user.UserUseCase {
	return user.NewUserUseCase(userRepo, teamRepo, solveRepo, txRepo, jwtService, fieldValidator, fieldValueRepo)
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
	tagRepo repo.TagRepository,
	solveRepo repo.SolveRepository,
	txRepo repo.TxRepository,
	compRepo repo.CompetitionRepository,
	teamRepo repo.TeamRepository,
	redis *redis.Client,
	broadcaster *pkgWS.Broadcaster,
	auditLogRepo repo.AuditLogRepository,
	cryptoService crypto.Service,
) *challenge.ChallengeUseCase {
	return challenge.NewChallengeUseCase(
		challengeRepo,
		challenge.WithTagRepo(tagRepo),
		challenge.WithSolveRepo(solveRepo),
		challenge.WithTxRepo(txRepo),
		challenge.WithCompetitionRepo(compRepo),
		challenge.WithTeamRepo(teamRepo),
		challenge.WithRedis(redis),
		challenge.WithBroadcaster(broadcaster),
		challenge.WithAuditLogRepo(auditLogRepo),
		challenge.WithCrypto(cryptoService),
	)
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
	teamRepo repo.TeamRepository,
	txRepo repo.TxRepository,
	c *cache.Cache,
	broadcaster *pkgWS.Broadcaster,
) *competition.SolveUseCase {
	return competition.NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, userRepo, teamRepo, txRepo, c, broadcaster)
}

func ProvideBroadcaster(hub *pkgWS.Hub) *pkgWS.Broadcaster {
	return pkgWS.NewBroadcaster(hub)
}

func ProvideCache(r *redis.Client) *cache.Cache {
	return cache.New(r)
}

func ProvideStatisticsUseCase(
	statsRepo repo.StatisticsRepository,
	c *cache.Cache,
) *competition.StatisticsUseCase {
	return competition.NewStatisticsUseCase(statsRepo, c)
}

func ProvideSubmissionUseCase(submissionRepo repo.SubmissionRepository) *competition.SubmissionUseCase {
	return competition.NewSubmissionUseCase(submissionRepo)
}

func ProvideTagUseCase(tagRepo repo.TagRepository) *challenge.TagUseCase {
	return challenge.NewTagUseCase(tagRepo)
}

func ProvideFieldUseCase(fieldRepo repo.FieldRepository) *settings.FieldUseCase {
	return settings.NewFieldUseCase(fieldRepo)
}

func ProvideFieldValidator(fieldRepo repo.FieldRepository) *settings.FieldValidator {
	return settings.NewFieldValidator(fieldRepo)
}

func ProvideNotificationUseCase(notifRepo repo.NotificationRepository) *notification.NotificationUseCase {
	return notification.NewNotificationUseCase(notifRepo)
}

func ProvidePageUseCase(pageRepo repo.PageRepository) *page.PageUseCase {
	return page.NewPageUseCase(pageRepo)
}

func ProvideCommentUseCase(commentRepo repo.CommentRepository, challengeRepo repo.ChallengeRepository) *challenge.CommentUseCase {
	return challenge.NewCommentUseCase(commentRepo, challengeRepo)
}

func ProvideBracketRepo(pool *pgxpool.Pool) *persistent.BracketRepo {
	return persistent.NewBracketRepo(pool)
}

func ProvideBracketUseCase(bracketRepo repo.BracketRepository) *competition.BracketUseCase {
	return competition.NewBracketUseCase(bracketRepo)
}

func ProvideRatingRepo(pool *pgxpool.Pool) *persistent.RatingRepo {
	return persistent.NewRatingRepo(pool)
}

func ProvideRatingUseCase(
	ratingRepo repo.RatingRepository,
	solveRepo repo.SolveRepository,
	teamRepo repo.TeamRepository,
) *competition.RatingUseCase {
	return competition.NewRatingUseCase(ratingRepo, solveRepo, teamRepo)
}

func ProvideAPITokenRepo(pool *pgxpool.Pool) *persistent.APITokenRepo {
	return persistent.NewAPITokenRepo(pool)
}

func ProvideAPITokenUseCase(apiTokenRepo repo.APITokenRepository) *user.APITokenUseCase {
	return user.NewAPITokenUseCase(apiTokenRepo)
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
) *settings.SettingsUseCase {
	return settings.NewSettingsUseCase(appSettingsRepo, auditLogRepo, redis)
}

func ProvideDynamicConfigUseCase(
	configRepo repo.ConfigRepository,
	auditLogRepo repo.AuditLogRepository,
) *competition.DynamicConfigUseCase {
	return competition.NewDynamicConfigUseCase(configRepo, auditLogRepo)
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

func ProvideServerDeps(
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
	submissionUC *competition.SubmissionUseCase,
	tagUC *challenge.TagUseCase,
	fieldUC *settings.FieldUseCase,
	pageUC *page.PageUseCase,
	bracketUC *competition.BracketUseCase,
	ratingUC *competition.RatingUseCase,
	notifUC *notification.NotificationUseCase,
	apiTokenUC *user.APITokenUseCase,
	backupUC *competition.BackupUseCase,
	settingsUC *settings.SettingsUseCase,
	dynamicConfigUC *competition.DynamicConfigUseCase,
	commentUC *challenge.CommentUseCase,
	jwtService *jwt.JWTService,
	redisClient *redis.Client,
	wsCtrl *wsController.Controller,
	v validator.Validator,
	l logger.Logger,
) *helper.ServerDeps {
	return &helper.ServerDeps{
		Challenge: helper.ChallengeDeps{
			ChallengeUC: challengeUC,
			HintUC:      hintUC,
			FileUC:      fileUC,
			TagUC:       tagUC,
			CommentUC:   commentUC,
		},
		Team: helper.TeamDeps{
			TeamUC:  teamUC,
			AwardUC: awardUC,
		},
		User: helper.UserDeps{
			UserUC:     userUC,
			EmailUC:    emailUC,
			APITokenUC: apiTokenUC,
		},
		Comp: helper.CompetitionDeps{
			CompetitionUC: competitionUC,
			SolveUC:       solveUC,
			StatsUC:       statsUC,
			SubmissionUC:  submissionUC,
			BracketUC:     bracketUC,
			RatingUC:      ratingUC,
		},
		Admin: helper.AdminDeps{
			BackupUC:        backupUC,
			SettingsUC:      settingsUC,
			DynamicConfigUC: dynamicConfigUC,
			FieldUC:         fieldUC,
			PageUC:          pageUC,
			NotifUC:         notifUC,
		},
		Infra: helper.InfraDeps{
			JWTService:   jwtService,
			RedisClient:  redisClient,
			WSController: wsCtrl,
			Validator:    v,
			Logger:       l,
		},
	}
}

func ProvideRouter(cfg *config.Config, l logger.Logger, deps *helper.ServerDeps) chi.Router {
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
		v1.NewRouter(r, deps, cfg.SubmitFlag, cfg.SubmitFlagDuration, cfg.VerifyEmails)
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
