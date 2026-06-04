package wire

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	restapimiddleware "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	v1 "github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	wsController "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/loginlockout"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/webapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/avatar"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/email"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/setup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	iws "github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/mailer"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/sse"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

const (
	rlKeyForgot   = "forgot"
	rlKeyResend   = "resend"
	rlKeyResetTok = "reset-token"
	rlKeyGeneral  = "general:ip"

	requestTimeout    = 60 * time.Second
	rateLimitCacheTTL = 30 * time.Second

	httpReadTimeout  = 15 * time.Second
	httpWriteTimeout = 100 * time.Second
	httpIdleTimeout  = time.Minute

	loginLockoutMaxAttempts      = 5
	loginLockoutTTL              = time.Minute
	forgotPasswordRateLimit      = 10
	resendVerificationRateLimit  = 10
	resetPasswordTokenRateLimit  = 5
	resetPasswordTokenRateWindow = time.Minute
	perKeyRateLimitWindow        = 24 * time.Hour
)

type healthCheckerFunc func(context.Context) error

func (f healthCheckerFunc) Check(ctx context.Context) error {
	return f(ctx)
}

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

func ProvideRatingRepo(pool *pgxpool.Pool) *persistent.RatingRepo {
	return persistent.NewRatingRepo(pool)
}

func ProvideSettingsRepo(pool *pgxpool.Pool) *persistent.SettingsRepo {
	return persistent.NewSettingsRepo(pool)
}

func ProvideBanAppealRepo(pool *pgxpool.Pool) *persistent.BanAppealRepo {
	return persistent.NewBanAppealRepo(pool)
}

func ProvideBanAppealUseCase(
	appealRepo repo.BanAppealRepository,
	userRepo repo.UserRepository,
	tm repo.TransactionManager,
) *user.BanAppealUseCase {
	return user.NewBanAppealUseCase(appealRepo, userRepo, tm)
}

func ProvideCompetitionParamRepo(pool *pgxpool.Pool) *persistent.CompetitionParamRepo {
	return persistent.NewCompetitionParamRepo(pool)
}

func ProvideVerificationTokenRepo(pool *pgxpool.Pool) *persistent.VerificationTokenRepo {
	return persistent.NewVerificationTokenRepo(pool)
}

func ProvideOAuthRepo(pool *pgxpool.Pool, cryptoService crypto.Service) *persistent.OAuthRepo {
	return persistent.NewOAuthRepo(pool, cryptoService)
}

func ProvideOAuthProviders() map[string]user.OAuthProviderAPI {
	return map[string]user.OAuthProviderAPI{
		"github": webapi.NewGitHubAPI(nil),
		"google": webapi.NewGoogleAPI(nil),
	}
}

func ProvideOAuthConfig(cfg *config.Config) user.OAuthConfig {
	return user.OAuthConfig{
		StateSecret: cfg.StateSecret,
		GitHub: user.OAuthProviderConfig{
			ClientID:     cfg.GitHub.ClientID,
			ClientSecret: cfg.GitHub.ClientSecret,
			RedirectURL:  cfg.GitHub.RedirectURL,
		},
		Google: user.OAuthProviderConfig{
			ClientID:     cfg.Google.ClientID,
			ClientSecret: cfg.Google.ClientSecret,
			RedirectURL:  cfg.Google.RedirectURL,
		},
	}
}

func ProvideOAuthUseCase(
	userRepo repo.UserRepository,
	oauthRepo repo.OAuthAccountRepository,
	TM repo.TransactionManager,
	settingsRepo repo.SettingsRepository,
	jwtService *jwtkit.JWTService,
	providers map[string]user.OAuthProviderAPI,
	oauthCfg user.OAuthConfig,
	compRepo repo.CompetitionRepository,
	soloTeamCreator user.SoloTeamCreator,
	l logkit.Logger,
) *user.OAuthUseCase {
	return user.NewOAuthUseCase(user.OAuthDeps{
		UserRepo:        userRepo,
		OAuthRepo:       oauthRepo,
		TM:              TM,
		SettingsRepo:    settingsRepo,
		JWTService:      jwtService,
		Providers:       providers,
		Cfg:             oauthCfg,
		CompRepo:        compRepo,
		SoloTeamCreator: soloTeamCreator,
		Logger:          l,
	})
}

