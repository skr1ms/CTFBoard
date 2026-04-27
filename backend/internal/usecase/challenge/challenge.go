package challenge

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/guard"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

// SolveRecordFn is the function type for recording a solve inside a transaction.
// Using a function value instead of a direct package import decouples the challenge
// usecase from the competition usecase package.
type SolveRecordFn func(ctx context.Context, solve *domain.Solve, challenge *domain.Challenge, challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository, decayFn ...scoring.DecayFunction) (int, error)

const (
	// challengeListCachePrefix is kept only for backward-compatible invalidation of old keys.
	challengeListCachePrefix = "challenges:list:"

	// Two-layer cache: shared base (challenges without per-team solve status) + lightweight per-team solved-ID set.
	challengeBaseCachePrefix   = "challenges:base:"
	challengeBaseTTL           = 30 * time.Second
	challengeSolvedCachePrefix = "challenges:solved:"
	challengeSolvedTTL         = 10 * time.Second

	// Frozen solve counts are immutable for the duration of the freeze; cache with a long TTL.
	frozenSolveCountsCachePrefix = "challenges:frozen_counts:"
	frozenSolveCountsTTL         = 10 * time.Minute
)

type ChallengeUseCase struct {
	deps              ChallengeDeps
	regexCache        *cachekit.LRFUCache[string, *regexp.Regexp]
	regexSf           singleflight.Group
	listBaseSF        singleflight.Group // deduplicates concurrent base-cache loader calls on cache miss
	challengeDetailSf singleflight.Group // for GetDetail (returns *usecase.ChallengeDetail)
	challengeFetchSf  singleflight.Group // for submitGetChallenge (returns *domain.Challenge)
	requirementsSf    singleflight.Group
}

type ChallengeDeps struct {
	ChallengeRepo   repo.ChallengeRepository
	TagRepo         repo.TagRepository
	SolveRepo       repo.SolveRepository
	SubmissionRepo  repo.SubmissionRepository
	FileRepo        repo.FileRepository
	Storage         storage.Provider
	HintUC          usecase.HintUseCase
	TM              repo.TransactionManager
	CompRepo        repo.CompetitionRepository
	CompUC          usecase.CompetitionUseCase
	CompParamUC     usecase.CompetitionParamUseCase
	TeamRepo        repo.TeamRepository
	UserRepo        repo.UserRepository
	ScoreboardCache cache.ScoreboardCacheInvalidator
	ListCache       *cachekit.Cache
	Broadcaster     solveBroadcaster
	AuditLogRepo    repo.AuditLogRepository
	Crypto          crypto.Service
	Logger          logkit.Logger
	SolveRecord     SolveRecordFn
}

var _ usecase.ChallengeUseCase = (*ChallengeUseCase)(nil)

type solveBroadcaster interface {
	NotifySolve(teamID uuid.UUID, challengeTitle string, points int, isFirstBlood bool)
}

