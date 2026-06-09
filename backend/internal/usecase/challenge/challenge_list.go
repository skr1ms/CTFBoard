package challenge

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
)

// GetAll returns all challenges with per-team solve status, optionally filtered by tagID
// It uses a two-layer cache to reduce cache cardinality
// 1. A shared base cache keyed by tagID only (challenges without team-specific data, TTL 30s)
// 2. A per-team solved-ID cache (TTL 10s) overlaid on the base to set Solved flags
// This reduces cache cardinality from O(teams) to O(1) for the expensive query, while keeping
// per-team solve status accurate.
func (uc *ChallengeUseCase) GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*usecase.ChallengeWithTags, error) {
	comp := computil.Cached(ctx, uc.deps.CompUC, uc.deps.CompRepo)
	if uc.deps.ListCache == nil {
		list, err := uc.getAllInner(ctx, teamID, tagID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetAll - getAllInner: %w", err)
		}

		return uc.applyFrozenSolveCounts(ctx, comp, list)
	}

	baseKey := challengeBaseCacheKey(tagID)

	sfVal, sfErr, _ := uc.listBaseSF.Do(baseKey, func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		return cachekit.GetOrLoad(uc.deps.ListCache, loadCtx, baseKey, challengeBaseTTL, func(loadCtx context.Context) ([]*usecase.ChallengeWithTags, error) {
			return uc.getAllInner(loadCtx, nil, tagID)
		})
	})
	if sfErr != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - cache.GetOrLoad: %w", sfErr)
	}

	base, ok := sfVal.([]*usecase.ChallengeWithTags)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - listBaseSF: unexpected type")
	}

	if teamID == nil {
		return uc.applyFrozenSolveCounts(ctx, comp, base)
	}

	solvedKey := challengeSolvedCachePrefix + teamID.String()

	ids := make([]uuid.UUID, len(base))
	for i, c := range base {
		ids[i] = c.Challenge.ID
	}

	solvedLoadCtx, solvedCancel := cacheutil.LoaderContext(ctx)
	defer solvedCancel()

	solvedIDs, err := cachekit.GetOrLoad(uc.deps.ListCache, solvedLoadCtx, solvedKey, challengeSolvedTTL, func(loadCtx context.Context) ([]uuid.UUID, error) {
		if uc.deps.SolveRepo == nil {
			return nil, nil
		}

		return uc.deps.SolveRepo.GetSolvedChallengeIDsByTeam(loadCtx, *teamID, ids)
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - GetSolvedChallengeIDsByTeam: %w", err)
	}

	solvedSet := make(map[uuid.UUID]struct{}, len(solvedIDs))
	for _, id := range solvedIDs {
		solvedSet[id] = struct{}{}
	}

	reqMetMap := uc.computeRequirementsMetMap(ctx, teamID, ids)

	if len(solvedSet) == 0 && len(reqMetMap) == 0 {
		return uc.applyFrozenSolveCounts(ctx, comp, base)
	}

	out := make([]*usecase.ChallengeWithTags, len(base))
	for i, c := range base {
		_, solved := solvedSet[c.Challenge.ID]

		reqMet, hasReqMet := reqMetMap[c.Challenge.ID]
		reqMetChanged := hasReqMet && (c.RequirementsMet == nil || *c.RequirementsMet != reqMet)

		if solved == c.Solved && !reqMetChanged {
			out[i] = c

			continue
		}

		copied := &usecase.ChallengeWithTags{
			ChallengeWithSolved: &domain.ChallengeWithSolved{
				Challenge: c.Challenge,
				Solved:    solved,
			},
			Tags: c.Tags,
		}

		if hasReqMet {
			copied.RequirementsMet = &reqMet
		} else {
			copied.RequirementsMet = c.RequirementsMet
		}

		out[i] = copied
	}

	return uc.applyFrozenSolveCounts(ctx, comp, out)
}

