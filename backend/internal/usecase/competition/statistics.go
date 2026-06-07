package competition

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const (
	statsGeneralKey              = "stats:general"
	statsChallengesKey           = "stats:challenges"
	statsChallengeDetailFmt      = "stats:challenge:%s"
	statsHistoryFmt              = "stats:history:%d"
	statsHistoryFrozenFmt        = "stats:history:frozen:%d:%d"
	statsGraphFmt                = "stats:graph:%d"
	statsGraphFrozenFmt          = "stats:graph:frozen:%d:%d"
	statsSolvePercentagesKey     = "stats:solve_percentages"
	statsScoreDistributionKey    = "stats:score_distribution"
	statsSubmissionTimeseriesKey = "stats:submission_timeseries"
	statsSubmissionByTypeFmt     = "stats:submission_timeseries:%v"
	statsTeamRegistrationKey     = "stats:team_registration"
	statsUserRegistrationKey     = "stats:user_registration"
	statsSolveMatrixKey          = "stats:solve_matrix"
	statsFunnelFmt               = "stats:funnel:%d"

	statsLongTTL   = 5 * time.Minute
	statsShortTTL  = 30 * time.Second
	statsDetailTTL = 1 * time.Minute
)

type competitionGetter interface {
	Get(ctx context.Context) (*domain.Competition, error)
}

type StatisticsUseCase struct {
	deps StatisticsDeps
	sf   singleflight.Group
}

type StatisticsDeps struct {
	StatsRepo  repo.StatisticsRepository
	Cache      *cachekit.Cache
	CompGetter competitionGetter
	TM         repo.TransactionManager
}

var _ usecase.StatisticsUseCase = (*StatisticsUseCase)(nil)

func NewStatisticsUseCase(deps StatisticsDeps) *StatisticsUseCase {
	return &StatisticsUseCase{deps: deps}
}

// GetGeneralStats returns overall competition statistics (team/user counts,
// solve totals) with freeze-aware caching. When the competition freeze is active
// the cache key encodes the freeze Unix timestamp so the frozen view is stored
// independently from the live view; a change in freeze time automatically
// produces a cache miss. forceLive=true bypasses freeze and always returns
// the live, real-time counts regardless of scoreboard freeze state.
func (uc *StatisticsUseCase) GetGeneralStats(ctx context.Context, forceLive bool) (*domain.GeneralStats, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	return freezeAwareLoad(uc, ctx, statsGeneralKey, statsLongTTL, frozen, freezeTime,
		func(ctx context.Context, ft *time.Time) (*domain.GeneralStats, error) {
			return statsReadOnlyLoad(uc, ctx, "GetGeneralStats", "StatsRepo.GetGeneralStats",
				func(roCtx context.Context) (*domain.GeneralStats, error) {
					return uc.deps.StatsRepo.GetGeneralStats(roCtx, ft)
				},
			)
		},
	)
}

// GetChallengeStats returns per-challenge solve counts using the same
// freeze-aware caching pattern as GetGeneralStats.
func (uc *StatisticsUseCase) GetChallengeStats(ctx context.Context, forceLive bool) ([]*domain.ChallengeStats, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	return freezeAwareLoad(uc, ctx, statsChallengesKey, statsLongTTL, frozen, freezeTime,
		func(ctx context.Context, ft *time.Time) ([]*domain.ChallengeStats, error) {
			return statsReadOnlyLoad(uc, ctx, "GetChallengeStats", "StatsRepo.GetChallengeStats",
				func(roCtx context.Context) ([]*domain.ChallengeStats, error) {
					return uc.deps.StatsRepo.GetChallengeStats(roCtx, ft)
				},
			)
		},
	)
}

