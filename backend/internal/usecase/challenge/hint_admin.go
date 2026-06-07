package challenge

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func (uc *HintUseCase) Create(ctx context.Context, challengeID uuid.UUID, title, content string, cost, orderIndex int) (*domain.Hint, error) {
	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
		return nil, fmt.Errorf("HintUseCase - Create - ChallengeRepo.GetByID: %w", err)
	}

	if cost < 0 {
		return nil, apperr.NewValidationErrorf("hint cost must be non-negative")
	}

	hint := &domain.Hint{
		ChallengeID: challengeID,
		Title:       title,
		Content:     content,
		Cost:        cost,
		OrderIndex:  orderIndex,
	}

	err := uc.deps.HintRepo.Create(ctx, hint)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - Create - HintRepo.Create: %w", err)
	}

	return hint, nil
}

func (uc *HintUseCase) Update(ctx context.Context, ID uuid.UUID, title, content string, cost, orderIndex int) (*domain.Hint, error) {
	if cost < 0 {
		return nil, apperr.NewValidationErrorf("hint cost must be non-negative")
	}

	var hint *domain.Hint

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err error

		hint, err = uc.deps.HintRepo.GetByIDForUpdate(ctx, ID)
		if err != nil {
			return fmt.Errorf("HintUseCase - Update - HintRepo.GetByIDForUpdate: %w", err)
		}

		hint.Title = title
		hint.Content = content
		hint.Cost = cost

		hint.OrderIndex = orderIndex
		if err := uc.deps.HintRepo.Update(ctx, hint); err != nil {
			return fmt.Errorf("HintUseCase - Update - HintRepo.Update: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - Update - TM.Run: %w", err)
	}

	return hint, nil
}

func (uc *HintUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	err := uc.deps.HintRepo.Delete(ctx, ID)
	if err != nil {
		return fmt.Errorf("HintUseCase - Delete - HintRepo.Delete: %w", err)
	}

	return nil
}

func (uc *HintUseCase) GetAllUnlocks(ctx context.Context, page, perPage int) (*usecase.Paginated[*domain.UnlockWithDetails], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.UnlockWithDetails, error) {
			unlocks, err := uc.deps.HintRepo.GetAll(ctx, limit, offset)
			if err != nil {
				return nil, err
			}

			out := make([]*domain.UnlockWithDetails, 0, len(unlocks))
			for _, unlock := range unlocks {
				out = append(out, hintUnlockToUnlock(unlock))
			}

			return out, nil
		},
		func(ctx context.Context) (int64, error) {
			n, err := uc.deps.HintRepo.CountAll(ctx)

			return int64(n), err
		},
	)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - GetAllUnlocks: %w", err)
	}

	return result, nil
}

func hintUnlockToUnlock(unlock *domain.HintUnlockWithDetails) *domain.UnlockWithDetails {
	return &domain.UnlockWithDetails{
		ID:          unlock.ID,
		Type:        domain.UnlockTypeHint,
		ResourceID:  unlock.HintID,
		HintID:      unlock.HintID,
		TeamID:      unlock.TeamID,
		UnlockedAt:  unlock.UnlockedAt,
		ChallengeID: unlock.ChallengeID,
		HintCost:    unlock.HintCost,
	}
}
