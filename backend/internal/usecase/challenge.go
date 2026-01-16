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
	redis         pkgRedis.Client
}

func NewChallengeUseCase(challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository, redis pkgRedis.Client) *ChallengeUseCase {
	return &ChallengeUseCase{
		challengeRepo: challengeRepo,
		solveRepo:     solveRepo,
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

func (uc *ChallengeUseCase) Create(ctx context.Context, title, description, category string, points int, flag string, isHidden bool) (*entity.Challenge, error) {
	hash := sha256.Sum256([]byte(flag))
	flagHash := hex.EncodeToString(hash[:])

	challenge := &entity.Challenge{
		Title:       title,
		Description: description,
		Category:    category,
		Points:      points,
		FlagHash:    flagHash,
		IsHidden:    isHidden,
	}

	err := uc.challengeRepo.Create(ctx, challenge)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create: %w", err)
	}
	return challenge, nil
}

func (uc *ChallengeUseCase) Update(ctx context.Context, id string, title, description, category string, points int, flag string, isHidden bool) (*entity.Challenge, error) {
	challenge, err := uc.challengeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update - GetByID: %w", err)
	}

	challenge.Title = title
	challenge.Description = description
	challenge.Category = category
	challenge.Points = points
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

	_, err = uc.solveRepo.GetByTeamAndChallenge(ctx, *teamId, challengeId)
	if err == nil {
		return true, entityError.ErrAlreadySolved
	}
	if !errors.Is(err, entityError.ErrSolveNotFound) {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - GetByTeamAndChallenge: %w", err)
	}

	solve := &entity.Solve{
		UserId:      userId,
		TeamId:      *teamId,
		ChallengeId: challengeId,
	}

	err = uc.solveRepo.Create(ctx, solve)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - Create: %w", err)
	}

	uc.redis.Del(ctx, "scoreboard")

	return true, nil
}
