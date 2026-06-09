package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

// SetHidden locks the team row, updates the hidden flag, recalculates affected
// challenge scores, then invalidates caches derived from team visibility.
func (uc *TeamUseCase) SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		_, err := uc.setHiddenTx(ctx, teamID, hidden)

		return err
	})
	if err != nil {
		return fmt.Errorf("TeamUseCase - SetHidden - TM.Run: %w", err)
	}

	cacheutil.InvalidateScoreboardForTeam(ctx, uc.deps.ScoreboardCache, teamID)
	cacheutil.InvalidateChallengeList(ctx, uc.deps.ChallengeListCache)
	cacheutil.InvalidateStatistics(ctx, uc.deps.StatsCache, uc.deps.Logger, "TeamUseCase - SetHidden")

	return nil
}

func (uc *TeamUseCase) setHiddenTx(ctx context.Context, teamID uuid.UUID, hidden bool) (bool, error) {
	if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
		return false, fmt.Errorf("TeamUseCase - setHiddenTx - TeamRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return false, fmt.Errorf("TeamUseCase - setHiddenTx - TeamRepo.GetByID: %w", err)
	}

	if team.IsHidden == hidden {
		return false, nil
	}

	challengeIDs, err := uc.getChallengeIDsForTeam(ctx, teamID)
	if err != nil {
		return false, fmt.Errorf("TeamUseCase - setHiddenTx - getChallengeIDsForTeam: %w", err)
	}

	if err := uc.deps.TeamRepo.SetHidden(ctx, teamID, hidden); err != nil {
		return false, fmt.Errorf("TeamUseCase - setHiddenTx - TeamRepo.SetHidden: %w", err)
	}

	if err := uc.adjustSolveCountsForChallenges(ctx, challengeIDs); err != nil {
		return false, fmt.Errorf("TeamUseCase - setHiddenTx - adjustSolveCountsForChallenges: %w", err)
	}

	return true, nil
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
	cacheutil.InvalidateStatistics(ctx, uc.deps.StatsCache, uc.deps.Logger, "TeamUseCase - SetBracket")

	return nil
}