func ProvideAvatarUseCase(
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	storageProvider storage.Provider,
	keyValueStore cachekit.KeyValueStore,
	TM repo.TransactionManager,
	auditLogRepo repo.AuditLogRepository,
	l logkit.Logger,
) *avatar.AvatarUseCase {
	return avatar.NewAvatarUseCase(avatar.AvatarDeps{
		UserRepo:     userRepo,
		TeamRepo:     teamRepo,
		Storage:      storageProvider,
		Cache:        keyValueStore,
		TM:           TM,
		AuditLogRepo: auditLogRepo,
		Config:       domain.GetDefaultAvatarConfig(),
		Logger:       l,
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
	challengeRepo repo.ChallengeRepository,
	submissionRepo repo.SubmissionRepository,
	awardRepo repo.AwardRepository,
	hintRepo repo.HintRepository,
	TM repo.TransactionManager,
	jwtService *jwtkit.JWTService,
	fieldValidator *settings.FieldValidator,
	fieldValueRepo repo.FieldValueRepository,
	settingsRepo repo.SettingsRepository,
	emailUC *email.EmailUseCase,
	failedLoginTracker *loginlockout.Tracker,
	compRepo repo.CompetitionRepository,
	soloTeamCreator user.SoloTeamCreator,
	notificationUC *notification.NotificationUseCase,
	userCacheSvc *cache.UserCacheService,
	scoreboardCache *cache.ScoreboardCacheService,
	challengeListCache cache.ChallengeListCacheInvalidator,
	sharedCache *cachekit.Cache,
	compParamUC *competition.CompetitionParamUseCase,
	l logkit.Logger,
) *user.UserUseCase {
	return user.NewUserUseCase(user.UserDeps{
		UserRepo: userRepo, TeamRepo: teamRepo, SolveRepo: solveRepo,
		ChallengeRepo:  challengeRepo,
		SubmissionRepo: submissionRepo, AwardRepo: awardRepo, HintRepo: hintRepo,
		TM: TM, JWTService: jwtService,
		FieldValidator: fieldValidator, FieldValueRepo: fieldValueRepo,
		SettingsRepo: settingsRepo, EmailSender: emailUC, FailedLogin: failedLoginTracker,
		CompRepo: compRepo, SoloTeamCreator: soloTeamCreator,
		PersonalNotificationSender: notificationUC,
		UserCache:                  userCacheSvc,
		ScoreboardCache:            scoreboardCache,
		ChallengeListCache:         challengeListCache,
		TeamCache:                  sharedCache,
		CompParamUC:                compParamUC,
		Logger:                     l,
	})
}

type teamBracketIDGetter struct {
	r repo.TeamRepository
}

func (g *teamBracketIDGetter) GetTeamBracketID(ctx context.Context, teamID uuid.UUID) (*uuid.UUID, error) {
	team, err := g.r.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("teamBracketIDGetter - GetTeamBracketID - TeamRepo.GetByID: %w", err)
	}

	if team == nil {
		return nil, fmt.Errorf("teamBracketIDGetter - GetTeamBracketID: team %s not found", teamID)
	}

	return team.BracketID, nil
}

func ProvideScoreboardCacheService(c *cachekit.Cache, teamRepo repo.TeamRepository) *cache.ScoreboardCacheService {
	return cache.NewScoreboardCacheService(c, &teamBracketIDGetter{r: teamRepo})
}

func ProvideUserCacheService(c *cachekit.Cache) *cache.UserCacheService {
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
	compParamUC *competition.CompetitionParamUseCase,
	TM repo.TransactionManager,
	guard usecase.CompetitionGuard,
	scoreboardCache *cache.ScoreboardCacheService,
	challengeListCache cache.ChallengeListCacheInvalidator,
	userCacheSvc *cache.UserCacheService,
	sharedCache *cachekit.Cache,
	hintRepo repo.HintRepository,
	ratingRepo repo.RatingRepository,
	fieldValueRepo repo.FieldValueRepository,
	jwtService *jwtkit.JWTService,
	l logkit.Logger,
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
		CompParamUC:        compParamUC,
		TM:                 TM,
		Guard:              guard,
		ScoreboardCache:    scoreboardCache,
		ChallengeListCache: challengeListCache,
		UserCache:          userCacheSvc,
		TeamCache:          sharedCache,
		HintRepo:           hintRepo,
		RatingRepo:         ratingRepo,
		FieldValueRepo:     fieldValueRepo,
		JWTRevoker:         jwtService,
		DefaultMaxTeamSize: cfg.MaxTeamSize,
		Logger:             l,
	})
}

