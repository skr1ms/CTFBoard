package competition

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

// InvalidateLocalCache clears the in-process competition cache so that the next call to Get
// re-fetches from Redis/DB. Use in tests after directly mutating the competition row in the DB.
func (uc *CompetitionUseCase) InvalidateLocalCache() {
	uc.localComp.Store(nil)
	uc.localCompAt.Store(0)
}

// competitionCacheStale reports whether the cached competition data should be
// considered stale because the current time has crossed one of its critical
// temporal boundaries (start, end, or freeze time). A boundary is considered
// crossed when now is after it but still within boundaryInvalidateWindow, so
// the cache is invalidated exactly once per transition rather than on every
// request after the boundary passes.
func competitionCacheStale(comp *domain.Competition, now time.Time) bool {
	if comp == nil {
		return false
	}

	if comp.StartTime != nil && now.After(*comp.StartTime) && now.Sub(*comp.StartTime) < boundaryInvalidateWindow {
		return true
	}

	if comp.EndTime != nil && now.After(*comp.EndTime) && now.Sub(*comp.EndTime) < boundaryInvalidateWindow {
		return true
	}

	if comp.FreezeTime != nil && now.After(*comp.FreezeTime) && now.Sub(*comp.FreezeTime) < boundaryInvalidateWindow {
		return true
	}

	return false
}

// Get returns the current competition using a three-layer cache: atomic local
// store -> Redis -> database. It deduplicates concurrent database loads via
// singleflight. If the cached value has crossed a temporal boundary (start,
// end, or freeze time), both the local store and the Redis entry are
// invalidated before the next load, and the statistics cache is also flushed
// so downstream callers reflect the updated competition state immediately.
func (uc *CompetitionUseCase) Get(ctx context.Context) (*domain.Competition, error) {
	now := time.Now()

	if cached := uc.localComp.Load(); cached != nil {
		age := time.Duration(now.UnixNano() - uc.localCompAt.Load())
		if age < localCompTTL && !competitionCacheStale(cached, now) {
			return cached, nil
		}

		if competitionCacheStale(cached, now) {
			uc.localComp.Store(nil)
			uc.localCompAt.Store(0)

			if uc.deps.Redis != nil {
				err := uc.deps.Redis.Del(ctx, cacheutil.KeyCompetition)
				if err != nil {
					uc.deps.Logger.WithError(err).Warn("CompetitionUseCase - Get - Redis.Del")
				}
			}

			if uc.deps.StatsCacheInvalidator != nil {
				err := uc.deps.StatsCacheInvalidator.InvalidateStatistics(ctx)
				if err != nil {
					uc.deps.Logger.WithError(err).Warn("CompetitionUseCase - Get - InvalidateStatistics")
				}
			}
		}
	}

	if uc.deps.Redis != nil {
		val, err := uc.deps.Redis.Get(ctx, cacheutil.KeyCompetition)
		if err == nil {
			var comp domain.Competition

			err := json.Unmarshal([]byte(val), &comp)
			if err == nil {
				if !competitionCacheStale(&comp, time.Now()) {
					uc.storeLocal(&comp)

					return &comp, nil
				}

				err := uc.deps.Redis.Del(ctx, cacheutil.KeyCompetition)
				if err != nil {
					uc.deps.Logger.WithError(err).Warn("CompetitionUseCase - Get - Redis.Del stale")
				}

				if uc.deps.StatsCacheInvalidator != nil {
					err := uc.deps.StatsCacheInvalidator.InvalidateStatistics(ctx)
					if err != nil {
						uc.deps.Logger.WithError(err).Warn("CompetitionUseCase - Get - InvalidateStatistics stale")
					}
				}
			}
		}
	}

	v, err, _ := uc.sf.Do(cacheutil.KeyCompetition, func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		comp, err := uc.deps.CompetitionRepo.Get(loadCtx)
		if err != nil {
			return nil, fmt.Errorf("CompetitionUseCase - Get - CompetitionRepo.Get: %w", err)
		}

		if uc.deps.Redis != nil {
			if bytes, err := json.Marshal(comp); err == nil {
				_ = uc.deps.Redis.Set(loadCtx, cacheutil.KeyCompetition, bytes, redisCacheTTL)
			}
		}

		return comp, nil
	})
	if err != nil {
		return nil, err
	}

	comp, ok := v.(*domain.Competition)
	if !ok {
		return nil, fmt.Errorf("CompetitionUseCase - Get: unexpected type")
	}

	uc.storeLocal(comp)

	return comp, nil
}

func (uc *CompetitionUseCase) storeLocal(comp *domain.Competition) {
	c := *comp
	uc.localComp.Store(&c)
	uc.localCompAt.Store(time.Now().UnixNano())
}
