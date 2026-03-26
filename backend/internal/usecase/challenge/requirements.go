package challenge

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

func requirementsMet(ctx context.Context, challengeID, teamID uuid.UUID, challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository) (bool, error) {
	if solveRepo == nil {
		return true, nil
	}

	requirements, err := challengeRepo.GetRequirements(ctx, challengeID)
	if err != nil {
		return false, fmt.Errorf("requirementsMet - GetRequirements: %w", err)
	}

	if len(requirements) == 0 {
		return true, nil
	}

	requirementIDs := make([]uuid.UUID, 0, len(requirements))
	for _, req := range requirements {
		requirementIDs = append(requirementIDs, req.ChallengeID)
	}

	solvedIDs, err := solveRepo.GetSolvedChallengeIDsByTeam(ctx, teamID, requirementIDs)
	if err != nil {
		return false, fmt.Errorf("requirementsMet - GetSolvedChallengeIDsByTeam: %w", err)
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