func NewChallengeUseCase(deps ChallengeDeps) *ChallengeUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &ChallengeUseCase{
		deps:       deps,
		regexCache: cachekit.NewLRFUCache[string, *regexp.Regexp](cachekit.DefaultLRFUCacheSize),
	}
}

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
		return cachekit.GetOrLoad(uc.deps.ListCache, context.WithoutCancel(ctx), baseKey, challengeBaseTTL, func(context.Context) ([]*usecase.ChallengeWithTags, error) {
			return uc.getAllInner(context.WithoutCancel(ctx), nil, tagID)
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

	solvedIDs, err := cachekit.GetOrLoad(uc.deps.ListCache, ctx, solvedKey, challengeSolvedTTL, func(context.Context) ([]uuid.UUID, error) {
		if uc.deps.SolveRepo == nil {
			return nil, nil
		}

		return uc.deps.SolveRepo.GetSolvedChallengeIDsByTeam(ctx, *teamID, ids)
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - GetSolvedChallengeIDsByTeam: %w", err)
	}

	if len(solvedIDs) == 0 {
		return uc.applyFrozenSolveCounts(ctx, comp, base)
	}

	solvedSet := make(map[uuid.UUID]struct{}, len(solvedIDs))
	for _, id := range solvedIDs {
		solvedSet[id] = struct{}{}
	}

	out := make([]*usecase.ChallengeWithTags, len(base))
	for i, c := range base {
		_, solved := solvedSet[c.Challenge.ID]
		if solved == c.Solved {
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

		cached, cacheErr := cachekit.GetOrLoad(uc.deps.ListCache, ctx, cacheKey, frozenSolveCountsTTL, func(context.Context) (map[uuid.UUID]int, error) {
			return uc.deps.SolveRepo.GetSolveCounts(context.WithoutCancel(ctx), ft)
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
			Tags: cwt.Tags,
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
func (uc *ChallengeUseCase) computeRequirementsMetMap(ctx context.Context, teamID *uuid.UUID, challengeIDs []uuid.UUID) map[uuid.UUID]bool {
	if uc.deps.CompParamUC == nil {
		return nil
	}

	if !uc.deps.CompParamUC.GetBool(ctx, "challenge_prerequisite_anonymize", false) {
		return nil
	}

	pairs, err := uc.deps.ChallengeRepo.GetAllRequirementPairs(ctx)
	if err != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - computeRequirementsMetMap - GetAllRequirementPairs")

		return nil
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

			return nil
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

func (uc *ChallengeUseCase) shouldAnonymizePrerequisites(ctx context.Context) bool {
	if uc.deps.CompParamUC == nil {
		return false
	}

	return uc.deps.CompParamUC.GetBool(ctx, "challenge_prerequisite_anonymize", false)
}

// anonymizedChallengeDetail returns a placeholder ChallengeDetail for a challenge whose
// prerequisites are unmet and the anonymize flag is enabled. Title and description are
// replaced with "???", ConnectionInfo is cleared, and state is forced to Locked so the
// client shows a locked card rather than a 404.
func anonymizedChallengeDetail(c *domain.Challenge) *usecase.ChallengeDetail {
	hidden := "???"

	masked := *c
	masked.Title = hidden
	masked.Description = hidden
	masked.ConnectionInfo = ""
	masked.State = domain.ChallengeStateLocked

	return &usecase.ChallengeDetail{
		Challenge:  &masked,
		Tags:       []*domain.Tag{},
		Files:      []*domain.File{},
		Hints:      []*usecase.HintWithUnlockStatus{},
		SolvedByMe: false,
		SolveCount: c.SolveCount,
	}
}

func (uc *ChallengeUseCase) GetByID(ctx context.Context, challengeID uuid.UUID) (*domain.Challenge, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetByID - ChallengeRepo.GetByID: %w", err)
	}

	return challenge, nil
}

// GetDetail is the singleflight-wrapped entry point for challenge detail. The singleflight
// key includes both challengeID and teamID so concurrent requests for the same pair share
// one DB round-trip. context.WithoutCancel is used internally so one caller's
// cancellation does not abort the shared flight for other waiters.
func (uc *ChallengeUseCase) GetDetail(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*usecase.ChallengeDetail, error) {
	teamIDStr := ""

	if teamID != nil {
		teamIDStr = teamID.String()
	}

	key := fmt.Sprintf("challenge_detail:%s:%s", challengeID, teamIDStr)

	// Use WithoutCancel so that one caller's context cancellation does not cancel the
	// shared singleflight work for other waiters on the same key
	v, err, _ := uc.challengeDetailSf.Do(key, func() (any, error) {
		return uc.getDetailInner(context.WithoutCancel(ctx), challengeID, teamID)
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail: %w", err)
	}

	d, ok := v.(*usecase.ChallengeDetail)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail: unexpected type from singleflight")
	}

	return d, nil
}

// getDetailInner fetches a challenge's full detail, gating access by prerequisite
// satisfaction before spawning a fan-out errgroup that loads tags, challenge files,
// hints (with unlock status), first-blood entry, and the team's solved status in
// parallel. After the errgroup completes it resolves the freeze-aware solve count: if the
// competition freeze is active it queries solves up to the freeze timestamp instead of
// using the live counter stored on the challenge row. When prerequisites are unmet and the
// challenge_prerequisite_anonymize flag is enabled, it returns an anonymized placeholder
// instead of ErrChallengeNotFound so that the challenge list entry remains visible.
func (uc *ChallengeUseCase) getDetailInner(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*usecase.ChallengeDetail, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, err
	}

	reqs, err := uc.deps.ChallengeRepo.GetRequirements(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - GetRequirements: %w", err)
	}

	if len(reqs) > 0 {
		if teamID == nil || uc.deps.SolveRepo == nil {
			if uc.shouldAnonymizePrerequisites(ctx) {
				return anonymizedChallengeDetail(challenge), nil
			}

			return nil, apperr.ErrChallengeNotFound
		}

		met, err := requirementsMet(ctx, challengeID, *teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetDetail - requirementsMet: %w", err)
		}

		if !met {
			if uc.shouldAnonymizePrerequisites(ctx) {
				return anonymizedChallengeDetail(challenge), nil
			}

			return nil, apperr.ErrChallengeNotFound
		}
	}

	var (
		tags       []*domain.Tag
		files      []*domain.File
		hints      []*usecase.HintWithUnlockStatus
		firstBlood *domain.FirstBloodEntry
		solvedByMe bool
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error

		tags, err = uc.getChallengeTags(gCtx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - GetDetail - getChallengeTags: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		var err error

		files, err = uc.getChallengeFiles(gCtx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - GetDetail - getChallengeFiles: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		var err error

		hints, err = uc.getChallengeHints(gCtx, challengeID, teamID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - GetDetail - getChallengeHints: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		var err error

		firstBlood, err = uc.getChallengeFirstBlood(gCtx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - GetDetail - getChallengeFirstBlood: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		var err error

		solvedByMe, err = uc.checkChallengeSolved(gCtx, challengeID, teamID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - GetDetail - checkChallengeSolved: %w", err)
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - errgroup.Wait: %w", err)
	}

	solveCount := challenge.SolveCount

	if uc.deps.SolveRepo != nil {
		comp := computil.Cached(ctx, uc.deps.CompUC, uc.deps.CompRepo)

		if comp != nil && comp.IsFreezeActive() {
			frozenSolves, err := uc.deps.SolveRepo.GetByChallengeID(ctx, challengeID, comp.FreezeTime)
			if err == nil {
				solveCount = len(frozenSolves)
			}
		}
	}

	return &usecase.ChallengeDetail{
		Challenge:  challenge,
		Tags:       tags,
		Files:      files,
		Hints:      hints,
		FirstBlood: firstBlood,
		SolvedByMe: solvedByMe,
		SolveCount: solveCount,
	}, nil
}

func (uc *ChallengeUseCase) getChallengeTags(ctx context.Context, challengeID uuid.UUID) ([]*domain.Tag, error) {
	if uc.deps.TagRepo == nil {
		return nil, nil
	}

	tags, err := uc.deps.TagRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - TagRepo.GetByChallengeID: %w", err)
	}

	return tags, nil
}

func (uc *ChallengeUseCase) getChallengeFiles(ctx context.Context, challengeID uuid.UUID) ([]*domain.File, error) {
	if uc.deps.FileRepo == nil {
		return nil, nil
	}

	files, err := uc.deps.FileRepo.GetByChallengeID(ctx, challengeID, domain.FileTypeChallenge)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - FileRepo.GetByChallengeID: %w", err)
	}

	return files, nil
}

// getChallengeHints returns hints with per-team unlock status via HintUC.
// Returns nil (not an error) when HintUC is not wired.
func (uc *ChallengeUseCase) getChallengeHints(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) ([]*usecase.HintWithUnlockStatus, error) {
	if uc.deps.HintUC == nil {
		return nil, nil
	}

	hints, err := uc.deps.HintUC.GetByChallengeID(ctx, challengeID, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - HintRepo.GetByChallengeID: %w", err)
	}

	return hints, nil
}

// getChallengeFirstBlood returns the first-blood entry for a challenge with freeze
// awareness: when the competition freeze is active it queries solves up to FreezeTime
// via GetFirstBloodFrozen; otherwise uses the live GetFirstBlood. ErrSolveNotFound
// is treated as "no first blood yet" and returns nil without error.
func (uc *ChallengeUseCase) getChallengeFirstBlood(ctx context.Context, challengeID uuid.UUID) (*domain.FirstBloodEntry, error) {
	if uc.deps.SolveRepo == nil {
		return nil, nil
	}

	comp := computil.Cached(ctx, uc.deps.CompUC, uc.deps.CompRepo)

	var ft *time.Time

	if comp != nil && comp.IsFreezeActive() {
		ft = comp.FreezeTime
	}

	fb, err := uc.deps.SolveRepo.GetFirstBlood(ctx, challengeID, ft)
	if err != nil {
		if errors.Is(err, apperr.ErrSolveNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - SolveRepo.GetFirstBlood: %w", err)
	}

	return fb, nil
}

// checkChallengeSolved returns true when the team has an existing solve for the challenge.
// ErrSolveNotFound is mapped to (false, nil); any other error is propagated.
func (uc *ChallengeUseCase) checkChallengeSolved(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (bool, error) {
	if teamID == nil || uc.deps.SolveRepo == nil {
		return false, nil
	}

	_, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, *teamID, challengeID)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, apperr.ErrSolveNotFound) {
		return false, nil
	}

	return false, fmt.Errorf("ChallengeUseCase - GetDetail - checkSolved: %w", err)
}

// GetSolves returns the solve list for a challenge, applying freeze-time filtering when active.
func (uc *ChallengeUseCase) GetSolves(ctx context.Context, challengeID uuid.UUID) ([]*domain.SolveWithDetails, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolves - ChallengeRepo.GetByID: %w", err)
	}

	if err := guard.EnsureChallengeVisible(challenge); err != nil {
		return nil, err
	}

	comp, err := uc.deps.CompUC.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolves - CompUC.Get: %w", err)
	}

	var ft *time.Time

	if comp != nil && comp.IsFreezeActive() {
		ft = comp.FreezeTime
	}

	solves, err := uc.deps.SolveRepo.GetByChallengeID(ctx, challengeID, ft)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolves - SolveRepo.GetByChallengeID: %w", err)
	}

	return solves, nil
}

func validateDynamicScoringRange(initialValue, minValue int) error {
	if initialValue < 0 {
		return apperr.NewValidationErrorf("dynamic scoring initial value must be non-negative")
	}

	if minValue < 0 {
		return apperr.NewValidationErrorf("dynamic scoring min value must be non-negative")
	}

	if initialValue > 0 && minValue > 0 && initialValue < minValue {
		return apperr.ErrInvalidScoringRange
	}

	return nil
}

const maxFlagFormatRegexLen = 1024

func validateFlagFormatRegex(flagFormatRegex *string) error {
	if flagFormatRegex == nil || *flagFormatRegex == "" {
		return nil
	}

	if len(*flagFormatRegex) > maxFlagFormatRegexLen {
		return apperr.ErrInvalidFlagFormat
	}

	if _, err := regexp.Compile(*flagFormatRegex); err != nil {
		return apperr.ErrInvalidFlagFormat
	}

	return nil
}

// Create creates a new challenge, hashing or AES-encrypting the flag, persisting it with tags,
// and invalidating the challenge list cache.
func (uc *ChallengeUseCase) Create(ctx context.Context, title, description, category string, points, initialValue, minValue, decay int, flag, connectionInfo string, maxAttempts int, maxAttemptsWindow time.Duration, position int, state string, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*domain.Challenge, error) {
	if err := validateDynamicScoringRange(initialValue, minValue); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - validateDynamicScoringRange: %w", err)
	}

	flagHash, flagRegex, err := uc.challengeCreateComputeFlagHash(flag, isRegex, isCaseInsensitive)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - challengeCreateComputeFlagHash: %w", err)
	}

	var flagRegexPtr *string

	if flagRegex != "" {
		flagRegexPtr = &flagRegex
	}

	challenge := &domain.Challenge{
		Title:             title,
		Description:       description,
		Category:          category,
		Points:            points,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		SolveCount:        0,
		FlagHash:          flagHash,
		ConnectionInfo:    connectionInfo,
		MaxAttempts:       maxAttempts,
		MaxAttemptsWindow: maxAttemptsWindow,
		Position:          position,
		State:             domain.ChallengeStateOrDefault(state),
		IsRegex:           isRegex,
		IsCaseInsensitive: isCaseInsensitive,
		FlagRegex:         flagRegexPtr,
		FlagFormatRegex:   flagFormatRegex,
	}
	if err := validateFlagFormatRegex(flagFormatRegex); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - validateFlagFormatRegex: %w", err)
	}

	if err := uc.challengeCreatePersist(ctx, challenge, tagIDs); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - challengeCreatePersist: %w", err)
	}

	uc.InvalidateChallengeListCache(ctx)

	return challenge, nil
}

// challengeCreateComputeFlagHash derives the stored flag representation from the plain-text flag.
// For regex challenges: AES-encrypts the pattern and sets FlagHash to the sentinel value.
// For plain challenges: SHA-256 hashes the (optionally lowercased) flag.
func (uc *ChallengeUseCase) challengeCreateComputeFlagHash(flag string, isRegex, isCaseInsensitive bool) (flagHash, flagRegex string, err error) {
	if isRegex {
		if uc.deps.Crypto == nil {
			return "", "", fmt.Errorf("ChallengeUseCase - Create - crypto.ErrServiceNotConfigured: %w", crypto.ErrServiceNotConfigured)
		}

		encrypted, err := uc.deps.Crypto.Encrypt(flag)
		if err != nil {
			return "", "", fmt.Errorf("ChallengeUseCase - Create - crypto.Encrypt: %w", err)
		}

		return domain.FlagHashRegexSentinel, encrypted, nil
	}

	userInput := crypto.NormalizeFlagInput(strings.TrimSpace(flag))

	if isCaseInsensitive {
		userInput = strings.ToLower(userInput)
	}

	return crypto.SHA256Hex(userInput), "", nil
}

func (uc *ChallengeUseCase) challengeCreatePersist(ctx context.Context, challenge *domain.Challenge, tagIDs []uuid.UUID) error {
	return uc.challengeCreatePersistTx(ctx, challenge, tagIDs)
}

// challengeCreatePersistTx persists a new challenge inside a transaction: generates a
// UUID, creates the row, and optionally attaches tags via SetTags. Requirements can be
// added after the challenge exists because SetRequirements is a separate call by the
// admin handler.
func (uc *ChallengeUseCase) challengeCreatePersistTx(ctx context.Context, challenge *domain.Challenge, tagIDs []uuid.UUID) error {
	return uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		challenge.ID = uuid.New()

		err := uc.deps.ChallengeRepo.Create(ctx, challenge)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - Create - ChallengeRepo.Create: %w", err)
		}

		if len(tagIDs) > 0 {
			err := uc.deps.ChallengeRepo.SetTags(ctx, challenge.ID, tagIDs)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - Create - ChallengeRepo.SetTags: %w", err)
			}
		}

		return nil
	})
}