// GetChallengeDetailStats returns detailed stats for a single challenge
// (attempt counts, first blood, solve timeline) using the same freeze-aware
// caching pattern as GetGeneralStats. The challengeID string is parsed to
// uuid.UUID inside the cache-miss loader.
func (uc *StatisticsUseCase) GetChallengeDetailStats(ctx context.Context, challengeID string, forceLive bool) (*domain.ChallengeDetailStats, error) {
	id, err := uuid.Parse(challengeID)
	if err != nil {
		return nil, fmt.Errorf("StatisticsUseCase - GetChallengeDetailStats - uuid.Parse: %w", err)
	}

	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	return freezeAwareLoad(uc, ctx, fmt.Sprintf(statsChallengeDetailFmt, challengeID), statsDetailTTL, frozen, freezeTime,
		func(ctx context.Context, ft *time.Time) (*domain.ChallengeDetailStats, error) {
			return statsReadOnlyLoad(uc, ctx, "GetChallengeDetailStats", "StatsRepo.GetChallengeDetailStats",
				func(roCtx context.Context) (*domain.ChallengeDetailStats, error) {
					return uc.deps.StatsRepo.GetChallengeDetailStats(roCtx, id, ft)
				},
			)
		},
	)
}

// GetScoreboardHistory returns the top-N teams' score history with freeze-aware caching.
// When frozen, the cache key encodes both the freeze Unix timestamp and limit so that
// the frozen view is cached independently from the live view and a freeze-time shift
// produces a distinct key. The query runs inside a read-only transaction when TM is available.
func (uc *StatisticsUseCase) GetScoreboardHistory(ctx context.Context, limit int, forceLive bool) ([]*domain.ScoreboardHistoryEntry, error) {
	if limit < 1 {
		limit = usecase.DefaultScoreboardHistoryLimit
	} else if limit > usecase.MaxScoreboardHistoryLimit {
		limit = usecase.MaxScoreboardHistoryLimit
	}

	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	var key string

	if frozen {
		key = fmt.Sprintf(statsHistoryFrozenFmt, freezeTime.Unix(), limit)
	} else {
		key = fmt.Sprintf(statsHistoryFmt, limit)
	}

	var ft *time.Time

	if frozen {
		ft = &freezeTime
	}

	return statsCachedLoad(uc, ctx, key, statsShortTTL, func(ctx context.Context) ([]*domain.ScoreboardHistoryEntry, error) {
		return statsReadOnlyLoad(uc, ctx, "GetScoreboardHistory", "StatsRepo.GetScoreboardHistory",
			func(roCtx context.Context) ([]*domain.ScoreboardHistoryEntry, error) {
				return uc.deps.StatsRepo.GetScoreboardHistory(roCtx, limit, ft)
			},
		)
	})
}

// GetScoreboardGraph returns the scoreboard score-over-time graph for the top
// N teams. It uses freeze-aware caching: when the competition freeze is active
// the cache key encodes both the freeze Unix timestamp and topN, so the frozen
// graph is cached independently from the live graph and a freeze-time shift
// (e.g. after an unpause) produces a different key and bypasses any stale
// entry. The history query runs inside a read-only transaction when TM is
// available. The raw history rows are transformed by buildScoreboardGraph into
// per-team timelines before being stored in the cache and returned.
func (uc *StatisticsUseCase) GetScoreboardGraph(ctx context.Context, topN int, forceLive bool) (*domain.ScoreboardGraph, error) {
	if topN < 1 {
		topN = usecase.DefaultScoreboardHistoryLimit
	} else if topN > usecase.MaxScoreboardHistoryLimit {
		topN = usecase.MaxScoreboardHistoryLimit
	}

	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	var key string

	if frozen {
		key = fmt.Sprintf(statsGraphFrozenFmt, freezeTime.Unix(), topN)
	} else {
		key = fmt.Sprintf(statsGraphFmt, topN)
	}

	var ft *time.Time

	if frozen {
		ft = &freezeTime
	}

	return statsCachedLoad(uc, ctx, key, statsShortTTL, func(ctx context.Context) (*domain.ScoreboardGraph, error) {
		history, err := statsReadOnlyLoad(uc, ctx, "GetScoreboardGraph", "StatsRepo.GetScoreboardHistory",
			func(roCtx context.Context) ([]*domain.ScoreboardHistoryEntry, error) {
				return uc.deps.StatsRepo.GetScoreboardHistory(roCtx, topN, ft)
			},
		)
		if err != nil {
			return nil, err
		}

		return buildScoreboardGraph(history), nil
	})
}

