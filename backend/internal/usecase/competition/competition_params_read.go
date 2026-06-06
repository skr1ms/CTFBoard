package competition

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

// Get returns a competition parameter by key using a multi-layer cache
// in-memory map (guarded by a read mutex, valid for localTTL) -> Redis -> database
// Concurrent cache misses are deduplicated with singleflight so only one
// database round-trip is made per key per miss window. When the database
// returns not-found, the registered config registry is checked; if the key has
// a default there, a synthetic param is returned and stored in the local cache
// Keys that are absent from both the database and the registry are stored in a
// negative cache for negativeCacheTTL to prevent repeated database queries.
func (uc *CompetitionParamUseCase) Get(ctx context.Context, key string) (*domain.CompetitionParam, error) {
	key = strings.TrimSpace(key)
	if err := validateCompetitionParamKey(key); err != nil {
		return nil, err
	}

	uc.mu.RLock()

	cacheValid := time.Since(uc.lastLoad) < localTTL
	if cacheValid {
		if p, ok := uc.cache[key]; ok {
			uc.mu.RUnlock()

			return p, nil
		}
	}

	uc.mu.RUnlock()

	if !cacheValid {
		if err := uc.ensureLoaded(ctx); err != nil {
			return nil, fmt.Errorf("CompetitionParamUseCase - Get - ensureLoaded: %w", err)
		}
	}

	uc.mu.RLock()

	p, ok := uc.cache[key]
	if !ok {
		if exp, ok := uc.negativeCache[key]; ok && time.Now().Before(exp) {
			uc.mu.RUnlock()

			return nil, apperr.ErrCompetitionParamNotFound
		}
	}

	uc.mu.RUnlock()

	if ok {
		return p, nil
	}

	sfKey := "competition_params:key:" + key

	v, err, _ := uc.sf.Do(sfKey, func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		c, err := uc.deps.Repo.GetByKey(loadCtx, key)
		if err != nil {
			if errors.Is(err, apperr.ErrCompetitionParamNotFound) {
				if def, ok := domain.GetConfigDef(key); ok {
					p := paramFromDef(def)

					uc.mu.Lock()
					uc.cache[key] = p
					uc.mu.Unlock()

					return p, nil
				}

				uc.mu.Lock()
				uc.negativeCache[key] = time.Now().Add(negativeCacheTTL)
				uc.mu.Unlock()
			}

			return nil, fmt.Errorf("CompetitionParamUseCase - Get - CompetitionParamRepo.GetByKey: %w", err)
		}

		uc.mu.Lock()
		uc.cache[key] = c
		uc.mu.Unlock()

		return c, nil
	})
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamUseCase - Get - singleflight: %w", err)
	}

	c, ok := v.(*domain.CompetitionParam)
	if !ok {
		return nil, fmt.Errorf("CompetitionParamUseCase - Get: unexpected cache type for key %q", key)
	}

	return c, nil
}

func paramFromDef(def domain.ConfigDef) *domain.CompetitionParam {
	return &domain.CompetitionParam{
		Key: def.Key, Value: def.DefaultValue, ValueType: def.ValueType,
		Category: def.Category, Description: def.Description,
	}
}

// GetAll returns all competition params merged with config registry defaults.
// Registry defaults are overridden by any database-persisted values. The result
// is sorted by key for deterministic output.
func (uc *CompetitionParamUseCase) GetAll(ctx context.Context) ([]*domain.CompetitionParam, error) {
	uc.mu.RLock()
	cacheValid := time.Since(uc.lastLoad) < localTTL
	uc.mu.RUnlock()

	if !cacheValid {
		if err := uc.ensureLoaded(ctx); err != nil {
			return nil, fmt.Errorf("CompetitionParamUseCase - GetAll - ensureLoaded: %w", err)
		}
	}

	uc.mu.RLock()
	defer uc.mu.RUnlock()

	merged := make(map[string]*domain.CompetitionParam, domain.ConfigRegistryCount()+len(uc.cache))

	domain.RangeConfigRegistry(func(k string, def domain.ConfigDef) bool {
		merged[k] = paramFromDef(def)

		return true
	})

	maps.Copy(merged, uc.cache)

	list := make([]*domain.CompetitionParam, 0, len(merged))
	for _, p := range merged {
		list = append(list, p)
	}

	slices.SortFunc(list, func(a, b *domain.CompetitionParam) int { return strings.Compare(a.Key, b.Key) })

	return list, nil
}

// GetByCategory returns all competition parameters for a specific category,
// merging config registry defaults with any database-persisted values.
// Registry defaults for the category are loaded first; persisted values
// then override them so callers always see the effective configuration.
// The result is sorted by key for deterministic output.
func (uc *CompetitionParamUseCase) GetByCategory(ctx context.Context, category string) ([]*domain.CompetitionParam, error) {
	if err := validateCategory(category); err != nil {
		return nil, err
	}

	merged := make(map[string]*domain.CompetitionParam)

	domain.RangeConfigRegistry(func(k string, def domain.ConfigDef) bool {
		if def.Category == category {
			merged[k] = paramFromDef(def)
		}

		return true
	})

	dbParams, err := uc.deps.Repo.GetByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamUseCase - GetByCategory - Repo.GetByCategory: %w", err)
	}

	for _, p := range dbParams {
		merged[p.Key] = p
	}

	out := make([]*domain.CompetitionParam, 0, len(merged))
	for _, p := range merged {
		out = append(out, p)
	}

	slices.SortFunc(out, func(a, b *domain.CompetitionParam) int { return strings.Compare(a.Key, b.Key) })

	return out, nil
}