func (uc *ChallengeUseCase) Update(ctx context.Context, ID uuid.UUID, title, description, category string, points int, initialValue, minValue, decay *int, flag string, connectionInfo *string, maxAttempts *int, maxAttemptsWindow *time.Duration, position *int, state string, isRegex, isCaseInsensitive *bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*domain.Challenge, error) {
	if err := validateFlagFormatRegex(flagFormatRegex); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update - validateFlagFormatRegex: %w", err)
	}

	challenge, err := uc.challengeUpdatePersist(ctx, ID, title, description, category, points, initialValue, minValue, decay, flag, connectionInfo, maxAttempts, maxAttemptsWindow, position, state, isRegex, isCaseInsensitive, flagFormatRegex, tagIDs)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update - challengeUpdatePersist: %w", err)
	}

	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	}

	uc.InvalidateChallengeListCache(ctx)

	return challenge, nil
}

// challengeUpdatePersist applies an update to a challenge inside a single transaction
// It reads the current row with GetByID, computes the effective initialValue/minValue
// after merging nil-able overrides, and validates the dynamic scoring range. Flag handling
// covers three cases: (a) no flag change - checks for a mode switch (plain↔regex) and
// returns an error if a new flag value was not provided; (b) switching to regex - encrypts
// the new pattern and stores the sentinel hash; (c) switching to plain - hashes the new
// value (lowercased if case-insensitive) and clears the regex column. Tags are replaced
// atomically via SetTags in the same transaction.
func (uc *ChallengeUseCase) challengeUpdatePersist(ctx context.Context, ID uuid.UUID, title, description, category string, points int, initialValue, minValue, decay *int, flag string, connectionInfo *string, maxAttempts *int, maxAttemptsWindow *time.Duration, position *int, state string, isRegex, isCaseInsensitive *bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*domain.Challenge, error) {
	var challenge *domain.Challenge

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var errTx error

		challenge, errTx = uc.deps.ChallengeRepo.GetByID(ctx, ID)
		if errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - ChallengeRepo.GetByID: %w", errTx)
		}

		effectiveIV, effectiveMV := challenge.InitialValue, challenge.MinValue

		if initialValue != nil {
			effectiveIV = *initialValue
		}

		if minValue != nil {
			effectiveMV = *minValue
		}

		if errTx = validateDynamicScoringRange(effectiveIV, effectiveMV); errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - validateDynamicScoringRange: %w", errTx)
		}

		uc.challengeUpdateApplyBasic(challenge, title, description, category, points, initialValue, minValue, decay, connectionInfo, maxAttempts, maxAttemptsWindow, position, state, isRegex, isCaseInsensitive, flagFormatRegex)
		applyRegex, applyCaseInsensitive := challenge.IsRegex, challenge.IsCaseInsensitive

		if isRegex != nil {
			applyRegex = *isRegex
		}

		if isCaseInsensitive != nil {
			applyCaseInsensitive = *isCaseInsensitive
		}

		if errTx = uc.challengeUpdateApplyFlag(challenge, flag, applyRegex, applyCaseInsensitive); errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - challengeUpdateApplyFlag: %w", errTx)
		}

		if errTx = uc.deps.ChallengeRepo.Update(ctx, challenge); errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - ChallengeRepo.Update: %w", errTx)
		}

		if errTx = uc.deps.ChallengeRepo.SetTags(ctx, ID, tagIDs); errTx != nil {
			return fmt.Errorf("ChallengeUseCase - Update - ChallengeRepo.SetTags: %w", errTx)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - challengeUpdatePersist - TM.Run: %w", err)
	}

	return challenge, nil
}