func ProvideAwardUseCase(
	awardRepo repo.AwardRepository,
	teamRepo repo.TeamRepository,
	TM repo.TransactionManager,
	scoreboardCache *cache.ScoreboardCacheService,
	compRepo repo.CompetitionRepository,
) *team.AwardUseCase {
	return team.NewAwardUseCase(team.AwardDeps{
		AwardRepo:       awardRepo,
		TeamRepo:        teamRepo,
		TM:              TM,
		ScoreboardCache: scoreboardCache,
		CompRepo:        compRepo,
	})
}

func ProvideChallengeUseCase(
	challengeRepo repo.ChallengeRepository,
	tagRepo repo.TagRepository,
	solveRepo repo.SolveRepository,
	submissionRepo repo.SubmissionRepository,
	TM repo.TransactionManager,
	compRepo repo.CompetitionRepository,
	compUC *competition.CompetitionUseCase,
	compParamUC *competition.CompetitionParamUseCase,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	scoreboardCache *cache.ScoreboardCacheService,
	c *cachekit.Cache,
	broadcaster *iws.Broadcaster,
	auditLogRepo repo.AuditLogRepository,
	cryptoService crypto.Service,
	fileRepo repo.FileRepository,
	storageProvider storage.Provider,
	hintUC *challenge.HintUseCase,
	l logkit.Logger,
) *challenge.ChallengeUseCase {
	return challenge.NewChallengeUseCase(challenge.ChallengeDeps{
		ChallengeRepo:   challengeRepo,
		TagRepo:         tagRepo,
		SolveRepo:       solveRepo,
		SubmissionRepo:  submissionRepo,
		TM:              TM,
		CompRepo:        compRepo,
		CompUC:          compUC,
		CompParamUC:     compParamUC,
		TeamRepo:        teamRepo,
		UserRepo:        userRepo,
		ScoreboardCache: scoreboardCache,
		ListCache:       c,
		Broadcaster:     broadcaster,
		AuditLogRepo:    auditLogRepo,
		Crypto:          cryptoService,
		FileRepo:        fileRepo,
		Storage:         storageProvider,
		HintUC:          hintUC,
		SolveRecord:     competition.RecordSolveInTx,
		Logger:          l,
	})
}

func ProvideHintUseCase(
	hintRepo repo.HintRepository,
	awardRepo repo.AwardRepository,
	TM repo.TransactionManager,
	solveRepo repo.SolveRepository,
	compRepo repo.CompetitionRepository,
	competitionUC *competition.CompetitionUseCase,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	challengeRepo repo.ChallengeRepository,
	scoreboardCache *cache.ScoreboardCacheService,
	l logkit.Logger,
) *challenge.HintUseCase {
	return challenge.NewHintUseCase(challenge.HintDeps{
		HintRepo:        hintRepo,
		AwardRepo:       awardRepo,
		TM:              TM,
		SolveRepo:       solveRepo,
		CompRepo:        compRepo,
		CompUC:          competitionUC,
		TeamRepo:        teamRepo,
		UserRepo:        userRepo,
		ChallengeRepo:   challengeRepo,
		ScoreboardCache: scoreboardCache,
		Logger:          l,
	})
}

