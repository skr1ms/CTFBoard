package challenge

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

// GetSolution returns the official solution for a challenge.
// Access is granted only to admins, challenge authors, or teams that have already solved it.
func (uc *ChallengeUseCase) GetSolution(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*domain.ChallengeSolution, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolution - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, err
	}

	if teamID == nil {
		return nil, apperr.ErrNotAuthenticated
	}

	if uc.deps.TeamRepo != nil {
		team, err := uc.deps.TeamRepo.GetByID(ctx, *teamID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetSolution - TeamRepo.GetByID: %w", err)
		}

		if team.IsBanned {
			return nil, apperr.ErrTeamBanned
		}
	}

	if _, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, *teamID, challengeID); err != nil {
		if errors.Is(err, apperr.ErrSolveNotFound) {
			return nil, apperr.ErrSolutionAccessDenied
		}

		return nil, fmt.Errorf("ChallengeUseCase - GetSolution - SolveRepo.GetByTeamAndChallenge: %w", err)
	}

	solution, err := uc.deps.ChallengeRepo.GetSolution(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolution - ChallengeRepo.GetSolution: %w", err)
	}

	return solution, nil
}

func (uc *ChallengeUseCase) ListSolutions(ctx context.Context, teamID uuid.UUID) ([]*domain.ChallengeSolutionEntry, error) {
	if uc.deps.TeamRepo != nil {
		team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - ListSolutions - TeamRepo.GetByID: %w", err)
		}

		if team.IsBanned {
			return nil, apperr.ErrTeamBanned
		}
	}

	entries, err := uc.deps.ChallengeRepo.ListSolutions(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - ListSolutions - ChallengeRepo.ListSolutions: %w", err)
	}

	return entries, nil
}

func (uc *ChallengeUseCase) GetFlags(ctx context.Context, challengeID uuid.UUID) (*domain.ChallengeFlags, error) {
	flags, err := uc.deps.ChallengeRepo.GetFlags(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetFlags - ChallengeRepo.GetFlags: %w", err)
	}

	return flags, nil
}

func (uc *ChallengeUseCase) GetTypes(_ context.Context) ([]string, error) {
	return []string{domain.ChallengeTypeStandard, domain.ChallengeTypeDynamic}, nil
}

func (uc *ChallengeUseCase) GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Challenge, error) {
	challenges, err := uc.deps.ChallengeRepo.GetMissingChallengesByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetMissingChallengesByTeamID - ChallengeRepo.GetMissingChallengesByTeamID: %w", err)
	}

	return challenges, nil
}

// GetMissingChallengesByUserID returns challenges not yet solved by the user's team
// Returns an empty list if the user has no team (user.TeamID == nil).
func (uc *ChallengeUseCase) GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Challenge, error) {
	if uc.deps.UserRepo == nil {
		return []*domain.Challenge{}, nil
	}

	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperr.ErrUserNotFound) {
			return []*domain.Challenge{}, nil
		}

		return nil, fmt.Errorf("ChallengeUseCase - GetMissingChallengesByUserID - UserRepo.GetByID: %w", err)
	}

	if user == nil || user.TeamID == nil {
		return []*domain.Challenge{}, nil
	}

	challenges, err := uc.deps.ChallengeRepo.GetMissingChallengesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetMissingChallengesByUserID - ChallengeRepo.GetMissingChallengesByUserID: %w", err)
	}

	return challenges, nil
}
