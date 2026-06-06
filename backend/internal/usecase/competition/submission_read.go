package competition

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func (uc *SubmissionUseCase) GetByChallenge(ctx context.Context, challengeID uuid.UUID, page, perPage int, forceLive bool) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	ft := uc.freezeTimeOrNil(ctx, forceLive)

	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetByChallenge(ctx, challengeID, ft, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountByChallenge(ctx, challengeID, ft)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetByChallenge: %w", err)
	}

	return result, nil
}

func (uc *SubmissionUseCase) GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int, forceLive bool) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	ft := uc.freezeTimeOrNil(ctx, forceLive)

	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetByUser(ctx, userID, ft, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountByUser(ctx, userID, ft)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetByUser: %w", err)
	}

	return result, nil
}

func (uc *SubmissionUseCase) GetByTeam(ctx context.Context, teamID uuid.UUID, page, perPage int, forceLive bool) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	ft := uc.freezeTimeOrNil(ctx, forceLive)

	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetByTeam(ctx, teamID, ft, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountByTeam(ctx, teamID, ft)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetByTeam: %w", err)
	}

	return result, nil
}

func (uc *SubmissionUseCase) GetAll(ctx context.Context, page, perPage int, forceLive bool) (*usecase.Paginated[*domain.SubmissionWithDetails], error) {
	ft := uc.freezeTimeOrNil(ctx, forceLive)

	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
			return uc.deps.SubmissionRepo.GetAll(ctx, ft, limit, offset)
		},
		func(ctx context.Context) (int64, error) {
			return uc.deps.SubmissionRepo.CountAll(ctx, ft)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetAll: %w", err)
	}

	return result, nil
}

func (uc *SubmissionUseCase) GetStats(ctx context.Context, challengeID uuid.UUID, forceLive bool) (*domain.SubmissionStats, error) {
	ft := uc.freezeTimeOrNil(ctx, forceLive)

	stats, err := uc.deps.SubmissionRepo.GetStats(ctx, challengeID, ft)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetStats - SubmissionRepo.GetStats: %w", err)
	}

	return stats, nil
}

func (uc *SubmissionUseCase) freezeTimeOrNil(ctx context.Context, forceLive bool) *time.Time {
	if forceLive || uc.deps.CompGetter == nil {
		return nil
	}

	comp, err := uc.deps.CompGetter.Get(ctx)
	if err != nil || comp == nil || !comp.IsFreezeActive() {
		return nil
	}

	return comp.FreezeTime
}

func (uc *SubmissionUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.SubmissionWithDetails, error) {
	sub, err := uc.deps.SubmissionRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - GetByID - SubmissionRepo.GetByID: %w", err)
	}

	return sub, nil
}