func ProvideCompetitionUseCase(
	competitionRepo repo.CompetitionRepository,
	auditLogRepo repo.AuditLogRepository,
	TM repo.TransactionManager,
	kv cachekit.KeyValueStore,
	statsCache *cachekit.Cache,
	scoreboardCache cache.ScoreboardCacheInvalidator,
	l logkit.Logger,
) *competition.CompetitionUseCase {
	var statsInvalidator competition.StatisticsCacheInvalidator

	if statsCache != nil {
		statsInvalidator = &competition.StatsCacheInvalidatorImpl{Cache: statsCache}
	}

	return competition.NewCompetitionUseCase(competition.CompetitionDeps{
		CompetitionRepo:       competitionRepo,
		AuditLogRepo:          auditLogRepo,
		TM:                    TM,
		Redis:                 kv,
		StatsCacheInvalidator: statsInvalidator,
		ScoreboardCache:       scoreboardCache,
		Logger:                l,
	})
}

func ProvideSolveUseCase(
	solveRepo repo.SolveRepository,
	challengeRepo repo.ChallengeRepository,
	competitionRepo repo.CompetitionRepository,
	competitionUC usecase.CompetitionUseCase,
	compParamUC *competition.CompetitionParamUseCase,
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	TM repo.TransactionManager,
	c *cachekit.Cache,
	scoreboardCache *cache.ScoreboardCacheService,
	challengeListCache cache.ChallengeListCacheInvalidator,
	broadcaster *iws.Broadcaster,
) *competition.SolveUseCase {
	return competition.NewSolveUseCase(competition.SolveDeps{
		SolveRepo:          solveRepo,
		ChallengeRepo:      challengeRepo,
		CompetitionRepo:    competitionRepo,
		CompetitionUC:      competitionUC,
		CompParamUC:        compParamUC,
		UserRepo:           userRepo,
		TeamRepo:           teamRepo,
		TM:                 TM,
		Cache:              c,
		ScoreboardCache:    scoreboardCache,
		ChallengeListCache: challengeListCache,
		Broadcaster:        broadcaster,
	})
}

func ProvideBroadcaster(hub *wskit.Hub) *iws.Broadcaster {
	return iws.NewBroadcaster(hub)
}

func ProvideCache(r *redis.Client) *cachekit.Cache {
	return cachekit.New(r)
}

func ProvideKeyValueStore(r *redis.Client) cachekit.KeyValueStore {
	return &cachekit.RedisKeyValueStore{Client: r}
}

func ProvidePubSubStore(r *redis.Client) cachekit.PubSubStore {
	return &cachekit.RedisPubSubStore{Client: r}
}

func ProvideStatisticsUseCase(
	statsRepo repo.StatisticsRepository,
	c *cachekit.Cache,
	compUC *competition.CompetitionUseCase,
	TM repo.TransactionManager,
) *competition.StatisticsUseCase {
	return competition.NewStatisticsUseCase(competition.StatisticsDeps{
		StatsRepo:  statsRepo,
		Cache:      c,
		CompGetter: compUC,
		TM:         TM,
	})
}

func ProvideSubmissionUseCase(submissionRepo repo.SubmissionRepository, competitionUC *competition.CompetitionUseCase, TM repo.TransactionManager, challengeUC *challenge.ChallengeUseCase, userRepo repo.UserRepository, teamRepo repo.TeamRepository, l logkit.Logger) *competition.SubmissionUseCase {
	return competition.NewSubmissionUseCase(competition.SubmissionDeps{
		SubmissionRepo:   submissionRepo,
		CompGetter:       competitionUC,
		TM:               TM,
		SolveCreator:     challengeUC,
		SolveDeleter:     challengeUC,
		CacheInvalidator: challengeUC,
		Logger:           l,
		UserRepo:         userRepo,
		TeamRepo:         teamRepo,
	})
}

