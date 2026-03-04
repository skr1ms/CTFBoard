package challenge

import (
	"context"
	"errors"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
)

type HintUseCase struct {
	deps HintDeps
}

type HintDeps struct {
	HintRepo        repo.HintRepository
	AwardRepo       repo.AwardRepository
	TM              repo.TransactionManager
	SolveRepo       repo.SolveRepository
	CompRepo        repo.CompetitionRepository
	TeamRepo        repo.TeamRepository
	UserRepo        repo.UserRepository
	ChallengeRepo   repo.ChallengeRepository
	ScoreboardCache cache.ScoreboardCacheInvalidator
}

var _ usecase.HintUseCase = (*HintUseCase)(nil)

func NewHintUseCase(deps HintDeps) *HintUseCase {
	return &HintUseCase{deps: deps}
}

func (uc *HintUseCase) Create(ctx context.Context, challengeID uuid.UUID, content string, cost, orderIndex int) (*entity.Hint, error) {
	hint := &entity.Hint{
		ChallengeID: challengeID,
		Content:     content,
		Cost:        cost,
		OrderIndex:  orderIndex,
	}

	if err := uc.deps.HintRepo.Create(ctx, hint); err != nil {
		return nil, fmt.Errorf("HintUseCase - Create - HintRepo.Create: %w", err)
	}

	return hint, nil
}

func (uc *HintUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Hint, error) {
	hint, err := uc.deps.HintRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - GetByID - HintRepo.GetByID: %w", err)
	}
	return hint, nil
}

func (uc *HintUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*usecase.HintWithUnlockStatus, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - GetByChallengeID - ChallengeRepo.GetByID: %w", err)
	}
	if challenge.IsHidden {
		return nil, httperr.ErrChallengeNotFound
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
			h.Hint = &entity.Hint{
				ID:          hint.ID,
				ChallengeID: hint.ChallengeID,
				Cost:        hint.Cost,
				OrderIndex:  hint.OrderIndex,
			}
		}
		result = append(result, h)
	}

	return result, nil
}

func (uc *HintUseCase) Update(ctx context.Context, ID uuid.UUID, content string, cost, orderIndex int) (*entity.Hint, error) {
	hint, err := uc.deps.HintRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - Update - HintRepo.GetByID: %w", err)
	}

	hint.Content = content
	hint.Cost = cost
	hint.OrderIndex = orderIndex

	if err := uc.deps.HintRepo.Update(ctx, hint); err != nil {
		return nil, fmt.Errorf("HintUseCase - Update - HintRepo.Update: %w", err)
	}

	return hint, nil
}

func (uc *HintUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := uc.deps.HintRepo.Delete(ctx, ID); err != nil {
		return fmt.Errorf("HintUseCase - Delete - HintRepo.Delete: %w", err)
	}
	return nil
}

func (uc *HintUseCase) UnlockHint(ctx context.Context, userID, teamID, challengeID, hintID uuid.UUID) (*entity.Hint, error) {
	var hint *entity.Hint
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		comp, err := uc.deps.CompRepo.Get(ctx)
		if err != nil {
			return fmt.Errorf("HintUseCase - UnlockHint - CompetitionRepo.Get: %w", err)
		}
		if !comp.IsSubmissionAllowed() {
			return httperr.ErrSubmissionNotAllowed
		}

		var err2 error
		hint, err2 = uc.deps.HintRepo.GetByID(ctx, hintID)
		if err2 != nil {
			return fmt.Errorf("HintUseCase - UnlockHint - HintRepo.GetByID: %w", err2)
		}
		if hint.ChallengeID != challengeID {
			return httperr.ErrHintNotFound
		}
		challenge, errChal := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
		if errChal != nil {
			return fmt.Errorf("HintUseCase - UnlockHint - ChallengeRepo.GetByID: %w", errChal)
		}
		if challenge.IsHidden {
			return httperr.ErrChallengeNotFound
		}
		return uc.unlockHintInTx(ctx, userID, teamID, hintID, hint, comp)
	})
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - UnlockHint - TM.Run: %w", err)
	}
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	}
	return hint, nil
}

