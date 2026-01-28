package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"golang.org/x/sync/singleflight"
)

type StatisticsUseCase struct {
	statsRepo repo.StatisticsRepository
	redis     *redis.Client
	sf        singleflight.Group
}

func NewStatisticsUseCase(statsRepo repo.StatisticsRepository, redis *redis.Client) *StatisticsUseCase {
	return &StatisticsUseCase{
		statsRepo: statsRepo,
		redis:     redis,
	}
}

// getWithCache tries to get data from Redis, falling back to fetchFn combined with singleflight.
func getWithCache[T any](
	ctx context.Context,
	uc *StatisticsUseCase,
	key string,
	ttl time.Duration,
	fetchFn func() (T, error),
) (T, error) {
	var result T

	// 1. Try cache
	val, err := uc.redis.Get(ctx, key).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(val), &result); err == nil {
			return result, nil
		}
	}

	// 2. Singleflight + DB fetch
	v, err, _ := uc.sf.Do(key, func() (any, error) {
		data, err := fetchFn()
		if err != nil {
			return nil, err
		}

		if bytes, err := json.Marshal(data); err == nil {
			uc.redis.Set(ctx, key, bytes, ttl)
		}
		return data, nil
	})

	if err != nil {
		var zero T
		return zero, err
	}

	return v.(T), nil
}

func (uc *StatisticsUseCase) GetGeneralStats(ctx context.Context) (*entity.GeneralStats, error) {
	return getWithCache(ctx, uc, "stats:general", 5*time.Minute, func() (*entity.GeneralStats, error) {
		stats, err := uc.statsRepo.GetGeneralStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetGeneralStats: %w", err)
		}
		return stats, nil
	})
}

func (uc *StatisticsUseCase) GetChallengeStats(ctx context.Context) ([]*entity.ChallengeStats, error) {
	return getWithCache(ctx, uc, "stats:challenges", 5*time.Minute, func() ([]*entity.ChallengeStats, error) {
		stats, err := uc.statsRepo.GetChallengeStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetChallengeStats: %w", err)
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

	return getWithCache(ctx, uc, key, 30*time.Second, func() ([]*entity.ScoreboardHistoryEntry, error) {
		history, err := uc.statsRepo.GetScoreboardHistory(ctx, limit)
		if err != nil {
			return nil, fmt.Errorf("StatisticsUseCase - GetScoreboardHistory: %w", err)
		}
		return history, nil
	})
}