// challengeUpdateApplyBasic applies all non-flag scalar fields to the challenge struct.
// Pointer parameters are applied only when non-nil (partial-update semantics); string
// and int parameters that are always present overwrite unconditionally.
func (uc *ChallengeUseCase) challengeUpdateApplyBasic(c *domain.Challenge, title, description, category string, points int, initialValue, minValue, decay *int, connectionInfo *string, maxAttempts *int, maxAttemptsWindow *time.Duration, position *int, state string, isRegex, isCaseInsensitive *bool, flagFormatRegex *string) {
	c.Title = title
	c.Description = description
	c.Category = category
	c.Points = points

	if connectionInfo != nil {
		c.ConnectionInfo = *connectionInfo
	}

	if maxAttempts != nil {
		c.MaxAttempts = *maxAttempts
	}

	if maxAttemptsWindow != nil {
		c.MaxAttemptsWindow = *maxAttemptsWindow
	}

	if position != nil {
		c.Position = *position
	}

	if state != "" {
		c.State = domain.ChallengeStateOrDefault(state)
	}

	if initialValue != nil {
		c.InitialValue = *initialValue
	}

	if minValue != nil {
		c.MinValue = *minValue
	}

	if decay != nil {
		c.Decay = *decay
	}

	if isRegex != nil {
		c.IsRegex = *isRegex
	}

	if isCaseInsensitive != nil {
		c.IsCaseInsensitive = *isCaseInsensitive
	}

	c.FlagFormatRegex = flagFormatRegex
}

