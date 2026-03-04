package wire

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	v1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	wsController "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/webapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	backup "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	challenge "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	competition "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	email "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email"
	notification "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification"
	page "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	team "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
	user "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/loginlockout"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
	pkgWS "github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	rlKeyForgot  = "forgot"
	rlKeyResend  = "resend"
	rlKeyGeneral = "general:ip"

	requestTimeout    = 60 * time.Second
	rateLimitCacheTTL = 30 * time.Second

	httpReadTimeout  = 15 * time.Second
	httpWriteTimeout = 100 * time.Second
	httpIdleTimeout  = 60 * time.Second

	loginLockoutMaxAttempts     = 5
	loginLockoutTTL             = 15 * time.Minute
	forgotPasswordRateLimit     = 10
	resendVerificationRateLimit = 10
	perKeyRateLimitWindow       = 24 * time.Hour
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

func ProvideCompetitionGuard(compUC *competition.CompetitionUseCase) *competition.Guard {
	return competition.NewGuard(compUC)
}

func ProvideHintRepo(pool *pgxpool.Pool) *persistent.HintRepo {
	return persistent.NewHintRepo(pool)
}

func ProvideTrackingRepo(pool *pgxpool.Pool) *persistent.TrackingRepo {
	return persistent.NewTrackingRepo(pool)
}

func ProvideAwardRepo(pool *pgxpool.Pool) *persistent.AwardRepo {
	return persistent.NewAwardRepo(pool)
}

func ProvideAuditLogRepo(pool *pgxpool.Pool) *persistent.AuditLogRepo {
	return persistent.NewAuditLogRepo(pool)
}

func ProvideStatisticsRepo(pool *pgxpool.Pool) *persistent.StatisticsRepo {
	return persistent.NewStatisticsRepo(pool)
}

func ProvideFileRepo(pool *pgxpool.Pool) *persistent.FileRepo {
	return persistent.NewFileRepo(pool)
}

