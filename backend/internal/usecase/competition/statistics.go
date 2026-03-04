package competition

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/google/uuid"
)

const (
	MaxScoreboardHistoryLimit = 100

	statsGeneralKey              = "stats:general"
	statsChallengesKey           = "stats:challenges"
	statsChallengeDetailFmt      = "stats:challenge:%s"
	statsHistoryFmt              = "stats:history:%d"
	statsHistoryFrozenFmt        = "stats:history:frozen:%d"
	statsGraphFmt                = "stats:graph:%d"
	statsGraphFrozenFmt          = "stats:graph:frozen:%d"
	statsSolvePercentagesKey     = "stats:solve_percentages"
	statsScoreDistributionKey    = "stats:score_distribution"
	statsSubmissionTimeseriesKey = "stats:submission_timeseries"
	statsSubmissionByTypeFmt     = "stats:submission_timeseries:%v"
	statsTeamRegistrationKey     = "stats:team_registration"
	statsUserRegistrationKey     = "stats:user_registration"
	statsSolveMatrixKey          = "stats:solve_matrix"

	statsLongTTL   = 5 * time.Minute
	statsShortTTL  = 30 * time.Second
	statsDetailTTL = 1 * time.Minute

	defaultScoreboardHistoryLimit = 10
)

type competitionGetter interface {
	Get(ctx context.Context) (*entity.Competition, error)
}

type StatisticsUseCase struct {
	deps StatisticsDeps
}

type StatisticsDeps struct {
	StatsRepo  repo.StatisticsRepository
	Cache      *cache.Cache
	CompGetter competitionGetter
}

var _ usecase.StatisticsUseCase = (*StatisticsUseCase)(nil)

func NewStatisticsUseCase(deps StatisticsDeps) *StatisticsUseCase {
	return &StatisticsUseCase{deps: deps}
}

func (uc *StatisticsUseCase) isFrozen(ctx context.Context) (bool, time.Time) {
	if uc.deps.CompGetter == nil {
		return false, time.Time{}
	}
	comp, err := uc.deps.CompGetter.Get(ctx)
	if err != nil || comp == nil {
		return false, time.Time{}
	}
	if comp.GetStatus() == entity.CompetitionStatusFrozen {
		return true, *comp.FreezeTime
	}
	return false, time.Time{}
}

func (uc *StatisticsUseCase) GetGeneralStats(ctx context.Context) (*entity.GeneralStats, error) {
	return cache.GetOrLoad(uc.deps.Cache, ctx, statsGeneralKey, statsLongTTL, func() (*entity.GeneralStats, error) {
		stats, err := uc.deps.StatsRepo.GetGeneralStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetGeneralStats - StatisticsRepo.GetGeneralStats: %w", err)
		}
		return stats, nil
	})
}

func (uc *StatisticsUseCase) GetChallengeStats(ctx context.Context) ([]*entity.ChallengeStats, error) {
	return cache.GetOrLoad(uc.deps.Cache, ctx, statsChallengesKey, statsLongTTL, func() ([]*entity.ChallengeStats, error) {
		stats, err := uc.deps.StatsRepo.GetChallengeStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeStats - StatisticsRepo.GetChallengeStats: %w", err)
		}
		return stats, nil
	})
}

func (uc *StatisticsUseCase) GetChallengeDetailStats(ctx context.Context, challengeID string) (*entity.ChallengeDetailStats, error) {
	key := fmt.Sprintf(statsChallengeDetailFmt, challengeID)
	return cache.GetOrLoad(uc.deps.Cache, ctx, key, statsDetailTTL, func() (*entity.ChallengeDetailStats, error) {
		id, err := uuid.Parse(challengeID)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeDetailStats - uuid.Parse: %w", err)
		}
		stats, err := uc.deps.StatsRepo.GetChallengeDetailStats(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeDetailStats - StatisticsRepo.GetChallengeDetailStats: %w", err)
		}
		return stats, nil
	})
}

func (uc *StatisticsUseCase) GetScoreboardHistory(ctx context.Context, limit int) ([]*entity.ScoreboardHistoryEntry, error) {
	if limit < 1 {
		limit = defaultScoreboardHistoryLimit
	} else if limit > MaxScoreboardHistoryLimit {
		limit = MaxScoreboardHistoryLimit
	}

	frozen, freezeTime := uc.isFrozen(ctx)

	var key string
	if frozen {
		key = fmt.Sprintf(statsHistoryFrozenFmt, limit)
	} else {
		key = fmt.Sprintf(statsHistoryFmt, limit)
	}

	return cache.GetOrLoad(uc.deps.Cache, ctx, key, statsShortTTL, func() ([]*entity.ScoreboardHistoryEntry, error) {
		var (
			history []*entity.ScoreboardHistoryEntry
			err     error
		)
		if frozen {
			history, err = uc.deps.StatsRepo.GetScoreboardHistoryFrozen(ctx, freezeTime, limit)
		} else {
			history, err = uc.deps.StatsRepo.GetScoreboardHistory(ctx, limit)
		}
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardHistory - StatisticsRepo.GetScoreboardHistory: %w", err)
		}
		return history, nil
	})
}