// challengeUpdateApplyFlag applies a flag change during challenge update.
// Enforces that a new flag value is provided when switching between regex and hash modes.
// For regex: AES-encrypts the new pattern. For plain: SHA-256 hashes it.
func (uc *ChallengeUseCase) challengeUpdateApplyFlag(c *domain.Challenge, flag string, isRegex, isCaseInsensitive bool) error {
	if flag == "" {
		wasRegex := c.FlagHash == domain.FlagHashRegexSentinel
		if isRegex && !wasRegex {
			return apperr.ErrChallengeFlagRequiredWhenSwitchingMode
		}

		if !isRegex && wasRegex {
			return apperr.ErrChallengeFlagRequiredWhenSwitchingMode
		}

		return nil
	}

	if isRegex {
		if uc.deps.Crypto == nil {
			return fmt.Errorf("ChallengeUseCase - Update - crypto.ErrServiceNotConfigured: %w", crypto.ErrServiceNotConfigured)
		}

		encrypted, err := uc.deps.Crypto.Encrypt(flag)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - Update - crypto.Encrypt: %w", err)
		}

		c.FlagRegex = &encrypted
		c.FlagHash = domain.FlagHashRegexSentinel

		return nil
	}

	userInput := crypto.NormalizeFlagInput(strings.TrimSpace(flag))

	if isCaseInsensitive {
		userInput = strings.ToLower(userInput)
	}

	c.FlagHash = crypto.SHA256Hex(userInput)
	c.FlagRegex = nil

	return nil
}