// applyFrozenSolveCounts replaces the live solve counts in the challenge list with the snapshot
// counts taken at freeze time. Returns the original list unchanged when freeze is not active.
// The frozen counts are cached for frozenSolveCountsTTL because they are immutable during freeze.
func (uc *ChallengeUseCase) applyFrozenSolveCounts(ctx context.Context, comp *domain.Competition, list []*usecase.ChallengeWithTags) ([]*usecase.ChallengeWithTags, error) {
	if comp == nil || !comp.IsFreezeActive() || comp.FreezeTime == nil || uc.deps.SolveRepo == nil {
		return list, nil
	}

	ft := comp.FreezeTime

	var frozenCounts map[uuid.UUID]int

	if uc.deps.ListCache != nil {
		cacheKey := frozenSolveCountsCachePrefix + fmt.Sprintf("%d", ft.Unix())

		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		cached, cacheErr := cachekit.GetOrLoad(uc.deps.ListCache, loadCtx, cacheKey, frozenSolveCountsTTL, func(loadCtx context.Context) (map[uuid.UUID]int, error) {
			return uc.deps.SolveRepo.GetSolveCounts(loadCtx, ft)
		})
		if cacheErr != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetAll - GetSolveCounts: %w", cacheErr)
		}

		frozenCounts = cached
	} else {
		var err error

		frozenCounts, err = uc.deps.SolveRepo.GetSolveCounts(ctx, ft)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetAll - GetSolveCounts: %w", err)
		}
	}

	result := make([]*usecase.ChallengeWithTags, len(list))
	for i, cwt := range list {
		count := frozenCounts[cwt.Challenge.ID]
		chCopy := *cwt.Challenge
		chCopy.SolveCount = count
		result[i] = &usecase.ChallengeWithTags{
			ChallengeWithSolved: &domain.ChallengeWithSolved{
				Challenge: &chCopy,
				Solved:    cwt.Solved,
			},
			Tags:            cwt.Tags,
			RequirementsMet: cwt.RequirementsMet,
		}
	}

	return result, nil
}

func challengeBaseCacheKey(tagID *uuid.UUID) string {
	if tagID != nil {
		return challengeBaseCachePrefix + tagID.String()
	}

	return challengeBaseCachePrefix
}

// getAllInner fetches challenges from the DB, batch-loads their tags via TagRepo, and
// evaluates the requirements-met map so the caller can apply anonymization. It is the
// single source of truth for the challenge list; GetAll wraps it with caching.
func (uc *ChallengeUseCase) getAllInner(ctx context.Context, teamID, tagID *uuid.UUID) ([]*usecase.ChallengeWithTags, error) {
	challenges, err := uc.deps.ChallengeRepo.GetAll(ctx, teamID, tagID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - ChallengeRepo.GetAll: %w", err)
	}

	ids := make([]uuid.UUID, len(challenges))
	for i, c := range challenges {
		ids[i] = c.Challenge.ID
	}

	var tagsMap map[uuid.UUID][]*domain.Tag

	if uc.deps.TagRepo != nil {
		tagsMap, err = uc.deps.TagRepo.GetByChallengeIDs(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetAll - TagRepo.GetByChallengeIDs: %w", err)
		}
	}

	reqMetMap := uc.computeRequirementsMetMap(ctx, teamID, ids)

	out := make([]*usecase.ChallengeWithTags, len(challenges))
	for i, c := range challenges {
		tags := tagsMap[c.Challenge.ID]
		if tags == nil {
			tags = []*domain.Tag{}
		}

		cwt := &usecase.ChallengeWithTags{
			ChallengeWithSolved: c,
			Tags:                tags,
		}

		if met, ok := reqMetMap[c.Challenge.ID]; ok {
			cwt.RequirementsMet = &met
		}

		out[i] = cwt
	}

	return out, nil
}

