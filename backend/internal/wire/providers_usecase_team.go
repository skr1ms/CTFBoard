package wire

import (
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team"
)

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
	challengeListCache cacheutil.ChallengeListCacheInvalidator,
	userCacheSvc *cache.UserCacheService,
	sharedCache *cachekit.Cache,
	hintRepo repo.HintRepository,
	ratingRepo repo.RatingRepository,
	fieldValidator *settings.FieldValidator,
	fieldRepo repo.FieldRepository,
	fieldValueRepo repo.FieldValueRepository,
	jwtService *jwtkit.JWTService,
	l logkit.Logger,
) *team.TeamUseCase {
	var statsInvalidator cacheutil.StatisticsCacheInvalidator

	if sharedCache != nil {
		statsInvalidator = &competition.StatsCacheInvalidatorImpl{Cache: sharedCache}
	}

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
		StatsCache:         statsInvalidator,
		ChallengeListCache: challengeListCache,
		UserCache:          userCacheSvc,
		TeamCache:          sharedCache,
		HintRepo:           hintRepo,
		RatingRepo:         ratingRepo,
		FieldValidator:     fieldValidator,
		FieldRepo:          fieldRepo,
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
	statsCache *cachekit.Cache,
	compRepo repo.CompetitionRepository,
) *team.AwardUseCase {
	var statsInvalidator cacheutil.StatisticsCacheInvalidator

	if statsCache != nil {
		statsInvalidator = &competition.StatsCacheInvalidatorImpl{Cache: statsCache}
	}

	return team.NewAwardUseCase(team.AwardDeps{
		AwardRepo:       awardRepo,
		TeamRepo:        teamRepo,
		TM:              TM,
		ScoreboardCache: scoreboardCache,
		StatsCache:      statsInvalidator,
		CompRepo:        compRepo,
	})
}