func (uc *StatisticsUseCase) GetScoreboardGraph(ctx context.Context, topN int) (*entity.ScoreboardGraph, error) {
	if topN < 1 {
		topN = defaultScoreboardHistoryLimit
	} else if topN > MaxScoreboardHistoryLimit {
		topN = MaxScoreboardHistoryLimit
	}

	frozen, freezeTime := uc.isFrozen(ctx)

	var key string
	if frozen {
		key = fmt.Sprintf(statsGraphFrozenFmt, topN)
	} else {
		key = fmt.Sprintf(statsGraphFmt, topN)
	}

	return cache.GetOrLoad(uc.deps.Cache, ctx, key, statsShortTTL, func() (*entity.ScoreboardGraph, error) {
		var (
			history []*entity.ScoreboardHistoryEntry
			err     error
		)
		if frozen {
			history, err = uc.deps.StatsRepo.GetScoreboardHistoryFrozen(ctx, freezeTime, topN)
		} else {
			history, err = uc.deps.StatsRepo.GetScoreboardHistory(ctx, topN)
		}
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardGraph - StatisticsRepo.GetScoreboardHistory: %w", err)
		}
		return buildScoreboardGraph(history), nil
	})
}

func (uc *StatisticsUseCase) GetChallengeSolvePercentages(ctx context.Context) ([]*entity.ChallengeSolvePercentage, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	var key string
	if frozen {
		key = statsSolvePercentagesKey + ":frozen"
	} else {
		key = statsSolvePercentagesKey
	}

	return cache.GetOrLoad(uc.deps.Cache, ctx, key, statsLongTTL, func() ([]*entity.ChallengeSolvePercentage, error) {
		if frozen {
			data, err := uc.deps.StatsRepo.GetChallengeSolvePercentagesFrozen(ctx, freezeTime)
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetChallengeSolvePercentages - StatsRepo.GetChallengeSolvePercentagesFrozen: %w", err)
			}
			return data, nil
		}
		data, err := uc.deps.StatsRepo.GetChallengeSolvePercentages(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeSolvePercentages - StatisticsRepo.GetChallengeSolvePercentages: %w", err)
		}
		return data, nil
	})
}

func (uc *StatisticsUseCase) GetScoreDistribution(ctx context.Context) ([]*entity.ScoreDistributionBucket, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	var key string
	if frozen {
		key = statsScoreDistributionKey + ":frozen"
	} else {
		key = statsScoreDistributionKey
	}

	return cache.GetOrLoad(uc.deps.Cache, ctx, key, statsLongTTL, func() ([]*entity.ScoreDistributionBucket, error) {
		if frozen {
			data, err := uc.deps.StatsRepo.GetScoreDistributionFrozen(ctx, freezeTime)
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetScoreDistribution - StatsRepo.GetScoreDistributionFrozen: %w", err)
			}
			return data, nil
		}
		data, err := uc.deps.StatsRepo.GetScoreDistribution(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetScoreDistribution - StatisticsRepo.GetScoreDistribution: %w", err)
		}
		return data, nil
	})
}

func (uc *StatisticsUseCase) GetSubmissionTimeSeries(ctx context.Context) (*entity.SubmissionTimeSeriesStats, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	var key string
	if frozen {
		key = statsSubmissionTimeseriesKey + ":frozen"
	} else {
		key = statsSubmissionTimeseriesKey
	}

	return cache.GetOrLoad(uc.deps.Cache, ctx, key, statsLongTTL, func() (*entity.SubmissionTimeSeriesStats, error) {
		if frozen {
			data, err := uc.deps.StatsRepo.GetSubmissionTimeSeriesFrozen(ctx, freezeTime)
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetSubmissionTimeSeries - StatsRepo.GetSubmissionTimeSeriesFrozen: %w", err)
			}
			return data, nil
		}
		data, err := uc.deps.StatsRepo.GetSubmissionTimeSeries(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetSubmissionTimeSeries - StatisticsRepo.GetSubmissionTimeSeries: %w", err)
		}
		return data, nil
	})
}

