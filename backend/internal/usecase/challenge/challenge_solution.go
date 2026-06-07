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
// Access is granted according to the solution state, global writeup setting,
// event lifecycle, and team solve status. Admin callers bypass player policy.
func (uc *ChallengeUseCase) GetSolution(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID, isAdmin bool) (*domain.ChallengeSolution, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolution - ChallengeRepo.GetByID: %w", err)
	}

	if !isAdmin {
		if err := guard.EnsureChallengeVisible(challenge); err != nil {
			return nil, err
		}
	}

	if teamID != nil && uc.deps.TeamRepo != nil {
		team, err := uc.deps.TeamRepo.GetByID(ctx, *teamID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetSolution - TeamRepo.GetByID: %w", err)
		}

		if team.IsBanned {
			return nil, apperr.ErrTeamBanned
		}
	}

	solution, err := uc.deps.ChallengeRepo.GetSolution(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolution - ChallengeRepo.GetSolution: %w", err)
	}

	err = ensureSolutionPolicyAccess(ctx, solution.State, challengeID, teamID, isAdmin, uc.deps.SettingsRepo, uc.deps.CompRepo, uc.deps.SolveRepo, apperr.ErrSolutionAccessDenied)
	if err != nil {
		return nil, err
	}

	return solution, nil
}

func (uc *ChallengeUseCase) ListSolutions(ctx context.Context, teamID *uuid.UUID, isAdmin bool) ([]*domain.ChallengeSolutionEntry, error) {
	if teamID != nil && uc.deps.TeamRepo != nil {
		team, err := uc.deps.TeamRepo.GetByID(ctx, *teamID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - ListSolutions - TeamRepo.GetByID: %w", err)
		}

		if team.IsBanned {
			return nil, apperr.ErrTeamBanned
		}
	}

	entries, err := uc.deps.ChallengeRepo.ListSolutions(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - ListSolutions - ChallengeRepo.ListSolutions: %w", err)
	}

	if isAdmin || len(entries) == 0 {
		return entries, nil
	}

	writeupEnabled, err := loadWriteupEnabled(ctx, uc.deps.SettingsRepo)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - ListSolutions - loadWriteupEnabled: %w", err)
	}

	if !writeupEnabled {
		return nil, apperr.ErrWriteupsDisabled
	}

	eventEnded := false

	if solutionEntriesNeedEventStatus(entries) {
		eventEnded, err = loadCompetitionEnded(ctx, uc.deps.CompRepo)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - ListSolutions - loadCompetitionEnded: %w", err)
		}
	}

	solved := map[uuid.UUID]struct{}{}

	if teamID != nil && uc.deps.SolveRepo != nil {
		challengeIDs := make([]uuid.UUID, 0, len(entries))
		for _, entry := range entries {
			challengeIDs = append(challengeIDs, entry.ChallengeID)
		}

		solvedIDs, err := uc.deps.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, *teamID, challengeIDs)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - ListSolutions - SolveRepo.GetSolvedChallengeIDsByTeam: %w", err)
		}

		for _, id := range solvedIDs {
			solved[id] = struct{}{}
		}
	}

	out := make([]*domain.ChallengeSolutionEntry, 0, len(entries))
	for _, entry := range entries {
		_, isSolved := solved[entry.ChallengeID]
		if domain.CanViewSolution(entry.State, isSolved, eventEnded, writeupEnabled, false) {
			out = append(out, entry)
		}
	}

	return out, nil
}

func solutionEntriesNeedEventStatus(entries []*domain.ChallengeSolutionEntry) bool {
	for _, entry := range entries {
		if domain.SolutionStateOrDefault(entry.State) == domain.SolutionStateAfterEvent {
			return true
		}
	}

	return false
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