// computeRequirementsMetMap batch-loads all requirement pairs and, when a team is
// present, the team's existing solves for those requirements. It builds an adjacency map
// (challengeID -> []requiredChallengeID) from the full requirement graph, then evaluates
// each challenge in challengeIDs: a challenge is "met" when every required challenge has
// been solved by the team. Challenges with no requirements are omitted from the result map
// (callers treat a missing key as "no requirements"). Returns nil when the
// challenge_prerequisite_anonymize feature flag is off or when no requirements exist.
// Requirement-loading failures fail closed so a transient repository error cannot
// expose gated challenge metadata as if the challenge had no prerequisites.
func (uc *ChallengeUseCase) computeRequirementsMetMap(ctx context.Context, teamID *uuid.UUID, challengeIDs []uuid.UUID) map[uuid.UUID]bool {
	if uc.deps.CompParamUC == nil {
		return nil
	}

	if !uc.deps.CompParamUC.GetBool(ctx, "challenge_prerequisite_anonymize", false) {
		return nil
	}

	pairs, err := uc.getAllRequirementPairsCached(ctx)
	if err != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - computeRequirementsMetMap - GetAllRequirementPairs")

		return requirementsUnknownMap(challengeIDs)
	}

	if len(pairs) == 0 {
		return nil
	}

	reqsByCh := make(map[uuid.UUID][]uuid.UUID)

	for _, p := range pairs {
		reqsByCh[p.ChallengeID] = append(reqsByCh[p.ChallengeID], p.RequiredChallengeID)
	}

	idSet := make(map[uuid.UUID]struct{}, len(challengeIDs))
	for _, id := range challengeIDs {
		idSet[id] = struct{}{}
	}

	hasReqs := false

	for id := range idSet {
		if _, ok := reqsByCh[id]; ok {
			hasReqs = true

			break
		}
	}

	if !hasReqs {
		return nil
	}

	var solvedSet map[uuid.UUID]struct{}

	if teamID != nil && uc.deps.SolveRepo != nil {
		allReqIDs := make(map[uuid.UUID]struct{})

		for id := range idSet {
			for _, reqID := range reqsByCh[id] {
				allReqIDs[reqID] = struct{}{}
			}
		}

		reqIDSlice := make([]uuid.UUID, 0, len(allReqIDs))
		for id := range allReqIDs {
			reqIDSlice = append(reqIDSlice, id)
		}

		solvedIDs, err := uc.deps.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, *teamID, reqIDSlice)
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - computeRequirementsMetMap - GetSolvedChallengeIDsByTeam")

			return requirementsUnmetMap(challengeIDs, reqsByCh)
		}

		solvedSet = make(map[uuid.UUID]struct{}, len(solvedIDs))
		for _, id := range solvedIDs {
			solvedSet[id] = struct{}{}
		}
	}

	result := make(map[uuid.UUID]bool, len(challengeIDs))
	for _, id := range challengeIDs {
		reqs, ok := reqsByCh[id]
		if !ok {
			continue
		}

		met := true

		for _, reqID := range reqs {
			if _, solved := solvedSet[reqID]; !solved {
				met = false

				break
			}
		}

		result[id] = met
	}

	return result
}

func (uc *ChallengeUseCase) getAllRequirementPairsCached(ctx context.Context) ([]*domain.ChallengeRequirementPair, error) {
	if uc.deps.ListCache == nil {
		return uc.deps.ChallengeRepo.GetAllRequirementPairs(ctx)
	}

	v, err, _ := uc.requirementPairsSF.Do(requirementPairsCacheKey, func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		return cachekit.GetOrLoad(uc.deps.ListCache, loadCtx, requirementPairsCacheKey, requirementPairsTTL, func(loadCtx context.Context) ([]*domain.ChallengeRequirementPair, error) {
			return uc.deps.ChallengeRepo.GetAllRequirementPairs(loadCtx)
		})
	})
	if err != nil {
		return nil, err
	}

	pairs, ok := v.([]*domain.ChallengeRequirementPair)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - getAllRequirementPairsCached: unexpected type")
	}

	return pairs, nil
}

func requirementsUnknownMap(challengeIDs []uuid.UUID) map[uuid.UUID]bool {
	result := make(map[uuid.UUID]bool, len(challengeIDs))

	for _, id := range challengeIDs {
		result[id] = false
	}

	return result
}

func requirementsUnmetMap(challengeIDs []uuid.UUID, reqsByCh map[uuid.UUID][]uuid.UUID) map[uuid.UUID]bool {
	result := make(map[uuid.UUID]bool)

	for _, id := range challengeIDs {
		if _, ok := reqsByCh[id]; ok {
			result[id] = false
		}
	}

	return result
}
