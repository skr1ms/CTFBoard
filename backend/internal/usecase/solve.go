package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type SolveUseCase struct {
	solveRepo repo.SolveRepository
}

func NewSolveUseCase(solveRepo repo.SolveRepository) *SolveUseCase {
	return &SolveUseCase{
		solveRepo: solveRepo,
	}
}

func (uc *SolveUseCase) Create(ctx context.Context, solve *entity.Solve) error {
	_, err := uc.solveRepo.GetByTeamAndChallenge(ctx, solve.TeamId, solve.ChallengeId)
	if err == nil {
		return entityError.ErrAlreadySolved
	}
	if !errors.Is(err, entityError.ErrSolveNotFound) {
		return fmt.Errorf("SolveUseCase - Create - GetByTeamAndChallenge: %w", err)
	}

	err = uc.solveRepo.Create(ctx, solve)
	if err != nil {
		return fmt.Errorf("SolveUseCase - Create: %w", err)
	}
	return nil
}

func (uc *SolveUseCase) GetScoreboard(ctx context.Context) ([]*repo.ScoreboardEntry, error) {
	entries, err := uc.solveRepo.GetScoreboard(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard: %w", err)
	}
	return entries, nil
}