func ProvideSubmissionBatcher(submissionRepo repo.SubmissionRepository, l logkit.Logger) *competition.SubmissionBatcher {
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

func ProvideNotificationUseCase(notifRepo repo.NotificationRepository, broadcaster *iws.Broadcaster, l logkit.Logger) *notification.NotificationUseCase {
	return notification.NewNotificationUseCase(notification.NotificationDeps{
		NotifRepo:   notifRepo,
		Broadcaster: broadcaster,
		Logger:      l,
	})
}

func ProvidePageUseCase(pageRepo repo.PageRepository, l logkit.Logger) *page.PageUseCase {
	return page.NewPageUseCase(page.PageDeps{PageRepo: pageRepo, Logger: l})
}

func ProvideCommentUseCase(commentRepo repo.CommentRepository, challengeRepo repo.ChallengeRepository, userRepo repo.UserRepository, teamRepo repo.TeamRepository, tm repo.TransactionManager) *challenge.CommentUseCase {
	return challenge.NewCommentUseCase(challenge.CommentDeps{
		CommentRepo:   commentRepo,
		ChallengeRepo: challengeRepo,
		UserRepo:      userRepo,
		TeamRepo:      teamRepo,
		TM:            tm,
	})
}

func ProvideRatingUseCase(challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository, ratingRepo repo.RatingRepository, userRepo repo.UserRepository, teamRepo repo.TeamRepository, tm repo.TransactionManager) *challenge.RatingUseCase {
	return challenge.NewRatingUseCase(challenge.RatingDeps{
		ChallengeRepo: challengeRepo,
		SolveRepo:     solveRepo,
		RatingRepo:    ratingRepo,
		UserRepo:      userRepo,
		TeamRepo:      teamRepo,
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
		DownloadSecret: cfg.DownloadSecret,
		BaseURL:        cfg.BaseURL,
	})
}

func ProvideBackupUseCase(
	competitionRepo repo.CompetitionRepository,
	challengeRepo repo.ChallengeRepository,
	tagRepo repo.TagRepository,
	hintRepo repo.HintRepository,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	awardRepo repo.AwardRepository,
	solveRepo repo.SolveRepository,
	submissionRepo repo.SubmissionRepository,
	fileRepo repo.FileRepository,
	backupRepo repo.BackupRepository,
	settingsRepo repo.SettingsRepository,
	auditLogRepo repo.AuditLogRepository,
	bracketRepo repo.BracketRepository,
	commentRepo repo.CommentRepository,
	fieldRepo repo.FieldRepository,
	fieldValueRepo repo.FieldValueRepository,
	ratingRepo repo.RatingRepository,
	storageProvider storage.Provider,
	TM repo.TransactionManager,
	l logkit.Logger,
) *backup.BackupUseCase {
	return backup.NewBackupUseCase(backup.BackupDeps{
		CompetitionRepo: competitionRepo,
		ChallengeRepo:   challengeRepo,
		TagRepo:         tagRepo,
		HintRepo:        hintRepo,
		TeamRepo:        teamRepo,
		UserRepo:        userRepo,
		AwardRepo:       awardRepo,
		SolveRepo:       solveRepo,
		SubmissionRepo:  submissionRepo,
		FileRepo:        fileRepo,
		BackupRepo:      backupRepo,
		SettingsRepo:    settingsRepo,
		AuditLogRepo:    auditLogRepo,
		BracketRepo:     bracketRepo,
		CommentRepo:     commentRepo,
		FieldRepo:       fieldRepo,
		FieldValueRepo:  fieldValueRepo,
		RatingRepo:      ratingRepo,
		Storage:         storageProvider,
		TM:              TM,
		Logger:          l,
	})
}

func ProvideSettingsUseCase(
	ctx context.Context,
	settingsRepo repo.SettingsRepository,
	auditLogRepo repo.AuditLogRepository,
	TM repo.TransactionManager,
	kv cachekit.KeyValueStore,
	competitionRepo repo.CompetitionRepository,
	competitionParamUC *competition.CompetitionParamUseCase,
	pubsub cachekit.PubSubStore,
	l logkit.Logger,
) *settings.SettingsUseCase {
	return settings.NewSettingsUseCase(settings.SettingsDeps{
		Repo:         settingsRepo,
		AuditLogRepo: auditLogRepo,
		TM:           TM,
		Redis:        kv,
		CompRepo:     competitionRepo,
		ConfigUC:     competitionParamUC,
		PubSub:       pubsub,
		StopContext:  ctx,
		Logger:       l,
	})
}

func ProvideCompetitionParamUseCase(
	ctx context.Context,
	paramRepo repo.CompetitionParamRepository,
	auditLogRepo repo.AuditLogRepository,
	TM repo.TransactionManager,
	l logkit.Logger,
	kv cachekit.KeyValueStore,
	pubsub cachekit.PubSubStore,
) *competition.CompetitionParamUseCase {
	return competition.NewCompetitionParamUseCase(competition.CompetitionParamDeps{
		Repo:         paramRepo,
		AuditLogRepo: auditLogRepo,
		TM:           TM,
		Logger:       l,
		Cache:        kv,
		PubSub:       pubsub,
		StopContext:  ctx,
	})
}

func ProvideEmailUseCase(
	userRepo repo.UserRepository,
	tokenRepo repo.VerificationTokenRepository,
	TM repo.TransactionManager,
	mailer mailer.Mailer,
	cfg *config.Config,
	competitionParamUC *competition.CompetitionParamUseCase,
	jwtService *jwtkit.JWTService,
	l logkit.Logger,
) *email.EmailUseCase {
	return email.NewEmailUseCase(email.EmailDeps{
		UserRepo: userRepo, TokenRepo: tokenRepo, TM: TM, Mailer: mailer,
		ConfigUC:   competitionParamUC,
		JWTRevoker: jwtService,
		VerifyTTL:  cfg.VerifyTTL, ResetTTL: cfg.ResetTTL, FrontendURL: cfg.FrontendURL, Enabled: cfg.Enabled,
		Logger: l,
	})
}

func ProvideWsController(wsHub *wskit.Hub, l logkit.Logger, cfg *config.Config) *wsController.Controller {
	return wsController.NewController(wsHub, l, cfg.CORSOrigins)
}

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
	ratingUC *challenge.RatingUseCase,
	trackingUC *user.TrackingUseCase,
	oauthUC *user.OAuthUseCase,
	avatarUC *avatar.AvatarUseCase,
	appealUC *user.BanAppealUseCase,
	jwtService *jwtkit.JWTService,
	redisClient *redis.Client,
	storageProvider storage.Provider,
	wsCtrl *wsController.Controller,
	wsHub *wskit.Hub,
	v validator.Validator,
	l logkit.Logger,
) (*helper.ServerDeps, error) {
	forgotLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyForgot, forgotPasswordRateLimit, perKeyRateLimitWindow)
	if err != nil {
		return nil, fmt.Errorf("ProvideServerDeps - create forgot-password rate limiter: %w", err)
	}

	resendLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyResend, resendVerificationRateLimit, perKeyRateLimitWindow)
	if err != nil {
		return nil, fmt.Errorf("ProvideServerDeps - create resend-verification rate limiter: %w", err)
	}

	resetTokenLimiter, err := restapimiddleware.NewPerKeyRateLimiter(redisClient, rlKeyResetTok, resetPasswordTokenRateLimit, resetPasswordTokenRateWindow)
	if err != nil {
		return nil, fmt.Errorf("ProvideServerDeps - create reset-password-token rate limiter: %w", err)
	}

	rateLimitCache := restapimiddleware.NewRateLimitConfigCache(rateLimitCacheTTL)
	ratelimitAuditWG := &sync.WaitGroup{}

	return &helper.ServerDeps{
		Challenge: helper.ChallengeDeps{
			ChallengeUC: challengeUC,
			HintUC:      hintUC,
			FileUC:      fileUC,
			TagUC:       tagUC,
			CommentUC:   commentUC,
			RatingUC:    ratingUC,
		},
		Team: helper.TeamDeps{
			TeamUC:  teamUC,
			AwardUC: awardUC,
		},
		User: helper.UserDeps{
			UserUC:              userUC,
			EmailUC:             emailUC,
			APITokenUC:          apiTokenUC,
			TrackingUC:          trackingUC,
			OAuthUC:             oauthUC,
			AvatarUC:            avatarUC,
			AppealUC:            appealUC,
			FrontendURL:         cfg.FrontendURL,
			SecureCookies:       cfg.SecureCookies,
			RefreshCookieMaxAge: int(cfg.RefreshTTL.Seconds()),
			OAuthGitHubEnabled:  cfg.GitHub.IsConfigured(),
			OAuthGoogleEnabled:  cfg.Google.IsConfigured(),
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
		},
		Infra: helper.InfraDeps{
			JWTService:                    jwtService,
			RedisClient:                   redisClient,
			StorageProvider:               storageProvider,
			WSController:                  wsCtrl,
			SSEHandler:                    sse.NewSSEHandler(wsHub, l),
			Validator:                     v,
			Logger:                        l,
			TrustedProxyCIDRs:             cfg.TrustedProxyCIDRs,
			StructuredLogger:              cfg.StructuredLogger,
			DebugEnabled:                  cfg.DebugEnabled,
			RateLimitConfigCache:          rateLimitCache,
			ForgotPasswordRateLimiter:     forgotLimiter,
			ResendVerificationRateLimiter: resendLimiter,
			ResetPasswordTokenRateLimiter: resetTokenLimiter,
			RatelimitAuditWG:              ratelimitAuditWG,
		},
	}, nil
}