func ProvideTransactionManager(pool *pgxpool.Pool) *persistent.TransactionManager {
	return persistent.NewTransactionManager(pool)
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

func ProvideFieldValueRepo(pool *pgxpool.Pool, tm *persistent.TransactionManager) *persistent.FieldValueRepo {
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

func ProvideSettingsRepo(pool *pgxpool.Pool) *persistent.SettingsRepo {
	return persistent.NewSettingsRepo(pool)
}

func ProvideCompetitionParamRepo(pool *pgxpool.Pool) *persistent.CompetitionParamRepo {
	return persistent.NewCompetitionParamRepo(pool)
}

func ProvideVerificationTokenRepo(pool *pgxpool.Pool) *persistent.VerificationTokenRepo {
	return persistent.NewVerificationTokenRepo(pool)
}

func ProvideOAuthRepo(pool *pgxpool.Pool) *persistent.OAuthRepo {
	return persistent.NewOAuthRepo(pool)
}

func ProvideOAuthProviders() map[string]webapi.OAuthProviderAPI {
	oauthClient := &http.Client{Timeout: 30 * time.Second}
	return map[string]webapi.OAuthProviderAPI{
		"github": webapi.NewGitHubAPI(oauthClient),
		"google": webapi.NewGoogleAPI(oauthClient),
	}
}

func ProvideOAuthUseCase(
	userRepo repo.UserRepository,
	oauthRepo repo.OAuthAccountRepository,
	TM repo.TransactionManager,
	settingsRepo repo.SettingsRepository,
	jwtService *jwt.JWTService,
	providers map[string]webapi.OAuthProviderAPI,
	cfg *config.Config,
	compRepo repo.CompetitionRepository,
	soloTeamCreator user.SoloTeamCreator,
	l logger.Logger,
) *user.OAuthUseCase {
	return user.NewOAuthUseCase(user.OAuthDeps{
		UserRepo:        userRepo,
		OAuthRepo:       oauthRepo,
		TM:              TM,
		SettingsRepo:    settingsRepo,
		JWTService:      jwtService,
		Providers:       providers,
		Cfg:             cfg.OAuth,
		CompRepo:        compRepo,
		SoloTeamCreator: soloTeamCreator,
		Logger:          l,
	})
}

func ProvideValidator() (validator.Validator, error) {
	return validator.New()
}

func ProvideFailedLoginTracker(redisClient *redis.Client) *loginlockout.Tracker {
	return loginlockout.NewTracker(redisClient, loginLockoutMaxAttempts, loginLockoutTTL)
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
	submissionRepo repo.SubmissionRepository,
	awardRepo repo.AwardRepository,
	TM repo.TransactionManager,
	jwtService *jwt.JWTService,
	fieldValidator *settings.FieldValidator,
	fieldValueRepo repo.FieldValueRepository,
	SettingsRepo repo.SettingsRepository,
	emailUC *email.EmailUseCase,
	failedLoginTracker *loginlockout.Tracker,
	compRepo repo.CompetitionRepository,
	soloTeamCreator user.SoloTeamCreator,
	userCacheSvc *cache.UserCacheService,
	scoreboardCache *cache.ScoreboardCacheService,
	l logger.Logger,
) *user.UserUseCase {
	return user.NewUserUseCase(user.UserDeps{
		UserRepo: userRepo, TeamRepo: teamRepo, SolveRepo: solveRepo,
		SubmissionRepo: submissionRepo, AwardRepo: awardRepo,
		TM: TM, JWTService: jwtService,
		FieldValidator: fieldValidator, FieldValueRepo: fieldValueRepo,
		SettingsRepo: SettingsRepo, EmailSender: emailUC, FailedLogin: failedLoginTracker,
		CompRepo: compRepo, SoloTeamCreator: soloTeamCreator,
		UserCache:       userCacheSvc,
		ScoreboardCache: scoreboardCache,
		Logger:          l,
	})
}

type teamBracketIDGetter struct {
	r repo.TeamRepository
}

func (g *teamBracketIDGetter) GetTeamBracketID(ctx context.Context, teamID uuid.UUID) (*uuid.UUID, error) {
	team, err := g.r.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("wire - GetTeamBracketID - TeamRepo.GetByID: %w", err)
	}
	if team == nil {
		return nil, fmt.Errorf("wire - GetTeamBracketID: team %s not found", teamID)
	}
	return team.BracketID, nil
}

func ProvideScoreboardCacheService(c *cache.Cache, teamRepo repo.TeamRepository) *cache.ScoreboardCacheService {
	return cache.NewScoreboardCacheService(c, &teamBracketIDGetter{r: teamRepo})
}

func ProvideUserCacheService(c *cache.Cache) *cache.UserCacheService {
	return cache.NewUserCacheService(c)
}

func ProvideTeamUseCase(
	cfg *config.Config,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	solveRepo repo.SolveRepository,
	submissionRepo repo.SubmissionRepository,
	awardRepo repo.AwardRepository,
	compRepo repo.CompetitionRepository,
	settingsUC *settings.SettingsUseCase,
	challengeRepo repo.ChallengeRepository,
	TM repo.TransactionManager,
	guard usecase.CompetitionGuard,
	scoreboardCache *cache.ScoreboardCacheService,
	userCacheSvc *cache.UserCacheService,
	sharedCache *cache.Cache,
	hintRepo repo.HintRepository,
) *team.TeamUseCase {
	return team.NewTeamUseCase(team.TeamDeps{
		TeamRepo:           teamRepo,
		UserRepo:           userRepo,
		SolveRepo:          solveRepo,
		SubmissionRepo:     submissionRepo,
		AwardRepo:          awardRepo,
		CompRepo:           compRepo,
		SettingsGetter:     settingsUC,
		ChallengeRepo:      challengeRepo,
		TM:                 TM,
		Guard:              guard,
		ScoreboardCache:    scoreboardCache,
		UserCache:          userCacheSvc,
		TeamCache:          sharedCache,
		HintRepo:           hintRepo,
		DefaultMaxTeamSize: cfg.MaxTeamSize,
	})
}

func ProvideAwardUseCase(
	awardRepo repo.AwardRepository,
	teamRepo repo.TeamRepository,
	TM repo.TransactionManager,
	scoreboardCache *cache.ScoreboardCacheService,
) *team.AwardUseCase {
	return team.NewAwardUseCase(team.AwardDeps{
		AwardRepo:       awardRepo,
		TeamRepo:        teamRepo,
		TM:              TM,
		ScoreboardCache: scoreboardCache,
	})
}

func ProvideChallengeUseCase(
	challengeRepo repo.ChallengeRepository,
	tagRepo repo.TagRepository,
	solveRepo repo.SolveRepository,
	TM repo.TransactionManager,
	compRepo repo.CompetitionRepository,
	compUC *competition.CompetitionUseCase,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	scoreboardCache *cache.ScoreboardCacheService,
	c *cache.Cache,
	broadcaster *pkgWS.Broadcaster,
	auditLogRepo repo.AuditLogRepository,
	cryptoService crypto.Service,
	fileRepo repo.FileRepository,
	storageProvider storage.Provider,
) *challenge.ChallengeUseCase {
	return challenge.NewChallengeUseCase(challenge.ChallengeDeps{
		ChallengeRepo:   challengeRepo,
		TagRepo:         tagRepo,
		SolveRepo:       solveRepo,
		TM:              TM,
		CompRepo:        compRepo,
		CompUC:          compUC,
		TeamRepo:        teamRepo,
		UserRepo:        userRepo,
		ScoreboardCache: scoreboardCache,
		ListCache:       c,
		Broadcaster:     broadcaster,
		AuditLogRepo:    auditLogRepo,
		Crypto:          cryptoService,
		FileRepo:        fileRepo,
		Storage:         storageProvider,
	})
}

func ProvideHintUseCase(
	hintRepo repo.HintRepository,
	awardRepo repo.AwardRepository,
	TM repo.TransactionManager,
	solveRepo repo.SolveRepository,
	compRepo repo.CompetitionRepository,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	challengeRepo repo.ChallengeRepository,
	scoreboardCache *cache.ScoreboardCacheService,
) *challenge.HintUseCase {
	return challenge.NewHintUseCase(challenge.HintDeps{
		HintRepo:        hintRepo,
		AwardRepo:       awardRepo,
		TM:              TM,
		SolveRepo:       solveRepo,
		CompRepo:        compRepo,
		TeamRepo:        teamRepo,
		UserRepo:        userRepo,
		ChallengeRepo:   challengeRepo,
		ScoreboardCache: scoreboardCache,
	})
}

func ProvideCompetitionUseCase(
	competitionRepo repo.CompetitionRepository,
	auditLogRepo repo.AuditLogRepository,
	TM repo.TransactionManager,
	kv cache.KeyValueStore,
	l logger.Logger,
) *competition.CompetitionUseCase {
	return competition.NewCompetitionUseCase(competition.CompetitionDeps{
		CompetitionRepo: competitionRepo,
		AuditLogRepo:    auditLogRepo,
		TM:              TM,
		Redis:           kv,
		Logger:          l,
	})
}

func ProvideSolveUseCase(
	solveRepo repo.SolveRepository,
	challengeRepo repo.ChallengeRepository,
	competitionRepo repo.CompetitionRepository,
	competitionUC usecase.CompetitionUseCase,
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	TM repo.TransactionManager,
	c *cache.Cache,
	scoreboardCache *cache.ScoreboardCacheService,
	challengeListCache cache.ChallengeListCacheInvalidator,
	broadcaster *pkgWS.Broadcaster,
) *competition.SolveUseCase {
	return competition.NewSolveUseCase(competition.SolveDeps{
		SolveRepo:          solveRepo,
		ChallengeRepo:      challengeRepo,
		CompetitionRepo:    competitionRepo,
		CompetitionUC:      competitionUC,
		UserRepo:           userRepo,
		TeamRepo:           teamRepo,
		TM:                 TM,
		Cache:              c,
		ScoreboardCache:    scoreboardCache,
		ChallengeListCache: challengeListCache,
		Broadcaster:        broadcaster,
	})
}

func ProvideBroadcaster(hub *pkgWS.Hub) *pkgWS.Broadcaster {
	return pkgWS.NewBroadcaster(hub)
}

func ProvideCache(r *redis.Client) *cache.Cache {
	return cache.New(r)
}

func ProvideKeyValueStore(r *redis.Client) cache.KeyValueStore {
	return &cache.RedisKeyValueStore{Client: r}
}

func ProvideStatisticsUseCase(
	statsRepo repo.StatisticsRepository,
	c *cache.Cache,
	compUC *competition.CompetitionUseCase,
) *competition.StatisticsUseCase {
	return competition.NewStatisticsUseCase(competition.StatisticsDeps{
		StatsRepo:  statsRepo,
		Cache:      c,
		CompGetter: compUC,
	})
}

func ProvideSubmissionUseCase(submissionRepo repo.SubmissionRepository, TM repo.TransactionManager, challengeUC *challenge.ChallengeUseCase, l logger.Logger) *competition.SubmissionUseCase {
	return competition.NewSubmissionUseCase(competition.SubmissionDeps{
		SubmissionRepo:   submissionRepo,
		TM:               TM,
		SolveCreator:     challengeUC,
		SolveDeleter:     challengeUC,
		CacheInvalidator: challengeUC,
		Logger:           l,
	})
}

func ProvideSubmissionBatcher(submissionRepo repo.SubmissionRepository, l logger.Logger) *competition.SubmissionBatcher {
	return competition.NewSubmissionBatcher(submissionRepo, competition.WithBatcherLogger(l))
}

func ProvideTagUseCase(tagRepo repo.TagRepository, challengeRepo repo.ChallengeRepository) *challenge.TagUseCase {
	return challenge.NewTagUseCase(challenge.TagDeps{TagRepo: tagRepo, ChallengeRepo: challengeRepo})
}

func ProvideFieldUseCase(fieldRepo repo.FieldRepository) *settings.FieldUseCase {
	return settings.NewFieldUseCase(settings.FieldDeps{FieldRepo: fieldRepo})
}

func ProvideFieldValidator(fieldRepo repo.FieldRepository) *settings.FieldValidator {
	return settings.NewFieldValidator(fieldRepo)
}

func ProvideNotificationUseCase(notifRepo repo.NotificationRepository, broadcaster *pkgWS.Broadcaster) *notification.NotificationUseCase {
	return notification.NewNotificationUseCase(notification.NotificationDeps{
		NotifRepo:   notifRepo,
		Broadcaster: broadcaster,
	})
}

func ProvidePageUseCase(pageRepo repo.PageRepository) *page.PageUseCase {
	return page.NewPageUseCase(page.PageDeps{PageRepo: pageRepo})
}

func ProvideCommentUseCase(commentRepo repo.CommentRepository, challengeRepo repo.ChallengeRepository, userRepo repo.UserRepository, tm repo.TransactionManager) *challenge.CommentUseCase {
	return challenge.NewCommentUseCase(challenge.CommentDeps{
		CommentRepo:   commentRepo,
		ChallengeRepo: challengeRepo,
		UserRepo:      userRepo,
		TM:            tm,
	})
}

func ProvideTrackingUseCase(trackingRepo repo.TrackingRepository) *user.TrackingUseCase {
	return user.NewTrackingUseCase(user.TrackingDeps{TrackingRepo: trackingRepo})
}

func ProvideBracketRepo(pool *pgxpool.Pool) *persistent.BracketRepo {
	return persistent.NewBracketRepo(pool)
}

func ProvideBracketUseCase(bracketRepo repo.BracketRepository, tm repo.TransactionManager) *competition.BracketUseCase {
	return competition.NewBracketUseCase(competition.BracketDeps{BracketRepo: bracketRepo, TM: tm})
}

func ProvideAPITokenRepo(pool *pgxpool.Pool) *persistent.APITokenRepo {
	return persistent.NewAPITokenRepo(pool)
}

func ProvideAPITokenUseCase(apiTokenRepo repo.APITokenRepository) *user.APITokenUseCase {
	return user.NewAPITokenUseCase(user.APITokenDeps{Repo: apiTokenRepo})
}

func ProvideFileUseCase(
	fileRepo repo.FileRepository,
	challengeRepo repo.ChallengeRepository,
	solveRepo repo.SolveRepository,
	storageProvider storage.Provider,
	cfg *config.Config,
) *challenge.FileUseCase {
	return challenge.NewFileUseCase(challenge.FileDeps{
		FileRepo:       fileRepo,
		ChallengeRepo:  challengeRepo,
		SolveRepo:      solveRepo,
		Storage:        storageProvider,
		Expiry:         cfg.PresignedExpiry,
		DownloadSecret: cfg.AccessSecret,
		BaseURL:        cfg.BaseURL,
	})
}

func ProvideBackupUseCase(
	competitionRepo repo.CompetitionRepository,
	challengeRepo repo.ChallengeRepository,
	hintRepo repo.HintRepository,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	awardRepo repo.AwardRepository,
	solveRepo repo.SolveRepository,
	submissionRepo repo.SubmissionRepository,
	fileRepo repo.FileRepository,
	backupRepo repo.BackupRepository,
	SettingsRepo repo.SettingsRepository,
	storageProvider storage.Provider,
	TM repo.TransactionManager,
	l logger.Logger,
) *backup.BackupUseCase {
	return backup.NewBackupUseCase(backup.BackupDeps{
		CompetitionRepo: competitionRepo,
		ChallengeRepo:   challengeRepo,
		HintRepo:        hintRepo,
		TeamRepo:        teamRepo,
		UserRepo:        userRepo,
		AwardRepo:       awardRepo,
		SolveRepo:       solveRepo,
		SubmissionRepo:  submissionRepo,
		FileRepo:        fileRepo,
		BackupRepo:      backupRepo,
		SettingsRepo:    SettingsRepo,
		Storage:         storageProvider,
		TM:              TM,
		Logger:          l,
	})
}

func ProvideSettingsUseCase(
	SettingsRepo repo.SettingsRepository,
	auditLogRepo repo.AuditLogRepository,
	TM repo.TransactionManager,
	kv cache.KeyValueStore,
	competitionRepo repo.CompetitionRepository,
) *settings.SettingsUseCase {
	return settings.NewSettingsUseCase(settings.SettingsDeps{
		Repo:         SettingsRepo,
		AuditLogRepo: auditLogRepo,
		TM:           TM,
		Redis:        kv,
		CompRepo:     competitionRepo,
	})
}

func ProvideCompetitionParamUseCase(
	paramRepo repo.CompetitionParamRepository,
	auditLogRepo repo.AuditLogRepository,
	TM repo.TransactionManager,
	l logger.Logger,
) *competition.CompetitionParamUseCase {
	return competition.NewCompetitionParamUseCase(competition.CompetitionParamDeps{
		Repo:         paramRepo,
		AuditLogRepo: auditLogRepo,
		TM:           TM,
		Logger:       l,
	})
}

func ProvideEmailUseCase(
	userRepo repo.UserRepository,
	tokenRepo repo.VerificationTokenRepository,
	TM repo.TransactionManager,
	mailer mailer.Mailer,
	cfg *config.Config,
) *email.EmailUseCase {
	return email.NewEmailUseCase(email.EmailDeps{
		UserRepo: userRepo, TokenRepo: tokenRepo, TM: TM, Mailer: mailer,
		VerifyTTL: cfg.VerifyTTL, ResetTTL: cfg.ResetTTL, FrontendURL: cfg.FrontendURL, Enabled: cfg.Enabled,
	})
}

func ProvideWsController(wsHub *pkgWS.Hub, l logger.Logger, cfg *config.Config) *wsController.Controller {
	return wsController.NewController(wsHub, l, cfg.CORSOrigins)
}

//nolint:funlen // Wire provider; many deps to assemble.
func ProvideServerDeps(
	cfg *config.Config,
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
	submissionBatcher *competition.SubmissionBatcher,
	tagUC *challenge.TagUseCase,
	fieldUC *settings.FieldUseCase,
	pageUC *page.PageUseCase,
	bracketUC *competition.BracketUseCase,
	notifUC *notification.NotificationUseCase,
	apiTokenUC *user.APITokenUseCase,
	backupUC *backup.BackupUseCase,
	settingsUC *settings.SettingsUseCase,
	competitionParamUC *competition.CompetitionParamUseCase,
	commentUC *challenge.CommentUseCase,
	trackingUC *user.TrackingUseCase,
	oauthUC *user.OAuthUseCase,
	jwtService *jwt.JWTService,
	redisClient *redis.Client,
	SettingsRepo repo.SettingsRepository,
	storageProvider storage.Provider,
	wsCtrl *wsController.Controller,
	v validator.Validator,
	l logger.Logger,
) (*helper.ServerDeps, error) {
	forgotLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyForgot, forgotPasswordRateLimit, perKeyRateLimitWindow)
	if err != nil {
		return nil, fmt.Errorf("wire - ProvideServerDeps - create forgot-password rate limiter: %w", err)
	}
	resendLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyResend, resendVerificationRateLimit, perKeyRateLimitWindow)
	if err != nil {
		return nil, fmt.Errorf("wire - ProvideServerDeps - create resend-verification rate limiter: %w", err)
	}
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
			UserUC:        userUC,
			EmailUC:       emailUC,
			APITokenUC:    apiTokenUC,
			TrackingUC:    trackingUC,
			OAuthUC:       oauthUC,
			FrontendURL:   cfg.FrontendURL,
			SecureCookies: strings.HasPrefix(cfg.BaseURL, "https://"),
		},
		Comp: helper.CompetitionDeps{
			CompetitionUC:     competitionUC,
			SolveUC:           solveUC,
			StatsUC:           statsUC,
			SubmissionUC:      submissionUC,
			SubmissionBatcher: submissionBatcher,
			BracketUC:         bracketUC,
		},
		Admin: helper.AdminDeps{
			BackupUC:           backupUC,
			SettingsUC:         settingsUC,
			CompetitionParamUC: competitionParamUC,
			FieldUC:            fieldUC,
			PageUC:             pageUC,
			NotifUC:            notifUC,
			SettingsRepo:       SettingsRepo,
		},
		Infra: helper.InfraDeps{
			JWTService:                    jwtService,
			RedisClient:                   redisClient,
			StorageProvider:               storageProvider,
			WSController:                  wsCtrl,
			Validator:                     v,
			Logger:                        l,
			TrustedProxyCIDRs:             cfg.TrustedProxyCIDRs,
			ForgotPasswordRateLimiter:     forgotLimiter,
			ResendVerificationRateLimiter: resendLimiter,
		},
	}, nil
}

