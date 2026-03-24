package competition

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const (
	MaxScoreboardHistoryLimit = 100

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

	statsLongTTL   = 5 * time.Minute
	statsShortTTL  = 30 * time.Second
	statsDetailTTL = 1 * time.Minute

	defaultScoreboardHistoryLimit = 10
)

type competitionGetter interface {
	Get(ctx context.Context) (*domain.Competition, error)
}

type StatisticsUseCase struct {
	deps StatisticsDeps
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

func (uc *StatisticsUseCase) isFrozen(ctx context.Context) (bool, time.Time) {
	if uc.deps.CompGetter == nil {
		return false, time.Time{}
	}
	comp, err := uc.deps.CompGetter.Get(ctx)
	if err != nil || comp == nil {
		return false, time.Time{}
	}
	if comp.IsFreezeActive() {
		return true, *comp.FreezeTime
	}
	return false, time.Time{}
}

func statsFrozenSuffix(freezeTime time.Time) string {
	return ":frozen:" + strconv.FormatInt(freezeTime.Unix(), 10)
}

func (uc *StatisticsUseCase) GetGeneralStats(ctx context.Context, forceLive bool) (*domain.GeneralStats, error) {
	frozen, freezeTime := uc.isFrozen(ctx)
	if forceLive {
		frozen = false
	}

	var key string
	if frozen {
		key = statsGeneralKey + statsFrozenSuffix(freezeTime)
	} else {
		key = statsGeneralKey
	}

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsLongTTL, func(context.Context) (*domain.GeneralStats, error) {
		if frozen {
			stats, err := uc.deps.StatsRepo.GetGeneralStatsFrozen(ctx, freezeTime)
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetGeneralStats - StatsRepo.GetGeneralStatsFrozen: %w", err)
			}
			return stats, nil
		}
		stats, err := uc.deps.StatsRepo.GetGeneralStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetGeneralStats - StatisticsRepo.GetGeneralStats: %w", err)
		}
		return stats, nil
	})
}

func (uc *StatisticsUseCase) GetChallengeStats(ctx context.Context, forceLive bool) ([]*domain.ChallengeStats, error) {
	frozen, freezeTime := uc.isFrozen(ctx)
	if forceLive {
		frozen = false
	}

	var key string
	if frozen {
		key = statsChallengesKey + statsFrozenSuffix(freezeTime)
	} else {
		key = statsChallengesKey
	}

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsLongTTL, func(context.Context) ([]*domain.ChallengeStats, error) {
		if frozen {
			stats, err := uc.deps.StatsRepo.GetChallengeStatsFrozen(ctx, freezeTime)
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetChallengeStats - StatsRepo.GetChallengeStatsFrozen: %w", err)
			}
			return stats, nil
		}
		stats, err := uc.deps.StatsRepo.GetChallengeStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeStats - StatisticsRepo.GetChallengeStats: %w", err)
		}
		return stats, nil
	})
}

func (uc *StatisticsUseCase) GetChallengeDetailStats(ctx context.Context, challengeID string, forceLive bool) (*domain.ChallengeDetailStats, error) {
	frozen, freezeTime := uc.isFrozen(ctx)
	if forceLive {
		frozen = false
	}

	var key string
	if frozen {
		key = fmt.Sprintf(statsChallengeDetailFmt, challengeID) + statsFrozenSuffix(freezeTime)
	} else {
		key = fmt.Sprintf(statsChallengeDetailFmt, challengeID)
	}

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsDetailTTL, func(context.Context) (*domain.ChallengeDetailStats, error) {
		id, err := uuid.Parse(challengeID)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeDetailStats - uuid.Parse: %w", err)
		}
		if frozen {
			stats, err := uc.deps.StatsRepo.GetChallengeDetailStatsFrozen(ctx, id, freezeTime)
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetChallengeDetailStats - StatsRepo.GetChallengeDetailStatsFrozen: %w", err)
			}
			return stats, nil
		}
		stats, err := uc.deps.StatsRepo.GetChallengeDetailStats(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeDetailStats - StatisticsRepo.GetChallengeDetailStats: %w", err)
		}
		return stats, nil
	})
}

