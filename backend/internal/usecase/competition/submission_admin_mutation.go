package competition

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// Update changes the correctness flag of a submission inside a transaction. It
// re-reads the row with FOR UPDATE to prevent concurrent admin edits from
// racing, then creates a solve when flipping from incorrect to correct or
// removes the existing solve when flipping from correct to incorrect.
func (uc *SubmissionUseCase) Update(ctx context.Context, ID uuid.UUID, isCorrect bool) (*domain.SubmissionWithDetails, error) {
	if err := uc.requireTxDeps("Update", true); err != nil {
		return nil, err
	}

	var locked *domain.Submission

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		// Re-read with FOR UPDATE inside the transaction to avoid TOCTOU on concurrent admin edits.
		var err error

		locked, err = uc.deps.SubmissionRepo.GetByIDForUpdate(ctx, ID)
		if err != nil {
			return fmt.Errorf("SubmissionUseCase - Update - SubmissionRepo.GetByIDForUpdate: %w", err)
		}

		if err := uc.deps.SubmissionRepo.Update(ctx, ID, isCorrect); err != nil {
			return fmt.Errorf("SubmissionUseCase - Update - SubmissionRepo.Update: %w", err)
		}

		switch {
		case locked.TeamID != nil && !locked.IsCorrect && isCorrect:
			if err := uc.deps.SolveCreator.AdminCreateSolve(ctx, locked.UserID, *locked.TeamID, locked.ChallengeID, true); err != nil {
				return fmt.Errorf("SubmissionUseCase - Update - SolveCreator.AdminCreateSolve: %w", err)
			}
		case locked.TeamID != nil && locked.IsCorrect && !isCorrect:
			if err := uc.deps.SolveDeleter.AdminDeleteSolve(ctx, *locked.TeamID, locked.ChallengeID); err != nil {
				return fmt.Errorf("SubmissionUseCase - Update - SolveDeleter.AdminDeleteSolve: %w", err)
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - Update - TM.Run: %w", err)
	}

	if uc.deps.CacheInvalidator != nil {
		if locked != nil && locked.TeamID != nil {
			uc.deps.CacheInvalidator.InvalidateScoreboardCacheForTeam(ctx, *locked.TeamID)
		} else {
			uc.deps.CacheInvalidator.InvalidateScoreboardCache(ctx)
		}
	}

	uc.invalidateStatisticsCache(ctx, "Update")

	sub, err := uc.deps.SubmissionRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - Update - SubmissionRepo.GetByID: %w", err)
	}

	return sub, nil
}

// Discard marks a submission as discarded. If the submission was correct it
// also removes the associated solve inside the same transaction.
func (uc *SubmissionUseCase) Discard(ctx context.Context, ID uuid.UUID) (*domain.SubmissionWithDetails, error) {
	if err := uc.requireTxDeps("Discard", false); err != nil {
		return nil, err
	}

	var teamIDToInvalidate *uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		locked, err := uc.deps.SubmissionRepo.GetByIDForUpdate(ctx, ID)
		if err != nil {
			return fmt.Errorf("SubmissionUseCase - Discard - SubmissionRepo.GetByIDForUpdate: %w", err)
		}

		if locked.IsCorrect && locked.TeamID != nil {
			if err := uc.deps.SolveDeleter.AdminDeleteSolve(ctx, *locked.TeamID, locked.ChallengeID); err != nil {
				return fmt.Errorf("SubmissionUseCase - Discard - SolveDeleter.AdminDeleteSolve: %w", err)
			}

			teamIDToInvalidate = locked.TeamID
		}

		if err := uc.deps.SubmissionRepo.Discard(ctx, ID); err != nil {
			return fmt.Errorf("SubmissionUseCase - Discard - SubmissionRepo.Discard: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - Discard - TM.Run: %w", err)
	}

	if uc.deps.CacheInvalidator != nil && teamIDToInvalidate != nil {
		uc.deps.CacheInvalidator.InvalidateScoreboardCacheForTeam(ctx, *teamIDToInvalidate)
	}

	uc.invalidateStatisticsCache(ctx, "Discard")

	result, err := uc.deps.SubmissionRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("SubmissionUseCase - Discard - SubmissionRepo.GetByID: %w", err)
	}

	return result, nil
}

func (uc *SubmissionUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := uc.requireTxDeps("Delete", false); err != nil {
		return err
	}

	var teamIDToInvalidate *uuid.UUID

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		locked, err := uc.deps.SubmissionRepo.GetByIDForUpdate(ctx, ID)
		if err != nil {
			if errors.Is(err, apperr.ErrSubmissionNotFound) {
				return nil
			}

			return fmt.Errorf("SubmissionUseCase - Delete - SubmissionRepo.GetByIDForUpdate: %w", err)
		}

		if locked.IsCorrect && locked.TeamID != nil {
			if err := uc.deps.SolveDeleter.AdminDeleteSolve(ctx, *locked.TeamID, locked.ChallengeID); err != nil {
				return fmt.Errorf("SubmissionUseCase - Delete - SolveDeleter.AdminDeleteSolve: %w", err)
			}

			teamIDToInvalidate = locked.TeamID
		}

		if err := uc.deps.SubmissionRepo.Delete(ctx, ID); err != nil {
			return fmt.Errorf("SubmissionUseCase - Delete - SubmissionRepo.Delete: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("SubmissionUseCase - Delete - TM.Run: %w", err)
	}

	if uc.deps.CacheInvalidator != nil && teamIDToInvalidate != nil {
		uc.deps.CacheInvalidator.InvalidateScoreboardCacheForTeam(ctx, *teamIDToInvalidate)
	}

	uc.invalidateStatisticsCache(ctx, "Delete")

	return nil
}
