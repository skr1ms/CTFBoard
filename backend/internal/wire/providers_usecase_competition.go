package wire

import (
	"context"

	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	iws "github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
)

func ProvideCompetitionUseCase(
	competitionRepo repo.CompetitionRepository,
	auditLogRepo repo.AuditLogRepository,
	TM repo.TransactionManager,
	kv cachekit.KeyValueStore,
	statsCache *cachekit.Cache,
	scoreboardCache cacheutil.ScoreboardCacheInvalidator,
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
	challengeListCache cacheutil.ChallengeListCacheInvalidator,
	broadcaster *iws.Broadcaster,
) *competition.SolveUseCase {
	var statsInvalidator competition.StatisticsCacheInvalidator

	if c != nil {
		statsInvalidator = &competition.StatsCacheInvalidatorImpl{Cache: c}
	}

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
		StatsCache:         statsInvalidator,
		ChallengeListCache: challengeListCache,
		Broadcaster:        broadcaster,
	})
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

func ProvideSubmissionUseCase(submissionRepo repo.SubmissionRepository, competitionUC *competition.CompetitionUseCase, TM repo.TransactionManager, challengeUC *challenge.ChallengeUseCase, userRepo repo.UserRepository, teamRepo repo.TeamRepository, c *cachekit.Cache, l logkit.Logger) *competition.SubmissionUseCase {
	var statsInvalidator competition.StatisticsCacheInvalidator

	if c != nil {
		statsInvalidator = &competition.StatsCacheInvalidatorImpl{Cache: c}
	}

	return competition.NewSubmissionUseCase(competition.SubmissionDeps{
		SubmissionRepo:   submissionRepo,
		CompGetter:       competitionUC,
		TM:               TM,
		SolveCreator:     challengeUC,
		SolveDeleter:     challengeUC,
		CacheInvalidator: challengeUC,
		StatsCache:       statsInvalidator,
		Logger:           l,
		UserRepo:         userRepo,
		TeamRepo:         teamRepo,
	})
}

func ProvideBracketUseCase(bracketRepo repo.BracketRepository, tm repo.TransactionManager) *competition.BracketUseCase {
	return competition.NewBracketUseCase(competition.BracketDeps{BracketRepo: bracketRepo, TM: tm})
}

func ProvideShareUseCase(
	cfg *config.Config,
	solveRepo repo.SolveRepository,
	challengeRepo repo.ChallengeRepository,
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	compParamUC *competition.CompetitionParamUseCase,
) *competition.ShareUseCase {
	return competition.NewShareUseCase(competition.ShareDeps{
		SolveRepo:     solveRepo,
		ChallengeRepo: challengeRepo,
		UserRepo:      userRepo,
		TeamRepo:      teamRepo,
		CompParamUC:   compParamUC,
		BaseURL:       cfg.BaseURL,
		FrontendURL:   cfg.FrontendURL,
		ShareSecret:   cfg.ShareSecret,
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
