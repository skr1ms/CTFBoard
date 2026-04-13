package computil

import (
	"context"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// CompetitionGetter is satisfied by both repo.CompetitionRepository and usecase.CompetitionUseCase.
type CompetitionGetter interface {
	Get(ctx context.Context) (*domain.Competition, error)
}

// Cached resolves the current competition preferring uc (cached layer) over repo (direct DB).
// Any error is silently swallowed and nil is returned so callers can degrade gracefully.
func Cached(ctx context.Context, uc, repo CompetitionGetter) *domain.Competition {
	if uc != nil {
		comp, err := uc.Get(ctx)
		if err == nil {
			return comp
		}
	}

	if repo != nil {
		comp, err := repo.Get(ctx)
		if err == nil {
			return comp
		}
	}

	return nil
}

// Fresh resolves the current competition preferring repo (direct DB) over uc (cached layer).
// Use this on the submission hot-path where stale freeze-time data must be avoided.
// Any error is silently swallowed and nil is returned.
func Fresh(ctx context.Context, repo, uc CompetitionGetter) *domain.Competition {
	if repo != nil {
		comp, err := repo.Get(ctx)
		if err == nil {
			return comp
		}
	}

	if uc != nil {
		comp, err := uc.Get(ctx)
		if err == nil {
			return comp
		}
	}

	return nil
}