// Delete removes a challenge and all of its associated data in a single transaction, then
// cleans up object-storage files after the commit. Inside the transaction it collects all
// file storage locations, deletes the challenge row (which cascades to solves, submissions,
// hints, unlocks, tags, requirements, and solution rows via foreign-key constraints), and
// writes an audit-log entry. Storage deletion happens outside the transaction via
// deleteStorageFiles so that a storage failure does not roll back the DB deletion; the
// call uses context.WithoutCancel so that an HTTP request cancellation does not abort the
// best-effort cleanup. Scoreboard and challenge-list caches are invalidated after the
// commit.
func (uc *ChallengeUseCase) Delete(ctx context.Context, ID, actorID uuid.UUID, clientIP string) error {
	var fileLocations []string

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.ChallengeRepo.GetByID(ctx, ID); err != nil {
			return fmt.Errorf("ChallengeUseCase - Delete - ChallengeRepo.GetByID: %w", err)
		}

		fileLocations = uc.collectFileLocations(ctx, ID)

		err := uc.deps.ChallengeRepo.Delete(ctx, ID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - Delete - ChallengeRepo.Delete: %w", err)
		}

		auditLog := &domain.AuditLog{
			UserID:     &actorID,
			Action:     domain.AuditActionDelete,
			EntityType: domain.AuditEntityChallenge,
			EntityID:   ID.String(),
			IP:         clientIP,
		}

		err = uc.deps.AuditLogRepo.Create(ctx, auditLog)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - Delete - AuditLogRepo.Create: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - Delete - TM.Run: %w", err)
	}

	uc.deleteStorageFiles(context.WithoutCancel(ctx), fileLocations)

	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	}

	uc.InvalidateChallengeListCache(ctx)

	return nil
}