// GetChallengeSolvePercentages returns the solve-rate percentage for every
// challenge using the same freeze-aware caching pattern as GetGeneralStats.
func (uc *StatisticsUseCase) GetChallengeSolvePercentages(ctx context.Context, forceLive bool) ([]*domain.ChallengeSolvePercentage, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	return freezeAwareLoad(uc, ctx, statsSolvePercentagesKey, statsLongTTL, frozen, freezeTime,
		func(ctx context.Context, ft *time.Time) ([]*domain.ChallengeSolvePercentage, error) {
			return statsReadOnlyLoad(uc, ctx, "GetChallengeSolvePercentages", "StatsRepo.GetChallengeSolvePercentages",
				func(roCtx context.Context) ([]*domain.ChallengeSolvePercentage, error) {
					return uc.deps.StatsRepo.GetChallengeSolvePercentages(roCtx, ft)
				},
			)
		},
	)
}

// GetScoreDistribution returns bucketed score distribution across participating
// teams using the same freeze-aware caching pattern as GetGeneralStats.
func (uc *StatisticsUseCase) GetScoreDistribution(ctx context.Context, forceLive bool) ([]*domain.ScoreDistributionBucket, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	return freezeAwareLoad(uc, ctx, statsScoreDistributionKey, statsLongTTL, frozen, freezeTime,
		func(ctx context.Context, ft *time.Time) ([]*domain.ScoreDistributionBucket, error) {
			return statsReadOnlyLoad(uc, ctx, "GetScoreDistribution", "StatsRepo.GetScoreDistribution",
				func(roCtx context.Context) ([]*domain.ScoreDistributionBucket, error) {
					return uc.deps.StatsRepo.GetScoreDistribution(roCtx, ft)
				},
			)
		},
	)
}

// GetSubmissionTimeSeries returns a time-series of all submission counts
// using the same freeze-aware caching pattern as GetGeneralStats.
func (uc *StatisticsUseCase) GetSubmissionTimeSeries(ctx context.Context, forceLive bool) (*domain.SubmissionTimeSeriesStats, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	return freezeAwareLoad(uc, ctx, statsSubmissionTimeseriesKey, statsLongTTL, frozen, freezeTime,
		func(ctx context.Context, ft *time.Time) (*domain.SubmissionTimeSeriesStats, error) {
			return statsReadOnlyLoad(uc, ctx, "GetSubmissionTimeSeries", "StatsRepo.GetSubmissionTimeSeries",
				func(roCtx context.Context) (*domain.SubmissionTimeSeriesStats, error) {
					return uc.deps.StatsRepo.GetSubmissionTimeSeries(roCtx, ft)
				},
			)
		},
	)
}

// GetSubmissionTimeSeriesByType returns a time-series filtered by submission
// correctness (isCorrect=true -> correct flags, false -> incorrect) using the
// same freeze-aware caching pattern as GetGeneralStats.
func (uc *StatisticsUseCase) GetSubmissionTimeSeriesByType(ctx context.Context, isCorrect, forceLive bool) ([]*domain.RegistrationTimePoint, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	return freezeAwareLoad(uc, ctx, fmt.Sprintf(statsSubmissionByTypeFmt, isCorrect), statsLongTTL, frozen, freezeTime,
		func(ctx context.Context, ft *time.Time) ([]*domain.RegistrationTimePoint, error) {
			return statsReadOnlyLoad(uc, ctx, "GetSubmissionTimeSeriesByType", "StatsRepo.GetSubmissionTimeSeriesByType",
				func(roCtx context.Context) ([]*domain.RegistrationTimePoint, error) {
					return uc.deps.StatsRepo.GetSubmissionTimeSeriesByType(roCtx, isCorrect, ft)
				},
			)
		},
	)
}