func (uc *StatisticsUseCase) GetSubmissionTimeSeriesByType(ctx context.Context, isCorrect bool) ([]*entity.RegistrationTimePoint, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	var cacheKey string
	if frozen {
		cacheKey = fmt.Sprintf(statsSubmissionByTypeFmt+":frozen", isCorrect)
	} else {
		cacheKey = fmt.Sprintf(statsSubmissionByTypeFmt, isCorrect)
	}

	return cache.GetOrLoad(uc.deps.Cache, ctx, cacheKey, statsLongTTL, func() ([]*entity.RegistrationTimePoint, error) {
		if frozen {
			data, err := uc.deps.StatsRepo.GetSubmissionTimeSeriesByTypeFrozen(ctx, isCorrect, freezeTime)
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetSubmissionTimeSeriesByType - StatsRepo.GetSubmissionTimeSeriesByTypeFrozen: %w", err)
			}
			return data, nil
		}
		data, err := uc.deps.StatsRepo.GetSubmissionTimeSeriesByType(ctx, isCorrect)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetSubmissionTimeSeriesByType - StatisticsRepo.GetSubmissionTimeSeriesByType: %w", err)
		}
		return data, nil
	})
}

func (uc *StatisticsUseCase) GetTeamRegistrationTimeSeries(ctx context.Context) ([]*entity.RegistrationTimePoint, error) {
	return cache.GetOrLoad(uc.deps.Cache, ctx, statsTeamRegistrationKey, statsLongTTL, func() ([]*entity.RegistrationTimePoint, error) {
		data, err := uc.deps.StatsRepo.GetTeamRegistrationTimeSeries(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetTeamRegistrationTimeSeries - StatisticsRepo.GetTeamRegistrationTimeSeries: %w", err)
		}
		return data, nil
	})
}

func (uc *StatisticsUseCase) GetUserRegistrationTimeSeries(ctx context.Context) ([]*entity.RegistrationTimePoint, error) {
	return cache.GetOrLoad(uc.deps.Cache, ctx, statsUserRegistrationKey, statsLongTTL, func() ([]*entity.RegistrationTimePoint, error) {
		data, err := uc.deps.StatsRepo.GetUserRegistrationTimeSeries(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetUserRegistrationTimeSeries - StatisticsRepo.GetUserRegistrationTimeSeries: %w", err)
		}
		return data, nil
	})
}

func buildScoreboardGraph(history []*entity.ScoreboardHistoryEntry) *entity.ScoreboardGraph {
	if len(history) == 0 {
		return &entity.ScoreboardGraph{
			Range: entity.TimeRange{},
			Teams: []entity.TeamTimeline{},
		}
	}

	teamMap := make(map[string]*entity.TeamTimeline)
	var minTime, maxTime time.Time

	for i, h := range history {
		if i == 0 {
			minTime = h.Timestamp
			maxTime = h.Timestamp
		} else {
			if h.Timestamp.Before(minTime) {
				minTime = h.Timestamp
			}
			if h.Timestamp.After(maxTime) {
				maxTime = h.Timestamp
			}
		}

		teamIDStr := h.TeamID.String()
		tl, exists := teamMap[teamIDStr]
		if !exists {
			tl = &entity.TeamTimeline{
				TeamID:   h.TeamID,
				TeamName: h.TeamName,
				Timeline: []entity.ScorePoint{},
			}
			teamMap[teamIDStr] = tl
		}

		tl.Timeline = append(tl.Timeline, entity.ScorePoint{
			Timestamp: h.Timestamp,
			Score:     h.Points,
		})
	}

	teams := make([]entity.TeamTimeline, 0, len(teamMap))
	for _, tl := range teamMap {
		teams = append(teams, *tl)
	}
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].TeamName < teams[j].TeamName
	})

	return &entity.ScoreboardGraph{
		Range: entity.TimeRange{
			Start: minTime,
			End:   maxTime,
		},
		Teams: teams,
	}
}

func (uc *StatisticsUseCase) GetSolveMatrix(ctx context.Context) ([]*entity.SolveMatrixRow, error) {
	return cache.GetOrLoad(uc.deps.Cache, ctx, statsSolveMatrixKey, statsShortTTL, func() ([]*entity.SolveMatrixRow, error) {
		matrix, err := uc.deps.StatsRepo.GetSolveMatrix(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetSolveMatrix - StatisticsRepo.GetSolveMatrix: %w", err)
		}
		return matrix, nil
	})
}
