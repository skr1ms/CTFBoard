package challenge

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// UnlockHint validates the competition state (submission window open) and prerequisite
// satisfaction for the owning challenge before delegating to unlockHintInTx. The
// competition row is read with GetForUpdate when available so the check is consistent with
// the rest of the transaction. After a successful unlock, scoreboard cache entries are
// invalidated; if the competition freeze is active only the live-only cache keys are
// purged so the frozen scoreboard is not disturbed.
func (uc *HintUseCase) UnlockHint(ctx context.Context, userID, teamID, challengeID, hintID uuid.UUID) (*domain.Hint, error) {
	var hint *domain.Hint

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var comp *domain.Competition

		if uc.deps.CompRepo != nil {
			var errComp error

			comp, errComp = uc.deps.CompRepo.GetForUpdate(ctx)
			if errComp != nil {
				return fmt.Errorf("HintUseCase - UnlockHint - CompRepo.GetForUpdate: %w", errComp)
			}
		}

		if comp == nil && uc.deps.CompUC != nil {
			var errComp error

			comp, errComp = uc.deps.CompUC.Get(ctx)
			if errComp != nil {
				return fmt.Errorf("HintUseCase - UnlockHint - CompUC.Get: %w", errComp)
			}
		}

		if comp == nil {
			return apperr.ErrCompetitionNotFound
		}

		if !comp.IsSubmissionAllowed() {
			return apperr.ErrSubmissionNotAllowed
		}

		var errHint error

		hint, errHint = uc.deps.HintRepo.GetByID(ctx, hintID)
		if errHint != nil {
			return fmt.Errorf("HintUseCase - UnlockHint - HintRepo.GetByID: %w", errHint)
		}

		if hint.ChallengeID != challengeID {
			return apperr.ErrHintNotFound
		}

		challenge, errChal := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
		if errChal != nil {
			return fmt.Errorf("HintUseCase - UnlockHint - ChallengeRepo.GetByID: %w", errChal)
		}

		if err := guard.EnsureChallengeVisible(challenge); err != nil {
			return err
		}

		if uc.deps.SolveRepo != nil {
			met, errReq := requirementsMet(ctx, challengeID, teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
			if errReq != nil {
				return fmt.Errorf("HintUseCase - UnlockHint - requirementsMet: %w", errReq)
			}

			if !met {
				return apperr.ErrChallengeNotFound
			}
		}

		return uc.unlockHintInTx(ctx, userID, teamID, hintID, hint, comp)
	})
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - UnlockHint - TM.Run: %w", err)
	}

	comp := computil.Cached(ctx, uc.deps.CompUC, uc.deps.CompRepo)
	frozen := comp != nil && comp.IsFreezeActive()
	cacheutil.InvalidateWithFreezeAwareness(ctx, uc.deps.ScoreboardCache, teamID, frozen)

	return hint, nil
}