func (uc *StatisticsUseCase) GetScoreboardHistory(ctx context.Context, limit int, forceLive bool) ([]*domain.ScoreboardHistoryEntry, error) {
	if limit < 1 {
		limit = defaultScoreboardHistoryLimit
	} else if limit > MaxScoreboardHistoryLimit {
		limit = MaxScoreboardHistoryLimit
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

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsShortTTL, func(context.Context) ([]*domain.ScoreboardHistoryEntry, error) {
		var history []*domain.ScoreboardHistoryEntry
		if uc.deps.TM != nil {
			if err := uc.deps.TM.ReadOnly(ctx, func(roCtx context.Context) error {
				var err error
				if frozen {
					history, err = uc.deps.StatsRepo.GetScoreboardHistoryFrozen(roCtx, freezeTime, limit)
				} else {
					history, err = uc.deps.StatsRepo.GetScoreboardHistory(roCtx, limit)
				}
				return err
			}); err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardHistory - TM.ReadOnly: %w", err)
			}
		} else {
			var err error
			if frozen {
				history, err = uc.deps.StatsRepo.GetScoreboardHistoryFrozen(ctx, freezeTime, limit)
			} else {
				history, err = uc.deps.StatsRepo.GetScoreboardHistory(ctx, limit)
			}
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardHistory - StatisticsRepo.GetScoreboardHistory: %w", err)
			}
		}
		return history, nil
	})
}

func (uc *StatisticsUseCase) GetScoreboardGraph(ctx context.Context, topN int, forceLive bool) (*domain.ScoreboardGraph, error) {
	if topN < 1 {
		topN = defaultScoreboardHistoryLimit
	} else if topN > MaxScoreboardHistoryLimit {
		topN = MaxScoreboardHistoryLimit
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

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsShortTTL, func(context.Context) (*domain.ScoreboardGraph, error) {
		var history []*domain.ScoreboardHistoryEntry
		if uc.deps.TM != nil {
			if err := uc.deps.TM.ReadOnly(ctx, func(roCtx context.Context) error {
				var err error
				if frozen {
					history, err = uc.deps.StatsRepo.GetScoreboardHistoryFrozen(roCtx, freezeTime, topN)
				} else {
					history, err = uc.deps.StatsRepo.GetScoreboardHistory(roCtx, topN)
				}
				return err
			}); err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardGraph - TM.ReadOnly: %w", err)
			}
		} else {
			var err error
			if frozen {
				history, err = uc.deps.StatsRepo.GetScoreboardHistoryFrozen(ctx, freezeTime, topN)
			} else {
				history, err = uc.deps.StatsRepo.GetScoreboardHistory(ctx, topN)
			}
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardGraph - StatisticsRepo.GetScoreboardHistory: %w", err)
			}
		}
		return buildScoreboardGraph(history), nil
	})
}