//nolint:gocognit,gocyclo // points deduction + penalty + team-size validation in one transaction
func (uc *HintUseCase) unlockHintInTx(ctx context.Context, userID, teamID, hintID uuid.UUID, hint *entity.Hint, comp *entity.Competition) error {
	// Re-verify membership inside the transaction to close the TOCTOU window between
	// RequireTeam middleware and the hint unlock: if the user was kicked between the
	// two points, the unlock would otherwise be credited to a team they no longer belong to.
	if uc.deps.UserRepo != nil {
		if err := uc.deps.UserRepo.Lock(ctx, userID); err != nil {
			return fmt.Errorf("HintUseCase - UnlockHint - UserRepo.Lock: %w", err)
		}
		freshUser, err := uc.deps.UserRepo.GetByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("HintUseCase - UnlockHint - UserRepo.GetByID: %w", err)
		}
		if freshUser.TeamID == nil || *freshUser.TeamID != teamID {
			return httperr.ErrTeamMemberNotFound
		}
		if freshUser.IsBanned {
			return httperr.ErrUserBanned
		}
	}
	if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - TeamRepo.Lock: %w", err)
	}
	team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
	if err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - TeamRepo.GetByID: %w", err)
	}
	if team.IsBanned {
		return httperr.ErrTeamBanned
	}
	if comp.Mode == entity.ModeTeamsOnly && team.IsSolo {
		return httperr.ErrTeamModeRequired
	}
	if comp.Mode == entity.ModeSoloOnly && !team.IsSolo {
		return httperr.ErrSoloModeRequired
	}
	if comp.MinTeamSize > 0 && !team.IsSolo {
		count, err := uc.deps.TeamRepo.CountTeamMembers(ctx, teamID)
		if err != nil {
			return fmt.Errorf("HintUseCase - UnlockHint - TeamRepo.CountTeamMembers: %w", err)
		}
		if count < comp.MinTeamSize {
			return httperr.ErrTeamBelowMinSize
		}
	}
	if err := uc.unlockHintCheckAlreadyUnlocked(ctx, teamID, hintID); err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - check already unlocked: %w", err)
	}
	if err := uc.unlockHintChargeIfNeeded(ctx, teamID, hint); err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - unlockHintChargeIfNeeded: %w", err)
	}
	if err := uc.deps.HintRepo.CreateUnlock(ctx, teamID, hintID); err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - HintRepo.CreateUnlock: %w", err)
	}
	return nil
}

func (uc *HintUseCase) unlockHintCheckAlreadyUnlocked(ctx context.Context, teamID, hintID uuid.UUID) error {
	_, err := uc.deps.HintRepo.GetByTeamAndHintForUpdate(ctx, teamID, hintID)
	if err == nil {
		return httperr.ErrHintAlreadyUnlocked
	}
	if !errors.Is(err, httperr.ErrHintNotFound) {
		return fmt.Errorf("HintUseCase - UnlockHint - HintRepo.GetByTeamAndHintForUpdate: %w", err)
	}
	return nil
}

func (uc *HintUseCase) unlockHintChargeIfNeeded(ctx context.Context, teamID uuid.UUID, hint *entity.Hint) error {
	if hint.Cost <= 0 {
		return nil
	}
	// GetTeamScore returns net score (solve points + awards, including negative hint costs).
	teamScore, err := uc.deps.SolveRepo.GetTeamScore(ctx, teamID)
	if err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - SolveRepo.GetTeamScore: %w", err)
	}
	if teamScore < hint.Cost {
		return httperr.ErrInsufficientPoints
	}
	award := &entity.Award{
		TeamID:      teamID,
		Value:       -hint.Cost,
		Description: fmt.Sprintf("Hint unlock: hint #%d", hint.OrderIndex+1),
	}
	if err := uc.deps.AwardRepo.Create(ctx, award); err != nil {
		return fmt.Errorf("HintUseCase - UnlockHint - AwardRepo.Create: %w", err)
	}
	return nil
}

func (uc *HintUseCase) GetAllUnlocks(ctx context.Context, page, perPage int) (*usecase.Paginated[*entity.HintUnlockWithDetails], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*entity.HintUnlockWithDetails, error) {
			return uc.deps.HintRepo.GetAll(ctx, limit, offset)
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
