package competition

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

// invalidateLocal clears local cache, negative cache, and forgets singleflight key; concurrent ensureLoaded may start a new loadAll (accepted: stale window bounded by localTTL).
func (uc *CompetitionParamUseCase) invalidateLocal() {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	uc.lastLoad = time.Time{}
	uc.negativeCache = make(map[string]time.Time)
	uc.sf.Forget(loadAllKey)
}

// invalidate defers cache invalidation until the outer transaction commits.
func (uc *CompetitionParamUseCase) invalidate(ctx context.Context) {
	txctx.AfterCommitOrNow(ctx, uc.invalidateNow)
}

// invalidateNow clears local cache, deletes Redis cache, and publishes invalidation message; concurrent Get/GetAll use stale local cache.
func (uc *CompetitionParamUseCase) invalidateNow(ctx context.Context) {
	uc.invalidateLocal()

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), invalidateTimeout)
	defer cancel()

	if uc.deps.Cache != nil {
		if err := uc.deps.Cache.Del(ctx, configsCacheKey); err != nil {
			uc.deps.Logger.WithError(err).Warn("competition_params: cache invalidation failed", logkit.Fields{"key": configsCacheKey})
		}
	}

	if uc.deps.PubSub != nil {
		if err := uc.deps.PubSub.Publish(ctx, configsInvChannel, "1"); err != nil {
			uc.deps.Logger.WithError(err).Warn("competition_params: pubsub invalidation broadcast failed", logkit.Fields{"channel": configsInvChannel})
		}
	}
}

// loadFromRedis attempts to populate the local in-memory cache from the Redis
// JSON blob stored under configsCacheKey. Returns an error when Redis is unavailable
// or the key is missing/stale, causing ensureLoaded to fall through to loadAll.
func (uc *CompetitionParamUseCase) loadFromRedis(ctx context.Context) error {
	if uc.deps.Cache == nil {
		return errCacheNotInitialized
	}

	raw, err := uc.deps.Cache.Get(ctx, configsCacheKey)
	if err != nil {
		return fmt.Errorf("CompetitionParamUseCase - loadFromRedis - Cache.Get: %w", err)
	}

	var params []*domain.CompetitionParam
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return fmt.Errorf("CompetitionParamUseCase - loadFromRedis - json.Unmarshal: %w", err)
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	uc.cache = make(map[string]*domain.CompetitionParam, len(params))
	for _, p := range params {
		uc.cache[p.Key] = p
	}

	uc.lastLoad = time.Now()

	return nil
}

// loadAll fetches all competition params from the database, rebuilds the
// in-memory map, and writes the result to Redis for other instances to share.
func (uc *CompetitionParamUseCase) loadAll(ctx context.Context) error {
	params, err := uc.deps.Repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("CompetitionParamUseCase - loadAll - CompetitionParamRepo.GetAll: %w", err)
	}

	uc.mu.Lock()

	uc.cache = make(map[string]*domain.CompetitionParam, len(params))
	for _, p := range params {
		uc.cache[p.Key] = p
	}

	uc.lastLoad = time.Now()
	uc.mu.Unlock()

	if uc.deps.Cache != nil {
		if b, err := json.Marshal(params); err == nil {
			if setErr := uc.deps.Cache.Set(ctx, configsCacheKey, b, redisTTL); setErr != nil {
				uc.deps.Logger.WithError(setErr).Warn("competition_params: redis cache set failed", logkit.Fields{"key": configsCacheKey})
			}
		}
	}

	return nil
}

// ensureLoaded deduplicates concurrent cache misses with singleflight.
// Order of fallback: in-memory TTL check -> Redis -> database (loadAll).
func (uc *CompetitionParamUseCase) ensureLoaded(ctx context.Context) error {
	_, err, _ := uc.sf.Do(loadAllKey, func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		uc.mu.RLock()
		cacheValid := time.Since(uc.lastLoad) < localTTL
		uc.mu.RUnlock()

		if cacheValid {
			return nil, nil
		}

		if uc.deps.Cache != nil {
			if err := uc.loadFromRedis(loadCtx); err == nil {
				return nil, nil
			}
		}

		if err := uc.loadAll(loadCtx); err != nil {
			return nil, fmt.Errorf("CompetitionParamUseCase - ensureLoaded - loadAll: %w", err)
		}

		return nil, nil
	})

	return err
}
