package competition

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/cache"
)

type StatisticsUseCase struct {
	statsRepo repo.StatisticsRepository
	cache     *cache.Cache
}

func NewStatisticsUseCase(
	statsRepo repo.StatisticsRepository,
	cache *cache.Cache,
) *StatisticsUseCase {
	return &StatisticsUseCase{
		statsRepo: statsRepo,
		cache:     cache,
	}
}

func (uc *StatisticsUseCase) GetGeneralStats(ctx context.Context) (*entity.GeneralStats, error) {
	return cache.GetOrLoad(uc.cache, ctx, "stats:general", 5*time.Minute, func() (*entity.GeneralStats, error) {
		stats, err := uc.statsRepo.GetGeneralStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetGeneralStats: %w", err)
		}
		return stats, nil
	})
}

func (uc *StatisticsUseCase) GetChallengeStats(ctx context.Context) ([]*entity.ChallengeStats, error) {
	return cache.GetOrLoad(uc.cache, ctx, "stats:challenges", 5*time.Minute, func() ([]*entity.ChallengeStats, error) {
		stats, err := uc.statsRepo.GetChallengeStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeStats: %w", err)
		}
		return stats, nil
	})
}

func (uc *StatisticsUseCase) GetChallengeDetailStats(ctx context.Context, challengeID string) (*entity.ChallengeDetailStats, error) {
	key := fmt.Sprintf("stats:challenge:%s", challengeID)
	return cache.GetOrLoad(uc.cache, ctx, key, 1*time.Minute, func() (*entity.ChallengeDetailStats, error) {
		id, err := uuid.Parse(challengeID)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeDetailStats: invalid id: %w", err)
		}
		stats, err := uc.statsRepo.GetChallengeDetailStats(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeDetailStats: %w", err)
		}
		return stats, nil
	})
}

func (uc *StatisticsUseCase) GetScoreboardHistory(ctx context.Context, limit int) ([]*entity.ScoreboardHistoryEntry, error) {
	if limit < 1 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}

	key := fmt.Sprintf("stats:history:%d", limit)

	return cache.GetOrLoad(uc.cache, ctx, key, 30*time.Second, func() ([]*entity.ScoreboardHistoryEntry, error) {
		history, err := uc.statsRepo.GetScoreboardHistory(ctx, limit)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardHistory: %w", err)
		}
		return history, nil
	})
}

func (uc *StatisticsUseCase) GetScoreboardGraph(ctx context.Context, topN int) (*entity.ScoreboardGraph, error) {
	if topN < 1 {
		topN = 10
	} else if topN > 50 {
		topN = 50
	}

	key := fmt.Sprintf("stats:graph:%d", topN)

	return cache.GetOrLoad(uc.cache, ctx, key, 30*time.Second, func() (*entity.ScoreboardGraph, error) {
		history, err := uc.statsRepo.GetScoreboardHistory(ctx, topN)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardGraph: %w", err)
		}

		return buildScoreboardGraph(history), nil
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

	return &entity.ScoreboardGraph{
		Range: entity.TimeRange{
			Start: minTime,
			End:   maxTime,
		},
		Teams: teams,
	}
}
