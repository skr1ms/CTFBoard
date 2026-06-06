package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

// SetHidden locks the team row, updates the hidden flag, then invalidates the
// scoreboard cache for that team only. The lock prevents a concurrent visibility
// flip from racing with score recalculations.
func (uc *TeamUseCase) SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - SetHidden - TeamRepo.Lock: %w", err)
		}

		_, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - SetHidden - TeamRepo.GetByID: %w", err)
		}

		if err := uc.deps.TeamRepo.SetHidden(ctx, teamID, hidden); err != nil {
			return fmt.Errorf("TeamUseCase - SetHidden - TeamRepo.SetHidden: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - SetHidden - TM.Run: %w", err)
	}

	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)

	return nil
}

// SetBracket locks the team row, assigns the bracket (nil removes it), then invalidates
// the scoreboard cache. Scoreboard entries are bracket-partitioned so any bracket change
// requires cache eviction.
func (uc *TeamUseCase) SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
			return fmt.Errorf("TeamUseCase - SetBracket - TeamRepo.Lock: %w", err)
		}

		_, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return fmt.Errorf("TeamUseCase - SetBracket - TeamRepo.GetByID: %w", err)
		}

		if err := uc.deps.TeamRepo.SetBracket(ctx, teamID, bracketID); err != nil {
			return fmt.Errorf("TeamUseCase - SetBracket - TeamRepo.SetBracket: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - SetBracket - TM.Run: %w", err)
	}

	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)

	return nil
}
