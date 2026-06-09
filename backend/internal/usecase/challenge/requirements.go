package challenge

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

func requirementsMet(ctx context.Context, challengeID, teamID uuid.UUID, challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository) (bool, error) {
	if solveRepo == nil {
		return true, nil
	}

	requirements, err := challengeRepo.GetRequirementsForEnforcement(ctx, challengeID)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - requirementsMet - GetRequirementsForEnforcement: %w", err)
	}

	return requirementsSatisfied(ctx, requirements, teamID, solveRepo)
}

func requirementsSatisfied(ctx context.Context, requirements []*domain.ChallengeRequirement, teamID uuid.UUID, solveRepo repo.SolveRepository) (bool, error) {
	if len(requirements) == 0 {
		return true, nil
	}

	if solveRepo == nil {
		return true, nil
	}

	requirementIDs := make([]uuid.UUID, 0, len(requirements))
	for _, req := range requirements {
		requirementIDs = append(requirementIDs, req.ChallengeID)
	}

	solvedIDs, err := solveRepo.GetSolvedChallengeIDsByTeam(ctx, teamID, requirementIDs)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - requirementsMet - GetSolvedChallengeIDsByTeam: %w", err)
	}

	solvedSet := make(map[uuid.UUID]struct{}, len(solvedIDs))
	for _, id := range solvedIDs {
		solvedSet[id] = struct{}{}
	}

	for _, req := range requirements {
		if _, ok := solvedSet[req.ChallengeID]; !ok {
			return false, nil
		}
	}

	return true, nil
}

func ensureRequirementsSatisfiedForRead(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID, challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository, op string) error {
	requirements, err := challengeRepo.GetRequirementsForEnforcement(ctx, challengeID)
	if err != nil {
		return fmt.Errorf("%s - GetRequirementsForEnforcement: %w", op, err)
	}

	if len(requirements) == 0 {
		return nil
	}

	if teamID == nil || solveRepo == nil {
		return apperr.ErrChallengeNotFound
	}

	met, err := requirementsSatisfied(ctx, requirements, *teamID, solveRepo)
	if err != nil {
		return fmt.Errorf("%s - requirementsSatisfied: %w", op, err)
	}

	if !met {
		return apperr.ErrChallengeNotFound
	}

	return nil
}