//nolint:gocognit,funlen
func ProvideRouter(ctx context.Context, cfg *config.Config, l logger.Logger, deps *helper.ServerDeps) chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	if cfg.ChiMode == "production" {
		router.Use(restapimiddleware.Logger(l))
	} else {
		router.Use(middleware.Logger)
	}
	router.Use(restapimiddleware.Metrics)
	router.Use(middleware.Recoverer)
	timeoutMW := middleware.Timeout(requestTimeout)
	router.Use(func(next http.Handler) http.Handler {
		withTimeout := timeoutMW(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/ws") {
				next.ServeHTTP(w, r)
				return
			}
			withTimeout.ServeHTTP(w, r)
		})
	})
	router.Use(securityHeadersMiddleware)
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		status := map[string]string{}
		allOK := true

		if _, err := deps.Admin.SettingsRepo.Get(ctx); err != nil {
			status["db"] = "error"
			allOK = false
		} else {
			status["db"] = "ok"
		}

		if err := deps.Infra.RedisClient.Ping(ctx).Err(); err != nil {
			status["redis"] = "error"
			allOK = false
		} else {
			status["redis"] = "ok"
		}

		if err := deps.Infra.StorageProvider.Ping(ctx); err != nil {
			status["storage"] = "error"
			allOK = false
		} else {
			status["storage"] = "ok"
		}

		w.Header().Set("Content-Type", "application/json")
		if !allOK {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		jsonBytes, err := json.Marshal(status)
		if err != nil {
			return
		}
		_, _ = w.Write(jsonBytes) //nolint:errcheck
	})
	metricsHandler := promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	)
	if len(cfg.MetricsAllowedIPs) > 0 {
		metricsHandler = metricsAllowlistMiddleware(cfg.MetricsAllowedIPs, cfg.TrustedProxyCIDRs, metricsHandler)
	}
	router.Handle("/metrics", metricsHandler)
	rateLimitCache := helper.NewRateLimitConfigCache(rateLimitCacheTTL)
	generalIPLimitMiddleware := helper.RateLimitFromConfig(
		deps.Infra.RedisClient, rlKeyGeneral, time.Minute,
		rateLimitCache, deps.Admin.SettingsUC,
		func(c *helper.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		func(r *http.Request) (string, error) { return helper.GetClientIP(r, cfg.TrustedProxyCIDRs), nil },
		l,
	)
	router.Use(generalIPLimitMiddleware)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	router.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		swagger, err := openapi.GetSwagger()
		if err != nil {
			httputil.HandleError(w, r, err)
			return
		}
		jsonBytes, err := json.Marshal(swagger)
		if err != nil {
			httputil.HandleError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonBytes) //nolint:errcheck
	})
	router.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/openapi.json")))
	router.Route("/api/v1", func(r chi.Router) {
		v1.NewRouter(ctx, r, deps, cfg.VerifyEmails, rateLimitCache)
	})
	return router
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// API-only: restrict fetching to same-origin; allow JSON/form data only.
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		// Disable all browser feature APIs for API responses.
		w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		next.ServeHTTP(w, r)
	})
}