func (uc *StatisticsUseCase) GetChallengeSolvePercentages(ctx context.Context, forceLive bool) ([]*domain.ChallengeSolvePercentage, error) {
	frozen, freezeTime := uc.isFrozen(ctx)
	if forceLive {
		frozen = false
	}

	var key string
	if frozen {
		key = statsSolvePercentagesKey + statsFrozenSuffix(freezeTime)
	} else {
		key = statsSolvePercentagesKey
	}

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsLongTTL, func(context.Context) ([]*domain.ChallengeSolvePercentage, error) {
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

func (uc *StatisticsUseCase) GetScoreDistribution(ctx context.Context, forceLive bool) ([]*domain.ScoreDistributionBucket, error) {
	frozen, freezeTime := uc.isFrozen(ctx)
	if forceLive {
		frozen = false
	}

	var key string
	if frozen {
		key = statsScoreDistributionKey + statsFrozenSuffix(freezeTime)
	} else {
		key = statsScoreDistributionKey
	}

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsLongTTL, func(context.Context) ([]*domain.ScoreDistributionBucket, error) {
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

func (uc *StatisticsUseCase) GetSubmissionTimeSeries(ctx context.Context, forceLive bool) (*domain.SubmissionTimeSeriesStats, error) {
	frozen, freezeTime := uc.isFrozen(ctx)
	if forceLive {
		frozen = false
	}

	var key string
	if frozen {
		key = statsSubmissionTimeseriesKey + statsFrozenSuffix(freezeTime)
	} else {
		key = statsSubmissionTimeseriesKey
	}

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsLongTTL, func(context.Context) (*domain.SubmissionTimeSeriesStats, error) {
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

func (uc *StatisticsUseCase) GetSubmissionTimeSeriesByType(ctx context.Context, isCorrect, forceLive bool) ([]*domain.RegistrationTimePoint, error) {
	frozen, freezeTime := uc.isFrozen(ctx)
	if forceLive {
		frozen = false
	}

	var cacheKey string
	if frozen {
		cacheKey = fmt.Sprintf(statsSubmissionByTypeFmt, isCorrect) + statsFrozenSuffix(freezeTime)
	} else {
		cacheKey = fmt.Sprintf(statsSubmissionByTypeFmt, isCorrect)
	}

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, cacheKey, statsLongTTL, func(context.Context) ([]*domain.RegistrationTimePoint, error) {
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

func (uc *StatisticsUseCase) GetTeamRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error) {
	return cachekit.GetOrLoad(uc.deps.Cache, ctx, statsTeamRegistrationKey, statsLongTTL, func(context.Context) ([]*domain.RegistrationTimePoint, error) {
		data, err := uc.deps.StatsRepo.GetTeamRegistrationTimeSeries(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetTeamRegistrationTimeSeries - StatisticsRepo.GetTeamRegistrationTimeSeries: %w", err)
		}
		return data, nil
	})
}

func (uc *StatisticsUseCase) GetUserRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error) {
	return cachekit.GetOrLoad(uc.deps.Cache, ctx, statsUserRegistrationKey, statsLongTTL, func(context.Context) ([]*domain.RegistrationTimePoint, error) {
		data, err := uc.deps.StatsRepo.GetUserRegistrationTimeSeries(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetUserRegistrationTimeSeries - StatisticsRepo.GetUserRegistrationTimeSeries: %w", err)
		}
		return data, nil
	})
}

func buildScoreboardGraph(history []*domain.ScoreboardHistoryEntry) *domain.ScoreboardGraph {
	if len(history) == 0 {
		return &domain.ScoreboardGraph{
			Range: domain.TimeRange{},
			Teams: []domain.TeamTimeline{},
		}
	}

	teamMap := make(map[string]*domain.TeamTimeline)
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
			tl = &domain.TeamTimeline{
				TeamID:   h.TeamID,
				TeamName: h.TeamName,
				Timeline: []domain.ScorePoint{},
			}
			teamMap[teamIDStr] = tl
		}

		tl.Timeline = append(tl.Timeline, domain.ScorePoint{
			Timestamp: h.Timestamp,
			Score:     h.Points,
		})
	}

	teams := make([]domain.TeamTimeline, 0, len(teamMap))
	for _, tl := range teamMap {
		teams = append(teams, *tl)
	}
	slices.SortFunc(teams, func(a, b domain.TeamTimeline) int {
		return strings.Compare(a.TeamName, b.TeamName)
	})

	return &domain.ScoreboardGraph{
		Range: domain.TimeRange{
			Start: minTime,
			End:   maxTime,
		},
		Teams: teams,
	}
}

func (uc *StatisticsUseCase) GetSolveMatrix(ctx context.Context, forceLive bool) ([]*domain.SolveMatrixRow, error) {
	frozen, freezeTime := uc.isFrozen(ctx)
	if forceLive {
		frozen = false
	}
	if frozen {
		key := statsSolveMatrixKey + statsFrozenSuffix(freezeTime)
		return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsShortTTL, func(context.Context) ([]*domain.SolveMatrixRow, error) {
			matrix, err := uc.deps.StatsRepo.GetSolveMatrixFrozen(ctx, freezeTime)
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetSolveMatrix - GetSolveMatrixFrozen: %w", err)
			}
			return matrix, nil
		})
	}
	return cachekit.GetOrLoad(uc.deps.Cache, ctx, statsSolveMatrixKey, statsShortTTL, func(context.Context) ([]*domain.SolveMatrixRow, error) {
		matrix, err := uc.deps.StatsRepo.GetSolveMatrix(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetSolveMatrix - StatisticsRepo.GetSolveMatrix: %w", err)
		}
		return matrix, nil
	})
}
