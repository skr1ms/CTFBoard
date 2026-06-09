package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const maxBulkUserActionIDs = 100

func (uc *UserUseCase) BanUsers(ctx context.Context, userIDs []uuid.UUID, reason string, actorID uuid.UUID) (*usecase.BulkActionResult, error) {
	ids, err := normalizeBulkUserIDs(userIDs)
	if err != nil {
		return nil, err
	}

	var (
		aggregate userBanTxResult
		changed   int
	)

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if id == actorID {
				return apperr.ErrAccessDenied
			}

			result, err := uc.banUserTx(ctx, id, reason)
			if err != nil {
				return err
			}

			if result.changed {
				changed++
			}

			aggregate.scoreboardInvalidateTeamIDs = append(aggregate.scoreboardInvalidateTeamIDs, result.scoreboardInvalidateTeamIDs...)
			aggregate.captainIDsToNotify = append(aggregate.captainIDsToNotify, result.captainIDsToNotify...)
			aggregate.changed = aggregate.changed || result.changed
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("UserUseCase - BanUsers - TM.Run: %w", err)
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) { uc.afterUserBanCommit(ctx, ids, aggregate) })

	return &usecase.BulkActionResult{AffectedCount: changed}, nil
}

func (uc *UserUseCase) UnbanUsers(ctx context.Context, userIDs []uuid.UUID, actorID uuid.UUID) (*usecase.BulkActionResult, error) {
	ids, err := normalizeBulkUserIDs(userIDs)
	if err != nil {
		return nil, err
	}

	var (
		aggregate userBanRestoreTxResult
		changed   int
	)

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if id == actorID {
				return apperr.ErrAccessDenied
			}

			result, err := uc.restoreUserBanTx(ctx, id, true)
			if err != nil {
				return err
			}

			if result.changed {
				changed++
			}

			aggregate.scoreboardInvalidateTeamIDs = append(aggregate.scoreboardInvalidateTeamIDs, result.scoreboardInvalidateTeamIDs...)
			aggregate.changed = aggregate.changed || result.changed
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("UserUseCase - UnbanUsers - TM.Run: %w", err)
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		uc.invalidateUserBanRestore(ctx, ids, aggregate.scoreboardInvalidateTeamIDs)
	})

	return &usecase.BulkActionResult{AffectedCount: changed}, nil
}

func normalizeBulkUserIDs(ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, apperr.NewValidationErrorf("ids must contain at least one user")
	}

	if len(ids) > maxBulkUserActionIDs {
		return nil, apperr.NewValidationErrorf("ids cannot contain more than %d users", maxBulkUserActionIDs)
	}

	normalized := domain.UniqueUUIDs(ids)
	domain.SortUUIDs(normalized)

	return normalized, nil
}
