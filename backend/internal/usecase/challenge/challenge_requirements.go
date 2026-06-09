package challenge

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
)

func (uc *ChallengeUseCase) GetTags(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*domain.Tag, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetTags - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, err
	}

	if err := ensureRequirementsSatisfiedForRead(ctx, challengeID, teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo, "ChallengeUseCase - GetTags"); err != nil {
		return nil, err
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
// Uses singleflight to coalesce the public metadata read, then applies caller-specific
// requirement visibility checks so locked challenges cannot leak prerequisite metadata.
func (uc *ChallengeUseCase) GetRequirements(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID, isAdmin bool) ([]*domain.ChallengeRequirement, error) {
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

	if !isAdmin {
		filtered, err := uc.requirementsVisibleToCaller(ctx, requirements, teamID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetRequirements - requirementsVisibleToCaller: %w", err)
		}

		requirements = filtered
	}

	return requirements, nil
}

func (uc *ChallengeUseCase) requirementsVisibleToCaller(ctx context.Context, requirements []*domain.ChallengeRequirement, teamID *uuid.UUID) ([]*domain.ChallengeRequirement, error) {
	if len(requirements) == 0 {
		return requirements, nil
	}

	anonymize := uc.shouldAnonymizePrerequisites(ctx)
	if teamID == nil || uc.deps.SolveRepo == nil {
		if anonymize {
			return []*domain.ChallengeRequirement{}, nil
		}

		return nil, apperr.ErrChallengeNotFound
	}

	met, err := requirementsSatisfied(ctx, requirements, *teamID, uc.deps.SolveRepo)
	if err != nil {
		return nil, fmt.Errorf("requirementsSatisfied: %w", err)
	}

	if met {
		return requirements, nil
	}

	if anonymize {
		return []*domain.ChallengeRequirement{}, nil
	}

	return nil, apperr.ErrChallengeNotFound
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

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if err := uc.deps.ChallengeRepo.AcquireRequirementsLock(ctx); err != nil {
			return fmt.Errorf("ChallengeUseCase - SetRequirements - AcquireRequirementsLock: %w", err)
		}

		if _, err := uc.deps.ChallengeRepo.GetByIDForUpdate(ctx, challengeID); err != nil {
			return fmt.Errorf("ChallengeUseCase - SetRequirements - ChallengeRepo.GetByIDForUpdate: %w", err)
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
	}); err != nil {
		return err
	}

	txctx.AfterCommitOrNow(ctx, func(context.Context) {
		uc.requirementsSf.Forget(challengeID.String() + ":req:pub")
	})
	uc.InvalidateChallengeListCache(ctx)

	return nil
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
