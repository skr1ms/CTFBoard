package challenge

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type HintUseCase struct {
	deps HintDeps
}

type hintCompGetter interface {
	Get(ctx context.Context) (*domain.Competition, error)
}

type HintDeps struct {
	HintRepo        repo.HintRepository
	AwardRepo       repo.AwardRepository
	TM              repo.TransactionManager
	SolveRepo       repo.SolveRepository
	CompRepo        repo.CompetitionRepository
	CompGetter      hintCompGetter
	TeamRepo        repo.TeamRepository
	UserRepo        repo.UserRepository
	ChallengeRepo   repo.ChallengeRepository
	ScoreboardCache cache.ScoreboardCacheInvalidator
	Logger          logkit.Logger
}

var _ usecase.HintUseCase = (*HintUseCase)(nil)

func NewHintUseCase(deps HintDeps) *HintUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &HintUseCase{deps: deps}
}

func (uc *HintUseCase) Create(ctx context.Context, challengeID uuid.UUID, title, content string, cost, orderIndex int) (*domain.Hint, error) {
	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
		return nil, fmt.Errorf("HintUseCase - Create - ChallengeRepo.GetByID: %w", err)
	}

	if cost < 0 {
		return nil, httperr.NewValidationErrorf("hint cost must be non-negative")
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

func (uc *HintUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Hint, error) {
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

	if challenge.State == domain.ChallengeStateHidden {
		return nil, httperr.ErrChallengeNotFound
	}

	reqs, err := uc.deps.ChallengeRepo.GetRequirements(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - GetByChallengeID - GetRequirements: %w", err)
	}

	if len(reqs) > 0 {
		if teamID == nil || uc.deps.SolveRepo == nil {
			return nil, httperr.ErrChallengeNotFound
		}

		met, err := requirementsMet(ctx, challengeID, *teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
		if err != nil {
			return nil, fmt.Errorf("HintUseCase - GetByChallengeID - requirementsMet: %w", err)
		}

		if !met {
			return nil, httperr.ErrChallengeNotFound
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

func (uc *HintUseCase) Update(ctx context.Context, ID uuid.UUID, title, content string, cost, orderIndex int) (*domain.Hint, error) {
	if cost < 0 {
		return nil, httperr.NewValidationErrorf("hint cost must be non-negative")
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
		return nil, err
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

		if comp == nil {
			var errComp error

			comp, errComp = uc.getCompetition(ctx)
			if errComp != nil {
				return fmt.Errorf("HintUseCase - UnlockHint - getCompetition: %w", errComp)
			}
		}

		if comp == nil {
			return httperr.ErrCompetitionNotFound
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

		if challenge.State == domain.ChallengeStateHidden {
			return httperr.ErrChallengeNotFound
		}

		if uc.deps.SolveRepo != nil {
			met, errReq := requirementsMet(ctx, challengeID, teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
			if errReq != nil {
				return fmt.Errorf("HintUseCase - UnlockHint - requirementsMet: %w", errReq)
			}

			if !met {
				return httperr.ErrChallengeNotFound
			}
		}

		return uc.unlockHintInTx(ctx, userID, teamID, hintID, hint, comp)
	})
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - UnlockHint - TM.Run: %w", err)
	}

	if uc.deps.ScoreboardCache != nil {
		comp, err := uc.getCompetition(ctx)
		if err == nil && comp != nil && comp.IsFreezeActive() {
			uc.deps.ScoreboardCache.InvalidateLiveOnly(ctx, teamID)

			return hint, nil
		}

		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	}

	return hint, nil
}

func (uc *HintUseCase) unlockHintInTx(ctx context.Context, userID, teamID, hintID uuid.UUID, hint *domain.Hint, comp *domain.Competition) error {
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

	if comp.Mode == domain.ModeTeamsOnly && team.IsSolo {
		return httperr.ErrTeamModeRequired
	}

	if comp.Mode == domain.ModeSoloOnly && !team.IsSolo {
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

	if uc.deps.SolveRepo != nil {
		_, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, teamID, hint.ChallengeID)
		if err != nil && !errors.Is(err, httperr.ErrSolveNotFound) {
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
				return httperr.ErrHintOrderRequired
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
		return httperr.ErrInsufficientPoints
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

func (uc *HintUseCase) acquireBalanceLock(ctx context.Context, teamID uuid.UUID) error {
	keyBytes := teamID[8:16]
	key := int64(uint64(keyBytes[0])<<56 | uint64(keyBytes[1])<<48 | uint64(keyBytes[2])<<40 | uint64(keyBytes[3])<<32 |
		uint64(keyBytes[4])<<24 | uint64(keyBytes[5])<<16 | uint64(keyBytes[6])<<8 | uint64(keyBytes[7]))
	key &= 0x7FFFFFFFFFFFFFFF

	db := uc.deps.TM.DB(ctx)
	if _, err := db.Exec(ctx, "SELECT pg_advisory_xact_lock($1::bigint)", key); err != nil {
		return fmt.Errorf("acquireBalanceLock: %w", err)
	}

	return nil
}

func (uc *HintUseCase) GetAllUnlocks(ctx context.Context, page, perPage int) (*usecase.Paginated[*domain.HintUnlockWithDetails], error) {
	result, err := usecase.FetchPage(ctx, page, perPage,
		func(ctx context.Context, limit, offset int) ([]*domain.HintUnlockWithDetails, error) {
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

func (uc *HintUseCase) getCompetition(ctx context.Context) (*domain.Competition, error) {
	if uc.deps.CompGetter != nil {
		return uc.deps.CompGetter.Get(ctx)
	}

	if uc.deps.CompRepo != nil {
		return uc.deps.CompRepo.Get(ctx)
	}

	return nil, nil
}
