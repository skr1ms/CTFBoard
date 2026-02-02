//go:build wireinject

package wire

import (
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/config"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/storage"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/mailer"
	pkgWS "github.com/skr1ms/CTFBoard/pkg/websocket"
)

var RepoSet = wire.NewSet(
	ProvideUserRepo,
	ProvideChallengeRepo,
	ProvideSolveRepo,
	ProvideTeamRepo,
	ProvideCompetitionRepo,
	ProvideHintRepo,
	ProvideHintUnlockRepo,
	ProvideAwardRepo,
	ProvideAuditLogRepo,
	ProvideStatisticsRepo,
	ProvideFileRepo,
	ProvideTxRepo,
	ProvideBackupRepo,
	ProvideAppSettingsRepo,
	ProvideVerificationTokenRepo,
	wire.Bind(new(repo.UserRepository), new(*persistent.UserRepo)),
	wire.Bind(new(repo.TeamRepository), new(*persistent.TeamRepo)),
	wire.Bind(new(repo.SolveRepository), new(*persistent.SolveRepo)),
	wire.Bind(new(repo.CompetitionRepository), new(*persistent.CompetitionRepo)),
	wire.Bind(new(repo.ChallengeRepository), new(*persistent.ChallengeRepo)),
	wire.Bind(new(repo.HintRepository), new(*persistent.HintRepo)),
	wire.Bind(new(repo.HintUnlockRepository), new(*persistent.HintUnlockRepo)),
	wire.Bind(new(repo.AwardRepository), new(*persistent.AwardRepo)),
	wire.Bind(new(repo.AuditLogRepository), new(*persistent.AuditLogRepo)),
	wire.Bind(new(repo.StatisticsRepository), new(*persistent.StatisticsRepository)),
	wire.Bind(new(repo.FileRepository), new(*persistent.FileRepository)),
	wire.Bind(new(repo.TxRepository), new(*persistent.TxRepo)),
	wire.Bind(new(repo.BackupRepository), new(*persistent.BackupRepo)),
	wire.Bind(new(repo.AppSettingsRepository), new(*persistent.AppSettingsRepo)),
	wire.Bind(new(repo.VerificationTokenRepository), new(*persistent.VerificationTokenRepo)),
)

var UseCaseSet = wire.NewSet(
	ProvideUserUseCase,
	ProvideTeamUseCase,
	ProvideAwardUseCase,
	ProvideChallengeUseCase,
	ProvideHintUseCase,
	ProvideCompetitionUseCase,
	ProvideSolveUseCase,
	ProvideStatisticsUseCase,
	ProvideFileUseCase,
	ProvideBackupUseCase,
	ProvideSettingsUseCase,
	ProvideEmailUseCase,
	wire.Bind(new(usecase.BackupUseCase), new(*competition.BackupUseCase)),
)

var InfraSet = wire.NewSet(
	ProvideValidator,
	ProvideCrypto,
	ProvideWsController,
	ProvideRouter,
	ProvideServer,
	ProvideApp,
)

func InitializeApp(
	cfg *config.Config,
	l logger.Logger,
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	storageProvider storage.Provider,
	jwtService *jwt.JWTService,
	wsHub *pkgWS.Hub,
	mailer mailer.Mailer,
) (*App, error) {
	wire.Build(RepoSet, UseCaseSet, InfraSet)
	return nil, nil
}