func (uc *StatisticsUseCase) GetTeamRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error) {
	return statsCachedLoad(uc, ctx, statsTeamRegistrationKey, statsLongTTL, func(ctx context.Context) ([]*domain.RegistrationTimePoint, error) {
		return statsReadOnlyLoad(uc, ctx, "GetTeamRegistrationTimeSeries", "StatsRepo.GetTeamRegistrationTimeSeries",
			func(roCtx context.Context) ([]*domain.RegistrationTimePoint, error) {
				return uc.deps.StatsRepo.GetTeamRegistrationTimeSeries(roCtx)
			},
		)
	})
}

func (uc *StatisticsUseCase) GetUserRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error) {
	return statsCachedLoad(uc, ctx, statsUserRegistrationKey, statsLongTTL, func(ctx context.Context) ([]*domain.RegistrationTimePoint, error) {
		return statsReadOnlyLoad(uc, ctx, "GetUserRegistrationTimeSeries", "StatsRepo.GetUserRegistrationTimeSeries",
			func(roCtx context.Context) ([]*domain.RegistrationTimePoint, error) {
				return uc.deps.StatsRepo.GetUserRegistrationTimeSeries(roCtx)
			},
		)
	})
}

// buildScoreboardGraph transforms a flat list of scoreboard history entries
// into a ScoreboardGraph. It groups entries by team ID and builds a per-team
// timeline of ScorePoints (timestamp + cumulative score). The shared time
// range (earliest and latest timestamp across all entries) is computed in the
// same pass. Teams are sorted alphabetically by name in the result so the
// output is deterministic.
// GetSolveMatrix returns a team×challenge solve matrix using the same
// freeze-aware caching pattern as GetGeneralStats.
func (uc *StatisticsUseCase) GetSolveMatrix(ctx context.Context, forceLive bool) ([]*domain.SolveMatrixRow, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	return freezeAwareLoad(uc, ctx, statsSolveMatrixKey, statsShortTTL, frozen, freezeTime,
		func(ctx context.Context, ft *time.Time) ([]*domain.SolveMatrixRow, error) {
			return statsReadOnlyLoad(uc, ctx, "GetSolveMatrix", "StatsRepo.GetSolveMatrix",
				func(roCtx context.Context) ([]*domain.SolveMatrixRow, error) {
					return uc.deps.StatsRepo.GetSolveMatrix(roCtx, ft)
				},
			)
		},
	)
}

// GetAdminStatisticsFunnel returns admin-only engagement funnel analytics for
// challenge opens, attempts, and solves. Team/user cells are bounded by limit
// while challenge aggregate rows include all visible and locked challenges.
func (uc *StatisticsUseCase) GetAdminStatisticsFunnel(ctx context.Context, limit int, forceLive bool) (*domain.AdminStatisticsFunnel, error) {
	if limit < 1 {
		limit = usecase.DefaultScoreboardHistoryLimit
	} else if limit > usecase.MaxScoreboardHistoryLimit {
		limit = usecase.MaxScoreboardHistoryLimit
	}

	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	return freezeAwareLoad(uc, ctx, fmt.Sprintf(statsFunnelFmt, limit), statsShortTTL, frozen, freezeTime,
		func(ctx context.Context, ft *time.Time) (*domain.AdminStatisticsFunnel, error) {
			return statsReadOnlyLoad(uc, ctx, "GetAdminStatisticsFunnel", "StatsRepo.GetAdminStatisticsFunnel",
				func(roCtx context.Context) (*domain.AdminStatisticsFunnel, error) {
					return uc.deps.StatsRepo.GetAdminStatisticsFunnel(roCtx, limit, ft)
				},
			)
		},
	)
}
