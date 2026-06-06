package challenge

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

func (uc *ChallengeUseCase) GetTags(ctx context.Context, challengeID uuid.UUID) ([]*domain.Tag, error) {
	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetTags - ChallengeRepo.GetByID: %w", err)
	}

	if uc.deps.TagRepo == nil {
		return []*domain.Tag{}, nil
	}

	tags, err := uc.deps.TagRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetTags - TagRepo.GetByChallengeID: %w", err)
	}

	return tags, nil
}

// GetRequirements returns prerequisite challenges for the given challenge.
// Uses singleflight to coalesce concurrent requests; hides the challenge (404) when state is Hidden.
func (uc *ChallengeUseCase) GetRequirements(ctx context.Context, challengeID uuid.UUID) ([]*domain.ChallengeRequirement, error) {
	key := challengeID.String() + ":req:pub"

	v, err, _ := uc.requirementsSf.Do(key, func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		challenge, err := uc.deps.ChallengeRepo.GetByID(loadCtx, challengeID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetRequirements - ChallengeRepo.GetByID: %w", err)
		}

		if err := guard.EnsureChallengeVisible(challenge); err != nil {
			return nil, err
		}

		return uc.deps.ChallengeRepo.GetRequirements(loadCtx, challengeID)
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetRequirements: %w", err)
	}

	requirements, ok := v.([]*domain.ChallengeRequirement)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - GetRequirements: unexpected type")
	}

	return requirements, nil
}

// SetRequirements replaces the requirement list for a challenge after validating that the
// proposed set would not introduce a directed cycle in the prerequisite graph. Candidate
// requirement IDs are validated first; the target challenge check, graph snapshot, cycle
// validation, and write run inside one transaction so the persisted edge set matches the
// graph that was validated.
func (uc *ChallengeUseCase) SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error {
	if len(requirementIDs) > 0 {
		challenges, err := uc.deps.ChallengeRepo.GetByIDs(ctx, requirementIDs)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - SetRequirements - ChallengeRepo.GetByIDs: %w", err)
		}

		for _, reqID := range requirementIDs {
			if _, ok := challenges[reqID]; !ok {
				return apperr.NewValidationErrorf("invalid requirement_id")
			}
		}
	}

	return uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
			return fmt.Errorf("ChallengeUseCase - SetRequirements - ChallengeRepo.GetByID: %w", err)
		}

		pairs, err := uc.deps.ChallengeRepo.GetAllRequirementPairs(ctx)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - SetRequirements - GetAllRequirementPairs: %w", err)
		}

		adj := make(map[uuid.UUID][]uuid.UUID)

		for _, p := range pairs {
			if p.ChallengeID != challengeID {
				adj[p.ChallengeID] = append(adj[p.ChallengeID], p.RequiredChallengeID)
			}
		}

		adj[challengeID] = requirementIDs
		if requirementsContainCycle(challengeID, adj) {
			return apperr.NewValidationErrorf("requirements contain a cycle")
		}

		if err := uc.deps.ChallengeRepo.SetRequirements(ctx, challengeID, requirementIDs); err != nil {
			return fmt.Errorf("ChallengeUseCase - SetRequirements - ChallengeRepo.SetRequirements: %w", err)
		}

		return nil
	})
}

// requirementsContainCycle reports whether the directed graph represented by adj contains
// a cycle reachable from start. It uses a recursive DFS with a single visiting set: a
// node is marked true on entry and false on return, so any back-edge (reaching a node
// already in the current DFS path) is detected as a cycle. The traversal uses
// slices.ContainsFunc to iterate neighbours, which short-circuits on the first true return.
func requirementsContainCycle(start uuid.UUID, adj map[uuid.UUID][]uuid.UUID) bool {
	visiting := make(map[uuid.UUID]bool)

	var dfs func(uuid.UUID) bool

	dfs = func(node uuid.UUID) bool {
		if visiting[node] {
			return true
		}

		visiting[node] = true

		defer func() { visiting[node] = false }()

		return slices.ContainsFunc(adj[node], dfs)
	}

	return dfs(start)
}
