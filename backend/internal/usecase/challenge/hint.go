package challenge

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type HintUseCase struct {
	hintRepo       repo.HintRepository
	hintUnlockRepo repo.HintUnlockRepository
	awardRepo      repo.AwardRepository
	txRepo         repo.TxRepository
	solveRepo      repo.SolveRepository
	redis          *redis.Client
}

func NewHintUseCase(
	hintRepo repo.HintRepository,
	hintUnlockRepo repo.HintUnlockRepository,
	awardRepo repo.AwardRepository,
	txRepo repo.TxRepository,
	solveRepo repo.SolveRepository,
	redis *redis.Client,
) *HintUseCase {
	return &HintUseCase{
		hintRepo:       hintRepo,
		hintUnlockRepo: hintUnlockRepo,
		awardRepo:      awardRepo,
		txRepo:         txRepo,
		solveRepo:      solveRepo,
		redis:          redis,
	}
}

func (uc *HintUseCase) Create(ctx context.Context, challengeID uuid.UUID, content string, cost, orderIndex int) (*entity.Hint, error) {
	hint := &entity.Hint{
		ChallengeID: challengeID,
		Content:     content,
		Cost:        cost,
		OrderIndex:  orderIndex,
	}

	if err := uc.hintRepo.Create(ctx, hint); err != nil {
		return nil, fmt.Errorf("HintUseCase - Create: %w", err)
	}

	return hint, nil
}

func (uc *HintUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Hint, error) {
	hint, err := uc.hintRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - GetByID: %w", err)
	}
	return hint, nil
}

func (uc *HintUseCase) GetByChallengeID(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*HintWithUnlockStatus, error) {
	hints, err := uc.hintRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - GetByChallengeID: %w", err)
	}

	unlockedMap := make(map[uuid.UUID]bool)
	if teamID != nil {
		unlockedIDs, err := uc.hintUnlockRepo.GetUnlockedHintIDs(ctx, *teamID, challengeID)
		if err != nil {
			return nil, fmt.Errorf("HintUseCase - GetByChallengeID - GetUnlockedHintIDs: %w", err)
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
	hint, err := uc.hintRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - Update - GetByID: %w", err)
	}

	hint.Content = content
	hint.Cost = cost
	hint.OrderIndex = orderIndex

	if err := uc.hintRepo.Update(ctx, hint); err != nil {
		return nil, fmt.Errorf("HintUseCase - Update: %w", err)
	}

	return hint, nil
}

func (uc *HintUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := uc.hintRepo.Delete(ctx, ID); err != nil {
		return fmt.Errorf("HintUseCase - Delete: %w", err)
	}
	return nil
}

//nolint:gocognit,gocyclo
func (uc *HintUseCase) UnlockHint(ctx context.Context, teamID, hintID uuid.UUID) (*entity.Hint, error) {
	hint, err := uc.hintRepo.GetByID(ctx, hintID)
	if err != nil {
		if errors.Is(err, entityError.ErrHintNotFound) {
			return nil, entityError.ErrHintNotFound
		}
		return nil, fmt.Errorf("HintUseCase - UnlockHint - GetByID: %w", err)
	}

	tx, err := uc.txRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("HintUseCase - UnlockHint - BeginTx: %w", err)
	}

	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf("%w, rollback: %w", err, rbErr)
			}
		}
	}()

	if err := uc.txRepo.LockTeamTx(ctx, tx, teamID); err != nil {
		return nil, fmt.Errorf("HintUseCase - UnlockHint - LockTeamTx: %w", err)
	}

	_, err = uc.txRepo.GetHintUnlockByTeamAndHintTx(ctx, tx, teamID, hintID)
	if err == nil {
		err = entityError.ErrHintAlreadyUnlocked
		return nil, err
	}
	if !errors.Is(err, entityError.ErrHintNotFound) {
		return nil, fmt.Errorf("HintUseCase - UnlockHint - GetByTeamAndHintTx: %w", err)
	}

	if hint.Cost > 0 {
		teamScore, err := uc.txRepo.GetTeamScoreTx(ctx, tx, teamID)
		if err != nil {
			return nil, fmt.Errorf("HintUseCase - UnlockHint - GetTeamScoreTx: %w", err)
		}

		if teamScore < hint.Cost {
			err = entityError.ErrInsufficientPoints
			return nil, err
		}

		award := &entity.Award{
			TeamID:      teamID,
			Value:       -hint.Cost,
			Description: fmt.Sprintf("Hint unlock: %s", hint.ID),
		}

		if err = uc.txRepo.CreateAwardTx(ctx, tx, award); err != nil {
			return nil, fmt.Errorf("HintUseCase - UnlockHint - CreateAwardTx: %w", err)
		}
	}

	if err = uc.txRepo.CreateHintUnlockTx(ctx, tx, teamID, hintID); err != nil {
		return nil, fmt.Errorf("HintUseCase - UnlockHint - CreateUnlockTx: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("HintUseCase - UnlockHint - Commit: %w", err)
	}

	uc.redis.Del(ctx, "scoreboard")
	uc.redis.Del(ctx, "scoreboard:frozen")

	return hint, nil
}