func ProvideRouter(ctx context.Context, cfg *config.Config, l logkit.Logger, deps *helper.ServerDeps) chi.Router {
	router := chi.NewRouter()
	router.Use(kitMiddleware.RequestID())

	clientIP, err := kitMiddleware.ClientIP(cfg.TrustedProxyCIDRs)
	if err != nil {
		panic(err)
	}

	router.Use(clientIP)

	if cfg.StructuredLogger {
		router.Use(kitMiddleware.Logger(l, cfg.TrustedProxyCIDRs))
	} else {
		router.Use(middleware.Logger)
	}

	router.Use(kitMiddleware.Metrics(prometheus.DefaultRegisterer, httputil.ChiPathFromRequest))
	router.Use(kitMiddleware.Recoverer(l))

	// CORS must be registered before any rate-limit middleware so that 429
	// responses (short-circuited by httprate.WithLimitHandler) still carry
	// Access-Control-Allow-Origin headers and the browser doesn't report a
	// spurious CORS error instead of a rate-limit error.
	if len(cfg.CORSOrigins) > 0 {
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	timeoutMW := kitMiddleware.Timeout(requestTimeout)

	router.Use(func(next http.Handler) http.Handler {
		withTimeout := timeoutMW(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/ws") || strings.HasSuffix(r.URL.Path, "/sse") {
				next.ServeHTTP(w, r)

				return
			}

			withTimeout.ServeHTTP(w, r)
		})
	})

	strictSecurity := kitMiddleware.SecurityHeaders(true)
	swaggerDocCSP := "default-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' 'unsafe-eval'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'"
	relaxedSecurity := kitMiddleware.SecurityHeaders(true, kitMiddleware.WithCSP(swaggerDocCSP))

	router.Use(func(next http.Handler) http.Handler {
		strictH := strictSecurity(next)
		relaxedH := relaxedSecurity(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSwaggerOrOpenAPIDocPath(r.URL.Path) {
				relaxedH.ServeHTTP(w, r)

				return
			}

			strictH.ServeHTTP(w, r)
		})
	})

	generalIPLimitMiddleware := restapimiddleware.DynamicRateLimit(
		deps.Infra.RedisClient, rlKeyGeneral, time.Minute,
		deps.Infra.RateLimitConfigCache, deps.Admin.SettingsUC,
		func(c *restapimiddleware.RateLimitConfig) int64 { return int64(c.GeneralIPPerMinute) },
		func(r *http.Request) (string, error) {
			return kitMiddleware.GetClientIPFromContext(r.Context()), nil
		},
		l,
	)
	router.Use(generalIPLimitMiddleware)

	healthHandler := httputil.HealthHandler(map[string]httputil.Checker{
		"db": healthCheckerFunc(func(ctx context.Context) error {
			_, err := deps.Admin.SettingsUC.Get(ctx)

			return err
		}),
		"redis":   healthCheckerFunc(func(ctx context.Context) error { return deps.Infra.RedisClient.Ping(ctx).Err() }),
		"storage": healthCheckerFunc(func(ctx context.Context) error { return deps.Infra.StorageProvider.Ping(ctx) }),
	})
	router.Get("/health", healthHandler)

	metricsHandler := promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	)

	metricsHandler = metricsAllowlistMiddleware(cfg.MetricsAllowedIPs, metricsHandler)

	router.Handle("/metrics", metricsHandler)

	openapiJSONHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		swagger, err := openapi.GetSwagger()
		if err != nil {
			httputil.HandleError(w, r, err)

			return
		}

		httputil.RenderOK(w, r, swagger)
	})
	router.Get("/openapi.json", openapiJSONHandler)
	router.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/openapi.json")))

	deps.Infra.ScoreboardVisibilityCache = restapimiddleware.NewScoreboardVisibilityCache()

	setupUC := setup.NewSetupUseCase(setup.SetupDeps{
		UserUC:      deps.User.UserUC,
		CompUC:      deps.Comp.CompetitionUC,
		CompParamUC: deps.Admin.CompetitionParamUC,
		SettingsUC:  deps.Admin.SettingsUC,
		JWTService:  deps.Infra.JWTService,
	})
	setupHandler := v1.NewSetupHandler(setupUC, l, deps.Infra.Validator, cfg.SetupToken, cfg.SecureCookies, int(cfg.RefreshTTL.Seconds()))

	// Paths that remain accessible before setup is complete.
	setupAllowedPaths := []string{
		"/api/v1/setup",
		"/api/v1/configs/public",
		"/api/v1/health",
		"/api/v1/healthcheck",
		"/health",
		"/metrics",
		"/avatars/",
	}
	setupRequiredMW := restapimiddleware.SetupRequired(setupUC, setupAllowedPaths)

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthHandler)
		r.Handle("/metrics", metricsHandler)
		r.Get("/openapi.json", openapiJSONHandler)
		r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/api/v1/openapi.json")))

		// Setup endpoints - always accessible, outside the SetupRequired gate.
		r.Get("/setup/status", setupHandler.GetSetupStatus)
		r.Post("/setup", setupHandler.PostSetup)

		// All other API routes gated by setup completion.
		r.Group(func(gated chi.Router) {
			gated.Use(setupRequiredMW)
			v1.NewRouter(ctx, gated, deps, cfg.VerifyEmails, deps.Infra.RateLimitConfigCache)
		})
	})

	return router
}