// unlockHintInTx performs the hint unlock inside an already-open transaction
// It re-verifies team membership with a row lock to close the TOCTOU window between the
// RequireTeam middleware check and this point (a user kicked in between must not charge
// their old team). It then locks the team row and re-validates ban status and competition
// mode constraints. Sequential unlock order is enforced: every hint whose OrderIndex is
// lower than the target's - or whose ID sorts earlier at the same index - must already be
// unlocked; violating this returns ErrHintOrderRequired. Finally, if the hint has a cost,
// acquireBalanceLock serialises concurrent unlocks for the same team before the balance
// check and award deduction.
func (uc *HintUseCase) unlockHintInTx(ctx context.Context, userID, teamID, hintID uuid.UUID, hint *domain.Hint, comp *domain.Competition) error {
	// Re-verify membership inside the transaction to close the TOCTOU window between
	// RequireTeam middleware and the hint unlock: if the user was kicked between the
	// two points, the unlock would otherwise be credited to a team they no longer belong to
	var freshUser *domain.User

	if uc.deps.UserRepo != nil {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			return fmt.Errorf("HintUseCase - UnlockHint - UserRepo.Lock: %w", err)
		}

		u, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("HintUseCase - UnlockHint - UserRepo.GetByID: %w", err)
		}

		if u.TeamID == nil || *u.TeamID != teamID {
			return apperr.ErrTeamMemberNotFound
		}

		if u.IsBanned {
			return apperr.ErrUserBanned
		}

		freshUser = u
	}

	if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - TeamRepo.Lock: %w", err)
	}

	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - TeamRepo.GetByID: %w", err)
	}

	if err := guard.ValidateSubmissionEligibility(ctx, freshUser, team, comp, uc.deps.TeamRepo); err != nil {
		return err
	}

	if uc.deps.SolveRepo != nil {
		_, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, teamID, hint.ChallengeID)
		if err != nil && !errors.Is(err, apperr.ErrSolveNotFound) {
			return fmt.Errorf("HintUseCase - UnlockHint - SolveRepo.GetByTeamAndChallenge: %w", err)
		}
	}

	hints, err := uc.deps.HintRepo.GetByChallengeID(ctx, hint.ChallengeID)
	if err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - HintRepo.GetByChallengeID: %w", err)
	}

	unlockedIDs, err := uc.deps.HintRepo.GetUnlockedHintIDs(ctx, teamID, hint.ChallengeID)
	if err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - HintRepo.GetUnlockedHintIDs: %w", err)
	}

	unlockedSet := make(map[uuid.UUID]struct{}, len(unlockedIDs))
	for _, id := range unlockedIDs {
		unlockedSet[id] = struct{}{}
	}

	for _, h := range hints {
		if h.ID == hint.ID {
			continue
		}

		mustBeUnlocked := h.OrderIndex < hint.OrderIndex ||
			(h.OrderIndex == hint.OrderIndex && h.ID.String() < hint.ID.String())
		if mustBeUnlocked {
			if _, ok := unlockedSet[h.ID]; !ok {
				return apperr.ErrHintOrderRequired
			}
		}
	}

	if err := uc.unlockHintChargeIfNeeded(ctx, teamID, hint); err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - unlockHintChargeIfNeeded: %w", err)
	}

	if err := uc.deps.HintRepo.CreateUnlock(ctx, teamID, hintID); err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - HintRepo.CreateUnlock: %w", err)
	}

	return nil
}

func (uc *HintUseCase) unlockHintChargeIfNeeded(ctx context.Context, teamID uuid.UUID, hint *domain.Hint) error {
	if hint.Cost <= 0 {
		return nil
	}

	if err := uc.acquireBalanceLock(ctx, teamID); err != nil {
		return fmt.Errorf("HintUseCase - unlockHintChargeIfNeeded - acquireBalanceLock: %w", err)
	}

	teamScore, err := uc.deps.SolveRepo.GetTeamScore(ctx, teamID)
	if err != nil {
		return fmt.Errorf("HintUseCase - unlockHintChargeIfNeeded - GetTeamScore: %w", err)
	}

	if teamScore < hint.Cost {
		return apperr.ErrInsufficientPoints
	}

	award := &domain.Award{
		TeamID:      teamID,
		Value:       -hint.Cost,
		Description: fmt.Sprintf("Hint unlock: hint #%d", hint.OrderIndex+1),
	}
	if err := uc.deps.AwardRepo.Create(ctx, award); err != nil {
		return fmt.Errorf("HintUseCase - unlockHintChargeIfNeeded - AwardRepo.Create: %w", err)
	}

	return nil
}

// acquireBalanceLock acquires a PostgreSQL transaction-scoped advisory lock keyed by the
// second 8 bytes of the team UUID (bytes 8-15, interpreted as a big-endian uint64 with the
// sign bit cleared to fit a signed int64). This serialises concurrent hint-unlock
// transactions for the same team so that the score read and award deduction are atomic
// with respect to other ongoing unlocks, preventing a team from spending more points than
// it has when multiple hints are unlocked simultaneously.
func (uc *HintUseCase) acquireBalanceLock(ctx context.Context, teamID uuid.UUID) error {
	keyBytes := teamID[8:16]
	key := int64(uint64(keyBytes[0])<<56 | uint64(keyBytes[1])<<48 | uint64(keyBytes[2])<<40 | uint64(keyBytes[3])<<32 |
		uint64(keyBytes[4])<<24 | uint64(keyBytes[5])<<16 | uint64(keyBytes[6])<<8 | uint64(keyBytes[7]))
	key &= 0x7FFFFFFFFFFFFFFF

	return uc.deps.HintRepo.AcquireAdvisoryLock(ctx, key)
}