// collectFileLocations fetches all file rows for a challenge and extracts their storage
// location strings. Errors are logged and swallowed so that a missing file record does
// not prevent challenge deletion; the caller handles storage cleanup best-effort.
func (uc *ChallengeUseCase) collectFileLocations(ctx context.Context, challengeID uuid.UUID) []string {
	if uc.deps.FileRepo == nil || uc.deps.Storage == nil {
		return nil
	}

	files, err := uc.deps.FileRepo.GetAllByChallengeID(ctx, challengeID)
	if err != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - collectFileLocations - GetAllByChallengeID")

		return nil
	}

	locations := make([]string, 0, len(files))
	for _, f := range files {
		locations = append(locations, f.Location)
	}

	return locations
}

// deleteStorageFiles removes files from storage. Failures are logged only and not returned
// the caller is not notified so this is best-effort cleanup (e.g. when deleting a challenge).
func (uc *ChallengeUseCase) deleteStorageFiles(ctx context.Context, locations []string) {
	if uc.deps.Storage == nil || len(locations) == 0 {
		return
	}

	const maxConcurrent = 4

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrent)

	for _, loc := range locations {
		g.Go(func() error {
			if err := uc.deps.Storage.Delete(ctx, loc); err != nil {
				uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - deleteStorageFiles", logkit.Fields{"location": loc})
			}

			return nil
		})
	}

	_ = g.Wait()
}

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
		challenge, err := uc.deps.ChallengeRepo.GetByID(context.WithoutCancel(ctx), challengeID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetRequirements - ChallengeRepo.GetByID: %w", err)
		}

		if err := guard.EnsureChallengeVisible(challenge); err != nil {
			return nil, err
		}

		return uc.deps.ChallengeRepo.GetRequirements(context.WithoutCancel(ctx), challengeID)
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
// proposed set would not introduce a directed cycle in the prerequisite graph. It first
// fetches all existing requirement pairs to reconstruct the current adjacency map, removes
// the old edges for challengeID, inserts the new ones, and then runs requirementsContainCycle
// from challengeID. Only if no cycle is detected does it open a transaction to persist the
// change via SetRequirements on the repo.
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

	return uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
			return fmt.Errorf("ChallengeUseCase - SetRequirements - ChallengeRepo.GetByID: %w", err)
		}

		err := uc.deps.ChallengeRepo.SetRequirements(ctx, challengeID, requirementIDs)
		if err != nil {
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