func metricsAllowlistMiddleware(allowedIPs []string, next http.Handler) http.Handler {
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
		clientIP := kitMiddleware.GetClientIPFromContext(r.Context())

		ip := net.ParseIP(clientIP)
		if ip == nil {
			httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrAccessDenied))

			return
		}

		for _, n := range nets {
			if n.Contains(ip) {
				next.ServeHTTP(w, r)

				return
			}
		}

		if slices.ContainsFunc(ips, ip.Equal) {
			next.ServeHTTP(w, r)

			return
		}

		httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrAccessDenied))
	})
}

func isSwaggerOrOpenAPIDocPath(path string) bool {
	switch path {
	case "/openapi.json", "/api/v1/openapi.json", "/swagger", "/api/v1/swagger":
		return true
	default:
		return strings.HasPrefix(path, "/swagger/") || strings.HasPrefix(path, "/api/v1/swagger/")
	}
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

func ProvideApp(server *http.Server, userRepo repo.UserRepository, batcher usecase.SubmissionBatcher, solveUC *competition.SolveUseCase, avatarUC *avatar.AvatarUseCase, serverDeps *helper.ServerDeps, broadcaster *iws.Broadcaster) *App {
	return &App{
		Server:            server,
		UserRepo:          userRepo,
		SubmissionBatcher: batcher,
		SolveUseCase:      solveUC,
		AvatarUC:          avatarUC,
		RatelimitAuditWG:  serverDeps.Infra.RatelimitAuditWG,
		Broadcaster:       broadcaster,
	}
}
