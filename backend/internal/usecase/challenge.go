package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	pkgRedis "github.com/skr1ms/CTFBoard/pkg/redis"
)

type ChallengeUseCase struct {
	challengeRepo repo.ChallengeRepository
	solveRepo     repo.SolveRepository
	txRepo        repo.TxRepository
	redis         pkgRedis.Client
}

func NewChallengeUseCase(challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository, txRepo repo.TxRepository, redis pkgRedis.Client) *ChallengeUseCase {
	return &ChallengeUseCase{
		challengeRepo: challengeRepo,
		solveRepo:     solveRepo,
		txRepo:        txRepo,
		redis:         redis,
	}
}

func (uc *ChallengeUseCase) GetAll(ctx context.Context, teamId *string) ([]*repo.ChallengeWithSolved, error) {
	challenges, err := uc.challengeRepo.GetAll(ctx, teamId)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll: %w", err)
	}
	return challenges, nil
}

func (uc *ChallengeUseCase) Create(ctx context.Context, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden bool) (*entity.Challenge, error) {
	hash := sha256.Sum256([]byte(flag))
	flagHash := hex.EncodeToString(hash[:])

	challenge := &entity.Challenge{
		Title:        title,
		Description:  description,
		Category:     category,
		Points:       points,
		InitialValue: initialValue,
		MinValue:     minValue,
		Decay:        decay,
		SolveCount:   0,
		FlagHash:     flagHash,
		IsHidden:     isHidden,
	}

	err := uc.challengeRepo.Create(ctx, challenge)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create: %w", err)
	}
	return challenge, nil
}

func (uc *ChallengeUseCase) Update(ctx context.Context, id string, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden bool) (*entity.Challenge, error) {
	challenge, err := uc.challengeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update - GetByID: %w", err)
	}

	challenge.Title = title
	challenge.Description = description
	challenge.Category = category
	challenge.Points = points
	challenge.InitialValue = initialValue
	challenge.MinValue = minValue
	challenge.Decay = decay
	challenge.IsHidden = isHidden

	if flag != "" {
		hash := sha256.Sum256([]byte(flag))
		challenge.FlagHash = hex.EncodeToString(hash[:])
	}

	err = uc.challengeRepo.Update(ctx, challenge)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update: %w", err)
	}

	uc.redis.Del(ctx, "scoreboard")

	return challenge, nil
}

func (uc *ChallengeUseCase) Delete(ctx context.Context, id string) error {
	err := uc.challengeRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - Delete: %w", err)
	}

	uc.redis.Del(ctx, "scoreboard")

	return nil
}

func (uc *ChallengeUseCase) SubmitFlag(ctx context.Context, challengeId, flag, userId string, teamId *string) (bool, error) {
	if teamId == nil {
		return false, entityError.ErrUserMustBeInTeam
	}

	// Quick check if challenge exists and flag is correct (no transaction needed)
	challenge, err := uc.challengeRepo.GetByID(ctx, challengeId)
	if err != nil {
		if errors.Is(err, entityError.ErrChallengeNotFound) {
			return false, entityError.ErrChallengeNotFound
		}
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - GetByID: %w", err)
	}

	hash := sha256.Sum256([]byte(flag))
	flagHash := hex.EncodeToString(hash[:])

	if flagHash != challenge.FlagHash {
		return false, nil
	}

	// Start transaction for atomic solve creation + score update
	tx, err := uc.txRepo.BeginTx(ctx)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - BeginTx: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Check if already solved INSIDE transaction with FOR UPDATE to prevent race
	_, err = uc.solveRepo.GetByTeamAndChallengeTx(ctx, tx, *teamId, challengeId)
	if err == nil {
		return true, entityError.ErrAlreadySolved
	}
	if !errors.Is(err, entityError.ErrSolveNotFound) {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - GetByTeamAndChallengeTx: %w", err)
	}

	// Lock challenge row for update
	challenge, err = uc.challengeRepo.GetByIDTx(ctx, tx, challengeId)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - GetByIDTx: %w", err)
	}

	// Create solve record
	solve := &entity.Solve{
		UserId:      userId,
		TeamId:      *teamId,
		ChallengeId: challengeId,
	}

	err = uc.solveRepo.CreateTx(ctx, tx, solve)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - CreateTx: %w", err)
	}

	// Increment solve count atomically
	solveCount, err := uc.challengeRepo.IncrementSolveCountTx(ctx, tx, challengeId)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - IncrementSolveCountTx: %w", err)
	}

	// Update points if dynamic scoring is enabled
	if challenge.InitialValue > 0 && challenge.Decay > 0 {
		newPoints := CalculateDynamicScore(challenge.InitialValue, challenge.MinValue, challenge.Decay, solveCount)
		if err = uc.challengeRepo.UpdatePointsTx(ctx, tx, challengeId, newPoints); err != nil {
			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - UpdatePointsTx: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - Commit: %w", err)
	}

	uc.redis.Del(ctx, "scoreboard")

	return true, nil
}
