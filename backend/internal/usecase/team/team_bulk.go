package team

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
)

const maxBulkTeamActionIDs = 100

func (uc *TeamUseCase) BanTeams(ctx context.Context, teamIDs []uuid.UUID, reason string, banMembers bool, actorID uuid.UUID) (*usecase.BulkActionResult, error) {
	ids, err := normalizeBulkTeamIDs(teamIDs)
	if err != nil {
		return nil, err
	}

	var (
		aggregate teamBanTxResult
		changed   int
	)

	for range maxBanRetries {
		aggregate = teamBanTxResult{}
		changed = 0

		err = uc.deps.TM.Run(ctx, func(ctx context.Context) error {
			for _, id := range ids {
				result, err := uc.banTeamTx(ctx, id, reason, banMembers, actorID)
				if err != nil {
					return err
				}

				if result.changed {
					changed++
				}

				aggregate.memberIDs = append(aggregate.memberIDs, result.memberIDs...)
				aggregate.bannedUserIDs = append(aggregate.bannedUserIDs, result.bannedUserIDs...)
				aggregate.changed = aggregate.changed || result.changed
			}

			return nil
		})
		if err == nil {
			break
		}

		if !errors.Is(err, apperr.ErrTeamConflict) {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("TeamUseCase - BanTeams - TM.Run: %w", err)
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) { uc.afterTeamBanCommit(ctx, ids, aggregate, banMembers) })

	return &usecase.BulkActionResult{AffectedCount: changed}, nil
}

func (uc *TeamUseCase) UnbanTeams(ctx context.Context, teamIDs []uuid.UUID, actorID uuid.UUID) (*usecase.BulkActionResult, error) {
	ids, err := normalizeBulkTeamIDs(teamIDs)
	if err != nil {
		return nil, err
	}

	var (
		memberIDsByTeam = make(map[uuid.UUID][]uuid.UUID, len(ids))
		changed         int
	)

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			var restoredMemberIDs []uuid.UUID

			didChange, err := uc.unbanTeamTx(ctx, id, actorID, &restoredMemberIDs)
			if err != nil {
				return err
			}

			if didChange {
				changed++
			}

			memberIDsByTeam[id] = restoredMemberIDs
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("TeamUseCase - UnbanTeams - TM.Run: %w", err)
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		comp := computil.Cached(ctx, nil, uc.deps.CompRepo)
		frozen := comp != nil && comp.IsFreezeActive()

		for _, id := range ids {
			uc.invalidateTeamAndMembers(ctx, id, memberIDsByTeam[id], frozen)
		}
	})

	return &usecase.BulkActionResult{AffectedCount: changed}, nil
}

func (uc *TeamUseCase) SetHiddenBulk(ctx context.Context, teamIDs []uuid.UUID, hidden bool) (*usecase.BulkActionResult, error) {
	ids, err := normalizeBulkTeamIDs(teamIDs)
	if err != nil {
		return nil, err
	}

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			if err := uc.setHiddenTx(ctx, id, hidden); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("TeamUseCase - SetHiddenBulk - TM.Run: %w", err)
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		for _, id := range ids {
			cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, id)
		}
	})

	return &usecase.BulkActionResult{AffectedCount: len(ids)}, nil
}

func normalizeBulkTeamIDs(ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, apperr.NewValidationErrorf("ids must contain at least one team")
	}

	if len(ids) > maxBulkTeamActionIDs {
		return nil, apperr.NewValidationErrorf("ids cannot contain more than %d teams", maxBulkTeamActionIDs)
	}

	normalized := domain.UniqueUUIDs(ids)
	domain.SortUUIDs(normalized)

	return normalized, nil
}