//nolint:gocognit
func metricsAllowlistMiddleware(allowedIPs, trustedProxyCIDRs []string, next http.Handler) http.Handler {
	nets := make([]*net.IPNet, 0, len(allowedIPs))
	ips := make([]net.IP, 0, len(allowedIPs))
	for _, s := range allowedIPs {
		if strings.Contains(s, "/") {
			_, n, err := net.ParseCIDR(s)
			if err != nil {
				continue
			}
			nets = append(nets, n)
		} else {
			ip := net.ParseIP(s)
			if ip != nil {
				ips = append(ips, ip)
			}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := helper.GetClientIP(r, trustedProxyCIDRs)
		ip := net.ParseIP(clientIP)
		if ip == nil {
			httputil.HandleError(w, r, httperr.ErrAccessDenied)
			return
		}
		for _, n := range nets {
			if n.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
		}
		for _, allowed := range ips {
			if ip.Equal(allowed) {
				next.ServeHTTP(w, r)
				return
			}
		}
		httputil.HandleError(w, r, httperr.ErrAccessDenied)
	})
}

func ProvideServer(router chi.Router, cfg *config.Config) *http.Server {
	return &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      router,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		IdleTimeout:  httpIdleTimeout,
	}
}

func ProvideApp(server *http.Server, userRepo repo.UserRepository, batcher usecase.SubmissionBatcher) *App {
	return &App{Server: server, UserRepo: userRepo, SubmissionBatcher: batcher}
}
