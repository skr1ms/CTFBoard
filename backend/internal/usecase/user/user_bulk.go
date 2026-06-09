package user

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const (
	maxBulkUserActionIDs = 100
	minBulkLockTargets   = 2
)

func (uc *UserUseCase) BanUsers(ctx context.Context, userIDs []uuid.UUID, reason string, actorID uuid.UUID) (*usecase.BulkActionResult, error) {
	ids, err := normalizeBulkUserIDs(userIDs)
	if err != nil {
		return nil, err
	}

	if err := rejectBulkSelfAction(ids, actorID); err != nil {
		return nil, err
	}

	var (
		aggregate userBanTxResult
		changed   int
	)

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.lockBulkUserModerationTargets(ctx, ids); err != nil {
			return err
		}

		for _, id := range ids {
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

	if err := rejectBulkSelfAction(ids, actorID); err != nil {
		return nil, err
	}

	var (
		aggregate userBanRestoreTxResult
		changed   int
	)

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.lockBulkUserModerationTargets(ctx, ids); err != nil {
			return err
		}

		for _, id := range ids {
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

func rejectBulkSelfAction(ids []uuid.UUID, actorID uuid.UUID) error {
	if slices.Contains(ids, actorID) {
		return apperr.ErrAccessDenied
	}

	return nil
}

func (uc *UserUseCase) lockBulkUserModerationTargets(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) < minBulkLockTargets {
		return nil
	}

	teamIDs := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if err := uc.deps.UserRepo.Lock(ctx, id); err != nil {
			return fmt.Errorf("UserUseCase - lockBulkUserModerationTargets - UserRepo.Lock: %w", err)
		}

		user, err := uc.deps.UserRepo.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf("UserUseCase - lockBulkUserModerationTargets - UserRepo.GetByID: %w", err)
		}

		if user.TeamID != nil {
			teamIDs = append(teamIDs, *user.TeamID)
		}
	}

	if uc.deps.TeamRepo == nil {
		return nil
	}

	teamIDs = domain.UniqueUUIDs(teamIDs)
	domain.SortUUIDs(teamIDs)

	for _, teamID := range teamIDs {
		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("UserUseCase - lockBulkUserModerationTargets - TeamRepo.Lock: %w", err)
		}
	}

	return nil
}
