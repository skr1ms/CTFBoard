package challenge

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/cache"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
)

type HintDeps struct {
	HintRepo        repo.HintRepository
	HintUnlockRepo  repo.HintUnlockRepository
	AwardRepo       repo.AwardRepository
	TxRepo          repo.TxRepository
	SolveRepo       repo.SolveRepository
	ScoreboardCache cache.ScoreboardCacheInvalidator
}

type HintUseCase struct {
	deps HintDeps
}

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
		return nil, usecaseutil.Wrap(err, "HintUseCase - Create")
	}

	return hint, nil
}

func (uc *HintUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Hint, error) {
	hint, err := uc.deps.HintRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "HintUseCase - GetByID")
	}
	return hint, nil
}

func (uc *HintUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*HintWithUnlockStatus, error) {
	hints, err := uc.deps.HintRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "HintUseCase - GetByChallengeID")
	}

	unlockedMap := make(map[uuid.UUID]bool)
	if teamID != nil {
		unlockedIDs, err := uc.deps.HintUnlockRepo.GetUnlockedHintIDs(ctx, *teamID, challengeID)
		if err != nil {
			return nil, usecaseutil.Wrap(err, "HintUseCase - GetByChallengeID - GetUnlockedHintIDs")
		}
		for _, ID := range unlockedIDs {
			unlockedMap[ID] = true
		}
	}

	result := make([]*HintWithUnlockStatus, 0, len(hints))
	for _, hint := range hints {
		h := &HintWithUnlockStatus{
			Hint:     hint,
			Unlocked: unlockedMap[hint.ID],
		}
		if !h.Unlocked {
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

type HintWithUnlockStatus struct {
	Hint     *entity.Hint
	Unlocked bool
}

func (uc *HintUseCase) Update(ctx context.Context, ID uuid.UUID, content string, cost, orderIndex int) (*entity.Hint, error) {
	hint, err := uc.deps.HintRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "HintUseCase - Update - GetByID")
	}

	hint.Content = content
	hint.Cost = cost
	hint.OrderIndex = orderIndex

	if err := uc.deps.HintRepo.Update(ctx, hint); err != nil {
		return nil, usecaseutil.Wrap(err, "HintUseCase - Update")
	}

	return hint, nil
}

func (uc *HintUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := uc.deps.HintRepo.Delete(ctx, ID); err != nil {
		return usecaseutil.Wrap(err, "HintUseCase - Delete")
	}
	return nil
}

func (uc *HintUseCase) UnlockHint(ctx context.Context, teamID, hintID uuid.UUID) (*entity.Hint, error) {
	hint, err := uc.deps.HintRepo.GetByID(ctx, hintID)
	if err != nil {
		if errors.Is(err, entityError.ErrHintNotFound) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, usecaseutil.Wrap(err, "HintUseCase - UnlockHint - GetByID")
	}
	tx, err := uc.deps.TxRepo.BeginTx(ctx)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "HintUseCase - UnlockHint - BeginTx")
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf("%w, rollback: %w", err, rbErr)
			}
		}
	}()
	if err = uc.unlockHintInTx(ctx, tx, teamID, hintID, hint); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, usecaseutil.Wrap(err, "HintUseCase - UnlockHint - Commit")
	}
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	}
	return hint, nil
}

func (uc *HintUseCase) unlockHintInTx(ctx context.Context, tx repo.Transaction, teamID, hintID uuid.UUID, hint *entity.Hint) error {
	if err := uc.deps.TxRepo.LockTeamTx(ctx, tx, teamID); err != nil {
		return usecaseutil.Wrap(err, "HintUseCase - UnlockHint - LockTeamTx")
	}
	if err := uc.unlockHintCheckAlreadyUnlocked(ctx, tx, teamID, hintID); err != nil {
		return err
	}
	if err := uc.unlockHintChargeIfNeeded(ctx, tx, teamID, hint); err != nil {
		return err
	}
	return uc.deps.TxRepo.CreateHintUnlockTx(ctx, tx, teamID, hintID)
}

func (uc *HintUseCase) unlockHintCheckAlreadyUnlocked(ctx context.Context, tx repo.Transaction, teamID, hintID uuid.UUID) error {
	_, err := uc.deps.TxRepo.GetHintUnlockByTeamAndHintTx(ctx, tx, teamID, hintID)
	if err == nil {
		return entityError.ErrHintAlreadyUnlocked
	}
	if !errors.Is(err, entityError.ErrHintNotFound) {
		return usecaseutil.Wrap(err, "HintUseCase - UnlockHint - GetByTeamAndHintTx")
	}
	return nil
}

func (uc *HintUseCase) unlockHintChargeIfNeeded(ctx context.Context, tx repo.Transaction, teamID uuid.UUID, hint *entity.Hint) error {
	if hint.Cost <= 0 {
		return nil
	}
	teamScore, err := uc.deps.TxRepo.GetTeamScoreTx(ctx, tx, teamID)
	if err != nil {
		return usecaseutil.Wrap(err, "HintUseCase - UnlockHint - GetTeamScoreTx")
	}
	if teamScore < hint.Cost {
		return entityError.ErrInsufficientPoints
	}
	award := &entity.Award{
		TeamID:      teamID,
		Value:       -hint.Cost,
		Description: fmt.Sprintf("Hint unlock: %s", hint.ID),
	}
	return uc.deps.TxRepo.CreateAwardTx(ctx, tx, award)
}
