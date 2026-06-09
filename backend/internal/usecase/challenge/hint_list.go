package challenge

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// GetByChallengeID returns hints for a challenge with per-hint unlock status for the given team.
// Enforces prerequisite checks: hints are hidden until requirements are met.
// Hint content is masked (empty) for locked hints that the team has not unlocked.
func (uc *HintUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*usecase.HintWithUnlockStatus, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - GetByChallengeID - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, err
	}

	reqs, err := uc.deps.ChallengeRepo.GetRequirementsForEnforcement(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - GetByChallengeID - GetRequirementsForEnforcement: %w", err)
	}

	if len(reqs) > 0 {
		if teamID == nil || uc.deps.SolveRepo == nil {
			return nil, apperr.ErrChallengeNotFound
		}

		met, err := requirementsSatisfied(ctx, reqs, *teamID, uc.deps.SolveRepo)
		if err != nil {
			return nil, fmt.Errorf("HintUseCase - GetByChallengeID - requirementsSatisfied: %w", err)
		}

		if !met {
			return nil, apperr.ErrChallengeNotFound
		}
	}

	hints, err := uc.deps.HintRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - GetByChallengeID - HintRepo.GetByChallengeID: %w", err)
	}

	unlockedMap := make(map[uuid.UUID]bool)

	if teamID != nil {
		unlockedIDs, err := uc.deps.HintRepo.GetUnlockedHintIDs(ctx, *teamID, challengeID)
		if err != nil {
			return nil, fmt.Errorf("HintUseCase - GetByChallengeID - HintRepo.GetUnlockedHintIDs: %w", err)
		}

		for _, ID := range unlockedIDs {
			unlockedMap[ID] = true
		}
	}

	result := make([]*usecase.HintWithUnlockStatus, 0, len(hints))
	for _, hint := range hints {
		unlocked := unlockedMap[hint.ID]

		h := &usecase.HintWithUnlockStatus{
			Hint:     hint,
			Unlocked: unlocked,
		}
		if !unlocked {
			h.Hint = &domain.Hint{
				ID:          hint.ID,
				ChallengeID: hint.ChallengeID,
				Title:       hint.Title,
				Cost:        hint.Cost,
				OrderIndex:  hint.OrderIndex,
			}
		}

		result = append(result, h)
	}

	return result, nil
}
