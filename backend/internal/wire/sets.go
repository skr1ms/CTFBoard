package wire

import (
	"github.com/google/wire"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/user"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
)

var RepoSet = wire.NewSet(
	ProvideUserRepo,
	ProvideChallengeRepo,
	ProvideSolveRepo,
	ProvideTeamRepo,
	ProvideCompetitionRepo,
	ProvideHintRepo,
	ProvideAwardRepo,
	ProvideAuditLogRepo,
	ProvideStatisticsRepo,
	ProvideFileRepo,
	ProvideTransactionManager,
	ProvideBackupRepo,
	ProvideSubmissionRepo,
	ProvideTagRepo,
	ProvideFieldRepo,
	ProvideFieldValueRepo,
	ProvideNotificationRepo,
	ProvidePageRepo,
	ProvideCommentRepo,
	ProvideRatingRepo,
	ProvideBracketRepo,
	ProvideAPITokenRepo,
	ProvideSettingsRepo,
	ProvideCompetitionParamRepo,
	ProvideVerificationTokenRepo,
	ProvideTrackingRepo,
	ProvideOAuthRepo,
	wire.Bind(new(repo.UserRepository), new(*persistent.UserRepo)),
	wire.Bind(new(repo.TeamRepository), new(*persistent.TeamRepo)),
	wire.Bind(new(repo.SolveRepository), new(*persistent.SolveRepo)),
	wire.Bind(new(repo.CompetitionRepository), new(*persistent.CompetitionRepo)),
	wire.Bind(new(repo.ChallengeRepository), new(*persistent.ChallengeRepo)),
	wire.Bind(new(repo.HintRepository), new(*persistent.HintRepo)),
	wire.Bind(new(repo.AwardRepository), new(*persistent.AwardRepo)),
	wire.Bind(new(repo.AuditLogRepository), new(*persistent.AuditLogRepo)),
	wire.Bind(new(repo.StatisticsRepository), new(*persistent.StatisticsRepo)),
	wire.Bind(new(repo.FileRepository), new(*persistent.FileRepo)),
	wire.Bind(new(repo.TransactionManager), new(*persistent.TransactionManager)),
	wire.Bind(new(repo.BackupRepository), new(*persistent.BackupRepo)),
	wire.Bind(new(repo.SubmissionRepository), new(*persistent.SubmissionRepo)),
	wire.Bind(new(repo.TagRepository), new(*persistent.TagRepo)),
	wire.Bind(new(repo.FieldRepository), new(*persistent.FieldRepo)),
	wire.Bind(new(repo.FieldValueRepository), new(*persistent.FieldValueRepo)),
	wire.Bind(new(repo.NotificationRepository), new(*persistent.NotificationRepo)),
	wire.Bind(new(repo.PageRepository), new(*persistent.PageRepo)),
	wire.Bind(new(repo.CommentRepository), new(*persistent.CommentRepo)),
	wire.Bind(new(repo.RatingRepository), new(*persistent.RatingRepo)),
	wire.Bind(new(repo.BracketRepository), new(*persistent.BracketRepo)),
	wire.Bind(new(repo.APITokenRepository), new(*persistent.APITokenRepo)),
	wire.Bind(new(repo.SettingsRepository), new(*persistent.SettingsRepo)),
	wire.Bind(new(repo.CompetitionParamRepository), new(*persistent.CompetitionParamRepo)),
	wire.Bind(new(repo.VerificationTokenRepository), new(*persistent.VerificationTokenRepo)),
	wire.Bind(new(repo.TrackingRepository), new(*persistent.TrackingRepo)),
	wire.Bind(new(repo.OAuthAccountRepository), new(*persistent.OAuthRepo)),
)

var UseCaseSet = wire.NewSet(
	ProvideScoreboardCacheService,
	ProvideUserCacheService,
	ProvideCompetitionGuard,
	ProvideFailedLoginTracker,
	ProvideUserUseCase,
	ProvideChallengeUseCase,
	ProvideTeamUseCase,
	ProvideAwardUseCase,
	ProvideHintUseCase,
	ProvideCompetitionUseCase,
	ProvideSolveUseCase,
	ProvideStatisticsUseCase,
	ProvideSubmissionUseCase,
	ProvideSubmissionBatcher,
	ProvideTagUseCase,
	ProvideFieldUseCase,
	ProvideFieldValidator,
	ProvideNotificationUseCase,
	ProvidePageUseCase,
	ProvideCommentUseCase,
	ProvideRatingUseCase,
	ProvideBracketUseCase,
	ProvideAPITokenUseCase,
	ProvideFileUseCase,
	ProvideBackupUseCase,
	ProvideSettingsUseCase,
	ProvideCompetitionParamUseCase,
	ProvideEmailUseCase,
	ProvideTrackingUseCase,
	ProvideOAuthUseCase,
	ProvideOAuthProviders,
	wire.Bind(new(usecase.BackupUseCase), new(*backup.BackupUseCase)),
	wire.Bind(new(usecase.CompetitionUseCase), new(*competition.CompetitionUseCase)),
	wire.Bind(new(usecase.CompetitionParamUseCase), new(*competition.CompetitionParamUseCase)),
	wire.Bind(new(usecase.CompetitionGuard), new(*competition.Guard)),
	wire.Bind(new(usecase.NotificationUseCase), new(*notification.NotificationUseCase)),
	wire.Bind(new(usecase.BracketUseCase), new(*competition.BracketUseCase)),
	wire.Bind(new(usecase.APITokenUseCase), new(*user.APITokenUseCase)),
	wire.Bind(new(user.SoloTeamCreator), new(*team.TeamUseCase)),
	wire.Bind(new(usecase.RatingUseCase), new(*challenge.RatingUseCase)),
	wire.Bind(new(cache.ChallengeListCacheInvalidator), new(*challenge.ChallengeUseCase)),
	wire.Bind(new(cache.ScoreboardCacheInvalidator), new(*cache.ScoreboardCacheService)),
	wire.Bind(new(usecase.SubmissionBatcher), new(*competition.SubmissionBatcher)),
)

var InfraSet = wire.NewSet(
	ProvideValidator,
	ProvideCrypto,
	ProvideCache,
	ProvideKeyValueStore,
	ProvidePubSubStore,
	ProvideBroadcaster,
	ProvideWsController,
)

var HTTPSet = wire.NewSet(
	ProvideServerDeps,
	ProvideRouter,
	ProvideServer,
	ProvideApp,
)
