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

// isFrozen checks whether the competition scoreboard freeze is currently active.
// Returns (false, zero) when CompGetter is nil, when the competition cannot be
// loaded, or when IsFreezeActive returns false.
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

// freezeAwareLoad is a generic helper that handles the repeated pattern across
// StatisticsUseCase methods: resolve freeze state, build a freeze-encoded cache
// key, and call cachekit.GetOrLoad passing *time.Time directly to the repo
// (nil = live data, non-nil = frozen at that timestamp).
func freezeAwareLoad[T any](
	uc *StatisticsUseCase,
	ctx context.Context,
	baseKey string,
	ttl time.Duration,
	frozen bool,
	freezeTime time.Time,
	fn func(context.Context, *time.Time) (T, error),
) (T, error) {
	key := baseKey

	var ft *time.Time

	if frozen {
		key = baseKey + statsFrozenSuffix(freezeTime)
		ft = &freezeTime
	}

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, ttl, func(ctx context.Context) (T, error) {
		return fn(ctx, ft)
	})
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
			return uc.deps.StatsRepo.GetGeneralStats(ctx, ft)
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
			return uc.deps.StatsRepo.GetChallengeStats(ctx, ft)
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
			return uc.deps.StatsRepo.GetChallengeDetailStats(ctx, id, ft)
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

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsShortTTL, func(ctx context.Context) ([]*domain.ScoreboardHistoryEntry, error) {
		var history []*domain.ScoreboardHistoryEntry

		if uc.deps.TM != nil {
			err := uc.deps.TM.ReadOnly(ctx, func(roCtx context.Context) error {
				var err error

				history, err = uc.deps.StatsRepo.GetScoreboardHistory(roCtx, limit, ft)

				return err
			})
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardHistory - TM.ReadOnly: %w", err)
			}
		} else {
			var err error

			history, err = uc.deps.StatsRepo.GetScoreboardHistory(ctx, limit, ft)
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardHistory - StatsRepo.GetScoreboardHistory: %w", err)
			}
		}

		return history, nil
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

	return cachekit.GetOrLoad(uc.deps.Cache, ctx, key, statsShortTTL, func(ctx context.Context) (*domain.ScoreboardGraph, error) {
		var history []*domain.ScoreboardHistoryEntry

		if uc.deps.TM != nil {
			err := uc.deps.TM.ReadOnly(ctx, func(roCtx context.Context) error {
				var err error

				history, err = uc.deps.StatsRepo.GetScoreboardHistory(roCtx, topN, ft)

				return err
			})
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardGraph - TM.ReadOnly: %w", err)
			}
		} else {
			var err error

			history, err = uc.deps.StatsRepo.GetScoreboardHistory(ctx, topN, ft)
			if err != nil {
				return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardGraph - StatsRepo.GetScoreboardHistory: %w", err)
			}
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
			return uc.deps.StatsRepo.GetChallengeSolvePercentages(ctx, ft)
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
			return uc.deps.StatsRepo.GetScoreDistribution(ctx, ft)
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
			return uc.deps.StatsRepo.GetSubmissionTimeSeries(ctx, ft)
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
			return uc.deps.StatsRepo.GetSubmissionTimeSeriesByType(ctx, isCorrect, ft)
		},
	)
}

func (uc *StatisticsUseCase) GetTeamRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error) {
	return cachekit.GetOrLoad(uc.deps.Cache, ctx, statsTeamRegistrationKey, statsLongTTL, func(context.Context) ([]*domain.RegistrationTimePoint, error) {
		data, err := uc.deps.StatsRepo.GetTeamRegistrationTimeSeries(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetTeamRegistrationTimeSeries - StatsRepo.GetTeamRegistrationTimeSeries: %w", err)
		}

		return data, nil
	})
}

func (uc *StatisticsUseCase) GetUserRegistrationTimeSeries(ctx context.Context) ([]*domain.RegistrationTimePoint, error) {
	return cachekit.GetOrLoad(uc.deps.Cache, ctx, statsUserRegistrationKey, statsLongTTL, func(context.Context) ([]*domain.RegistrationTimePoint, error) {
		data, err := uc.deps.StatsRepo.GetUserRegistrationTimeSeries(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetUserRegistrationTimeSeries - StatsRepo.GetUserRegistrationTimeSeries: %w", err)
		}

		return data, nil
	})
}

// buildScoreboardGraph transforms a flat list of scoreboard history entries
// into a ScoreboardGraph. It groups entries by team ID and builds a per-team
// timeline of ScorePoints (timestamp + cumulative score). The shared time
// range (earliest and latest timestamp across all entries) is computed in the
// same pass. Teams are sorted alphabetically by name in the result so the
// output is deterministic.
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

// GetSolveMatrix returns a team×challenge solve matrix using the same
// freeze-aware caching pattern as GetGeneralStats.
func (uc *StatisticsUseCase) GetSolveMatrix(ctx context.Context, forceLive bool) ([]*domain.SolveMatrixRow, error) {
	frozen, freezeTime := uc.isFrozen(ctx)

	if forceLive {
		frozen = false
	}

	return freezeAwareLoad(uc, ctx, statsSolveMatrixKey, statsShortTTL, frozen, freezeTime,
		func(ctx context.Context, ft *time.Time) ([]*domain.SolveMatrixRow, error) {
			return uc.deps.StatsRepo.GetSolveMatrix(ctx, ft)
		},
	)
}
