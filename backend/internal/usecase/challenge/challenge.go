package challenge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
)

const (
	// challengeListCachePrefix is kept only for backward-compatible invalidation of old keys.
	challengeListCachePrefix = "challenges:list:"

	// Two-layer cache: shared base (challenges without per-team solve status) + lightweight per-team solved-ID set.
	challengeBaseCachePrefix   = "challenges:base:"
	challengeBaseTTL           = 30 * time.Second
	challengeSolvedCachePrefix = "challenges:solved:"
	challengeSolvedTTL         = 10 * time.Second
)

type ChallengeUseCase struct {
	deps              ChallengeDeps
	regexCache        *cache.BoundedCache[string, *regexp.Regexp]
	regexSf           singleflight.Group
	challengeDetailSf singleflight.Group // for GetDetail (returns *usecase.ChallengeDetail)
	challengeFetchSf  singleflight.Group // for submitGetChallenge (returns *entity.Challenge)
	requirementsSf    singleflight.Group
}

type ChallengeDeps struct {
	ChallengeRepo   repo.ChallengeRepository
	TagRepo         repo.TagRepository
	SolveRepo       repo.SolveRepository
	FileRepo        repo.FileRepository
	Storage         storage.Provider
	HintUC          usecase.HintUseCase
	TM              repo.TransactionManager
	CompRepo        repo.CompetitionRepository
	CompUC          usecase.CompetitionUseCase
	TeamRepo        repo.TeamRepository
	UserRepo        repo.UserRepository
	ScoreboardCache cache.ScoreboardCacheInvalidator
	ListCache       *cache.Cache
	Broadcaster     websocket.SolveBroadcaster
	AuditLogRepo    repo.AuditLogRepository
	Crypto          crypto.Service
	Logger          logger.Logger
}

var _ usecase.ChallengeUseCase = (*ChallengeUseCase)(nil)

func NewChallengeUseCase(deps ChallengeDeps) *ChallengeUseCase {
	if deps.Logger == nil {
		deps.Logger = logger.Noop()
	}
	return &ChallengeUseCase{
		deps:       deps,
		regexCache: cache.NewBoundedCache[string, *regexp.Regexp](cache.DefaultBoundedCacheSize),
	}
}

// GetAll returns all challenges with per-team solve status. Uses a two-layer cache:
// 1. A shared base cache keyed by tagID only (challenges without team-specific data, TTL 30s).
// 2. A per-team solved-ID cache (TTL 10s) overlaid on the base to set Solved flags.
// This reduces cache cardinality from O(teams) to O(1) for the expensive query, while keeping
// per-team solve status accurate.
func (uc *ChallengeUseCase) GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*usecase.ChallengeWithTags, error) {
	comp := uc.getCompetitionForGetAll(ctx)
	if uc.deps.ListCache == nil {
		list, err := uc.getAllInner(ctx, teamID, tagID)
		if err != nil {
			return nil, err
		}
		return uc.applyFrozenSolveCounts(ctx, comp, list)
	}

	baseKey := challengeBaseCacheKey(tagID)
	base, err := cache.GetOrLoad(uc.deps.ListCache, ctx, baseKey, challengeBaseTTL, func() ([]*usecase.ChallengeWithTags, error) {
		return uc.getAllInner(ctx, nil, tagID)
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - cache.GetOrLoad: %w", err)
	}

	if teamID == nil {
		return uc.applyFrozenSolveCounts(ctx, comp, base)
	}

	solvedKey := challengeSolvedCachePrefix + teamID.String()
	ids := make([]uuid.UUID, len(base))
	for i, c := range base {
		ids[i] = c.Challenge.ID
	}
	solvedIDs, err := cache.GetOrLoad(uc.deps.ListCache, ctx, solvedKey, challengeSolvedTTL, func() ([]uuid.UUID, error) {
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
			ChallengeWithSolved: &entity.ChallengeWithSolved{
				Challenge: c.Challenge,
				Solved:    solved,
			},
			Tags: c.Tags,
		}
		out[i] = copied
	}
	return uc.applyFrozenSolveCounts(ctx, comp, out)
}

func (uc *ChallengeUseCase) getCompetitionForGetAll(ctx context.Context) *entity.Competition {
	if uc.deps.CompUC != nil {
		comp, err := uc.deps.CompUC.Get(ctx)
		if err != nil {
			return nil
		}
		return comp
	}
	if uc.deps.CompRepo != nil {
		comp, err := uc.deps.CompRepo.Get(ctx)
		if err != nil {
			return nil
		}
		return comp
	}
	return nil
}

func (uc *ChallengeUseCase) applyFrozenSolveCounts(ctx context.Context, comp *entity.Competition, list []*usecase.ChallengeWithTags) ([]*usecase.ChallengeWithTags, error) {
	if comp == nil || !comp.IsFreezeActive() || comp.FreezeTime == nil || uc.deps.SolveRepo == nil {
		return list, nil
	}
	frozenCounts, err := uc.deps.SolveRepo.GetSolveCountsFrozen(ctx, *comp.FreezeTime)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - GetSolveCountsFrozen: %w", err)
	}
	result := make([]*usecase.ChallengeWithTags, len(list))
	for i, cwt := range list {
		count := frozenCounts[cwt.Challenge.ID]
		chCopy := *cwt.Challenge
		chCopy.SolveCount = count
		result[i] = &usecase.ChallengeWithTags{
			ChallengeWithSolved: &entity.ChallengeWithSolved{
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

func (uc *ChallengeUseCase) getAllInner(ctx context.Context, teamID, tagID *uuid.UUID) ([]*usecase.ChallengeWithTags, error) {
	challenges, err := uc.deps.ChallengeRepo.GetAll(ctx, teamID, tagID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - ChallengeRepo.GetAll: %w", err)
	}
	if uc.deps.TagRepo == nil {
		out := make([]*usecase.ChallengeWithTags, len(challenges))
		for i, c := range challenges {
			out[i] = &usecase.ChallengeWithTags{
				ChallengeWithSolved: c,
				Tags:                []*entity.Tag{},
			}
		}
		return out, nil
	}
	ids := make([]uuid.UUID, len(challenges))
	for i, c := range challenges {
		ids[i] = c.Challenge.ID
	}
	tagsMap, err := uc.deps.TagRepo.GetByChallengeIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - TagRepo.GetByChallengeIDs: %w", err)
	}
	out := make([]*usecase.ChallengeWithTags, len(challenges))
	for i, c := range challenges {
		tags := tagsMap[c.Challenge.ID]
		if tags == nil {
			tags = []*entity.Tag{}
		}
		out[i] = &usecase.ChallengeWithTags{
			ChallengeWithSolved: c,
			Tags:                tags,
		}
	}
	return out, nil
}

func (uc *ChallengeUseCase) GetByID(ctx context.Context, challengeID uuid.UUID) (*entity.Challenge, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetByID - ChallengeRepo.GetByID: %w", err)
	}
	return challenge, nil
}

func (uc *ChallengeUseCase) GetDetail(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*usecase.ChallengeDetail, error) {
	teamIDStr := ""
	if teamID != nil {
		teamIDStr = teamID.String()
	}
	key := fmt.Sprintf("challenge_detail:%s:%s", challengeID, teamIDStr)

	// Use WithoutCancel so that one caller's context cancellation does not cancel the
	// shared singleflight work for other waiters on the same key.
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

func (uc *ChallengeUseCase) getDetailInner(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*usecase.ChallengeDetail, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - ChallengeRepo.GetByID: %w", err)
	}
	if challenge.IsHidden {
		return nil, httperr.ErrChallengeNotFound
	}
	reqs, err := uc.deps.ChallengeRepo.GetRequirements(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - GetRequirements: %w", err)
	}
	if len(reqs) > 0 {
		if teamID == nil || uc.deps.SolveRepo == nil {
			return nil, httperr.ErrChallengeNotFound
		}
		met, err := requirementsMet(ctx, challengeID, *teamID, uc.deps.ChallengeRepo, uc.deps.SolveRepo)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetDetail - requirementsMet: %w", err)
		}
		if !met {
			return nil, httperr.ErrChallengeNotFound
		}
	}

	var (
		tags       []*entity.Tag
		files      []*entity.File
		hints      []*usecase.HintWithUnlockStatus
		firstBlood *entity.FirstBloodEntry
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
		var comp *entity.Competition
		if uc.deps.CompUC != nil {
			c, err := uc.deps.CompUC.Get(ctx)
			if err != nil {
				uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - GetDetail - CompUC.Get, using live solve count")
			} else {
				comp = c
			}
		} else if uc.deps.CompRepo != nil {
			c, err := uc.deps.CompRepo.Get(ctx)
			if err != nil {
				uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - GetDetail - CompRepo.Get, using live solve count")
			} else {
				comp = c
			}
		}
		if comp != nil && comp.IsFreezeActive() {
			frozenSolves, err := uc.deps.SolveRepo.GetByChallengeIDFrozen(ctx, challengeID, *comp.FreezeTime)
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

func (uc *ChallengeUseCase) getChallengeTags(ctx context.Context, challengeID uuid.UUID) ([]*entity.Tag, error) {
	if uc.deps.TagRepo == nil {
		return nil, nil
	}
	tags, err := uc.deps.TagRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - TagRepo.GetByChallengeID: %w", err)
	}
	return tags, nil
}

func (uc *ChallengeUseCase) getChallengeFiles(ctx context.Context, challengeID uuid.UUID) ([]*entity.File, error) {
	if uc.deps.FileRepo == nil {
		return nil, nil
	}
	files, err := uc.deps.FileRepo.GetByChallengeID(ctx, challengeID, entity.FileTypeChallenge)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - FileRepo.GetByChallengeID: %w", err)
	}
	return files, nil
}

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

func (uc *ChallengeUseCase) getChallengeFirstBlood(ctx context.Context, challengeID uuid.UUID) (*entity.FirstBloodEntry, error) {
	if uc.deps.SolveRepo == nil {
		return nil, nil
	}
	var comp *entity.Competition
	if uc.deps.CompUC != nil {
		c, err := uc.deps.CompUC.Get(ctx)
		if err != nil && !errors.Is(err, httperr.ErrCompetitionNotFound) {
			return nil, fmt.Errorf("ChallengeUseCase - GetDetail - CompUC.Get: %w", err)
		}
		comp = c
	} else if uc.deps.CompRepo != nil {
		c, err := uc.deps.CompRepo.Get(ctx)
		if err != nil && !errors.Is(err, httperr.ErrCompetitionNotFound) {
			return nil, fmt.Errorf("ChallengeUseCase - GetDetail - CompRepo.Get: %w", err)
		}
		comp = c
	}
	if comp != nil && comp.IsFreezeActive() {
		fb, err := uc.deps.SolveRepo.GetFirstBloodFrozen(ctx, challengeID, *comp.FreezeTime)
		if err != nil {
			if errors.Is(err, httperr.ErrSolveNotFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("ChallengeUseCase - GetDetail - SolveRepo.GetFirstBloodFrozen: %w", err)
		}
		return fb, nil
	}
	fb, err := uc.deps.SolveRepo.GetFirstBlood(ctx, challengeID)
	if err != nil && !errors.Is(err, httperr.ErrSolveNotFound) {
		return nil, fmt.Errorf("ChallengeUseCase - GetDetail - SolveRepo.GetFirstBlood: %w", err)
	}
	if err == nil {
		return fb, nil
	}
	return nil, nil
}

func (uc *ChallengeUseCase) checkChallengeSolved(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (bool, error) {
	if teamID == nil || uc.deps.SolveRepo == nil {
		return false, nil
	}
	_, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, *teamID, challengeID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, httperr.ErrSolveNotFound) {
		return false, nil
	}
	return false, fmt.Errorf("ChallengeUseCase - GetDetail - checkSolved: %w", err)
}

func (uc *ChallengeUseCase) GetSolves(ctx context.Context, challengeID uuid.UUID) ([]*entity.SolveWithDetails, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolves - ChallengeRepo.GetByID: %w", err)
	}
	if challenge.IsHidden {
		return nil, httperr.ErrChallengeNotFound
	}
	comp, err := uc.deps.CompUC.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolves - CompUC.Get: %w", err)
	}
	if comp != nil && comp.IsFreezeActive() {
		solves, err := uc.deps.SolveRepo.GetByChallengeIDFrozen(ctx, challengeID, *comp.FreezeTime)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetSolves - SolveRepo.GetByChallengeIDFrozen: %w", err)
		}
		return solves, nil
	}
	solves, err := uc.deps.SolveRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolves - SolveRepo.GetByChallengeID: %w", err)
	}
	return solves, nil
}

func validateDynamicScoringRange(initialValue, minValue int) error {
	if initialValue < 0 {
		return httperr.NewValidationErrorf("dynamic scoring initial value must be non-negative")
	}
	if minValue < 0 {
		return httperr.NewValidationErrorf("dynamic scoring min value must be non-negative")
	}
	if initialValue > 0 && minValue > 0 && initialValue < minValue {
		return httperr.ErrInvalidScoringRange
	}
	return nil
}

const maxFlagFormatRegexLen = 1024

func validateFlagFormatRegex(flagFormatRegex *string) error {
	if flagFormatRegex == nil || *flagFormatRegex == "" {
		return nil
	}
	if len(*flagFormatRegex) > maxFlagFormatRegexLen {
		return httperr.ErrInvalidFlagFormat
	}
	if _, err := regexp.Compile(*flagFormatRegex); err != nil {
		return httperr.ErrInvalidFlagFormat
	}
	return nil
}

func (uc *ChallengeUseCase) Create(ctx context.Context, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error) {
	if err := validateDynamicScoringRange(initialValue, minValue); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - validateDynamicScoringRange: %w", err)
	}
	flagHash, flagRegex, err := uc.challengeCreateComputeFlagHash(flag, isRegex, isCaseInsensitive)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create - challengeCreateComputeFlagHash: %w", err)
	}
	challenge := &entity.Challenge{
		Title:             title,
		Description:       description,
		Category:          category,
		Points:            points,
		InitialValue:      initialValue,
		MinValue:          minValue,
		Decay:             decay,
		SolveCount:        0,
		FlagHash:          flagHash,
		IsHidden:          isHidden,
		IsRegex:           isRegex,
		IsCaseInsensitive: isCaseInsensitive,
		FlagRegex:         flagRegex,
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

func (uc *ChallengeUseCase) challengeCreateComputeFlagHash(flag string, isRegex, isCaseInsensitive bool) (flagHash, flagRegex string, err error) {
	if isRegex {
		if uc.deps.Crypto == nil {
			return "", "", fmt.Errorf("ChallengeUseCase - Create - crypto.ErrServiceNotConfigured: %w", crypto.ErrServiceNotConfigured)
		}
		encrypted, err := uc.deps.Crypto.Encrypt(flag)
		if err != nil {
			return "", "", fmt.Errorf("ChallengeUseCase - Create - crypto.Encrypt: %w", err)
		}
		return entity.FlagHashRegexSentinel, encrypted, nil
	}
	userInput := strings.TrimSpace(flag)
	if isCaseInsensitive {
		userInput = strings.ToLower(userInput)
	}
	hash := sha256.Sum256([]byte(userInput))
	return hex.EncodeToString(hash[:]), "", nil
}

func (uc *ChallengeUseCase) challengeCreatePersist(ctx context.Context, challenge *entity.Challenge, tagIDs []uuid.UUID) error {
	return uc.challengeCreatePersistTx(ctx, challenge, tagIDs)
}

func (uc *ChallengeUseCase) challengeCreatePersistTx(ctx context.Context, challenge *entity.Challenge, tagIDs []uuid.UUID) error {
	return uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		challenge.ID = uuid.New()
		if err := uc.deps.ChallengeRepo.Create(ctx, challenge); err != nil {
			return fmt.Errorf("ChallengeUseCase - Create - ChallengeRepo.Create: %w", err)
		}
		if len(tagIDs) > 0 {
			if err := uc.deps.ChallengeRepo.SetTags(ctx, challenge.ID, tagIDs); err != nil {
				return fmt.Errorf("ChallengeUseCase - Create - ChallengeRepo.SetTags: %w", err)
			}
		}
		return nil
	})
}

func (uc *ChallengeUseCase) Update(ctx context.Context, ID uuid.UUID, title, description, category string, points int, initialValue, minValue, decay *int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error) {
	if err := validateFlagFormatRegex(flagFormatRegex); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update - validateFlagFormatRegex: %w", err)
	}
	challenge, err := uc.challengeUpdatePersist(ctx, ID, title, description, category, points, initialValue, minValue, decay, flag, isHidden, isRegex, isCaseInsensitive, flagFormatRegex, tagIDs)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update - challengeUpdatePersist: %w", err)
	}
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	}
	uc.InvalidateChallengeListCache(ctx)
	return challenge, nil
}

func (uc *ChallengeUseCase) challengeUpdatePersist(ctx context.Context, ID uuid.UUID, title, description, category string, points int, initialValue, minValue, decay *int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error) {
	var challenge *entity.Challenge
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err2 error
		challenge, err2 = uc.deps.ChallengeRepo.GetByID(ctx, ID)
		if err2 != nil {
			return fmt.Errorf("ChallengeUseCase - Update - ChallengeRepo.GetByID: %w", err2)
		}
		effectiveIV, effectiveMV := challenge.InitialValue, challenge.MinValue
		if initialValue != nil {
			effectiveIV = *initialValue
		}
		if minValue != nil {
			effectiveMV = *minValue
		}
		if err := validateDynamicScoringRange(effectiveIV, effectiveMV); err != nil {
			return fmt.Errorf("ChallengeUseCase - Update - validateDynamicScoringRange: %w", err)
		}
		uc.challengeUpdateApplyBasic(challenge, title, description, category, points, initialValue, minValue, decay, isHidden, isRegex, isCaseInsensitive, flagFormatRegex)
		if err2 = uc.challengeUpdateApplyFlag(challenge, flag, isRegex, isCaseInsensitive); err2 != nil {
			return fmt.Errorf("ChallengeUseCase - Update - challengeUpdateApplyFlag: %w", err2)
		}
		if err2 = uc.deps.ChallengeRepo.Update(ctx, challenge); err2 != nil {
			return fmt.Errorf("ChallengeUseCase - Update - ChallengeRepo.Update: %w", err2)
		}
		if err2 = uc.deps.ChallengeRepo.SetTags(ctx, ID, tagIDs); err2 != nil {
			return fmt.Errorf("ChallengeUseCase - Update - ChallengeRepo.SetTags: %w", err2)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - challengeUpdatePersist - TM.Run: %w", err)
	}
	return challenge, nil
}

func (uc *ChallengeUseCase) challengeUpdateApplyBasic(c *entity.Challenge, title, description, category string, points int, initialValue, minValue, decay *int, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string) {
	c.Title = title
	c.Description = description
	c.Category = category
	c.Points = points
	if initialValue != nil {
		c.InitialValue = *initialValue
	}
	if minValue != nil {
		c.MinValue = *minValue
	}
	if decay != nil {
		c.Decay = *decay
	}
	c.IsHidden = isHidden
	c.IsRegex = isRegex
	c.IsCaseInsensitive = isCaseInsensitive
	c.FlagFormatRegex = flagFormatRegex
}

func (uc *ChallengeUseCase) challengeUpdateApplyFlag(c *entity.Challenge, flag string, isRegex, isCaseInsensitive bool) error {
	if flag == "" {
		wasRegex := c.FlagHash == entity.FlagHashRegexSentinel
		if isRegex && !wasRegex {
			return httperr.ErrChallengeFlagRequiredWhenSwitchingMode
		}
		if !isRegex && wasRegex {
			return httperr.ErrChallengeFlagRequiredWhenSwitchingMode
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
		c.FlagRegex = encrypted
		c.FlagHash = entity.FlagHashRegexSentinel
		return nil
	}
	userInput := strings.TrimSpace(flag)
	if isCaseInsensitive {
		userInput = strings.ToLower(userInput)
	}
	hash := sha256.Sum256([]byte(userInput))
	c.FlagHash = hex.EncodeToString(hash[:])
	c.FlagRegex = ""
	return nil
}

func (uc *ChallengeUseCase) Delete(ctx context.Context, ID, actorID uuid.UUID, clientIP string) error {
	var fileLocations []string

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.ChallengeRepo.GetByID(ctx, ID); err != nil {
			return fmt.Errorf("ChallengeUseCase - Delete - ChallengeRepo.GetByID: %w", err)
		}
		fileLocations = uc.collectFileLocations(ctx, ID)

		if err := uc.deps.ChallengeRepo.Delete(ctx, ID); err != nil {
			return fmt.Errorf("ChallengeUseCase - Delete - ChallengeRepo.Delete: %w", err)
		}

		auditLog := &entity.AuditLog{
			UserID:     &actorID,
			Action:     entity.AuditActionDelete,
			EntityType: entity.AuditEntityChallenge,
			EntityID:   ID.String(),
			IP:         clientIP,
		}
		if err := uc.deps.AuditLogRepo.Create(ctx, auditLog); err != nil {
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

// deleteStorageFiles removes files from storage. Failures are logged only and not returned;
// the caller is not notified so this is best-effort cleanup (e.g. when deleting a challenge).
func (uc *ChallengeUseCase) deleteStorageFiles(ctx context.Context, locations []string) {
	if uc.deps.Storage == nil || len(locations) == 0 {
		return
	}
	const maxConcurrent = 4
	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, maxConcurrent)
	for _, loc := range locations {
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			defer func() { <-sem }()
			if err := uc.deps.Storage.Delete(ctx, loc); err != nil {
				uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - deleteStorageFiles", logger.Fields{"location": loc})
				return err
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - deleteStorageFiles - wait")
	}
}

func (uc *ChallengeUseCase) GetTags(ctx context.Context, challengeID uuid.UUID) ([]*entity.Tag, error) {
	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetTags - ChallengeRepo.GetByID: %w", err)
	}
	if uc.deps.TagRepo == nil {
		return []*entity.Tag{}, nil
	}
	tags, err := uc.deps.TagRepo.GetByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetTags - TagRepo.GetByChallengeID: %w", err)
	}
	return tags, nil
}

func (uc *ChallengeUseCase) GetRequirements(ctx context.Context, challengeID uuid.UUID) ([]*entity.ChallengeRequirement, error) {
	key := challengeID.String() + ":req:pub"
	v, err, _ := uc.requirementsSf.Do(key, func() (any, error) {
		challenge, err := uc.deps.ChallengeRepo.GetByID(context.WithoutCancel(ctx), challengeID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetRequirements - ChallengeRepo.GetByID: %w", err)
		}
		if challenge.IsHidden {
			return nil, httperr.ErrChallengeNotFound
		}
		return uc.deps.ChallengeRepo.GetRequirements(context.WithoutCancel(ctx), challengeID)
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetRequirements: %w", err)
	}
	requirements, ok := v.([]*entity.ChallengeRequirement)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - GetRequirements: unexpected type")
	}
	return requirements, nil
}

func (uc *ChallengeUseCase) SetRequirements(ctx context.Context, challengeID uuid.UUID, requirementIDs []uuid.UUID) error {
	if len(requirementIDs) > 0 {
		challenges, err := uc.deps.ChallengeRepo.GetByIDs(ctx, requirementIDs)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - SetRequirements - ChallengeRepo.GetByIDs: %w", err)
		}
		for _, reqID := range requirementIDs {
			if _, ok := challenges[reqID]; !ok {
				return httperr.NewValidationErrorf("invalid requirement_id")
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
		return httperr.NewValidationErrorf("requirements contain a cycle")
	}
	return uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
			return fmt.Errorf("ChallengeUseCase - SetRequirements - ChallengeRepo.GetByID: %w", err)
		}
		if err := uc.deps.ChallengeRepo.SetRequirements(ctx, challengeID, requirementIDs); err != nil {
			return fmt.Errorf("ChallengeUseCase - SetRequirements - ChallengeRepo.SetRequirements: %w", err)
		}
		return nil
	})
}

func requirementsContainCycle(start uuid.UUID, adj map[uuid.UUID][]uuid.UUID) bool {
	visiting := make(map[uuid.UUID]bool)
	var dfs func(uuid.UUID) bool
	dfs = func(node uuid.UUID) bool {
		if visiting[node] {
			return true
		}
		visiting[node] = true
		defer func() { visiting[node] = false }()
		for _, next := range adj[node] {
			if dfs(next) {
				return true
			}
		}
		return false
	}
	return dfs(start)
}

func (uc *ChallengeUseCase) GetSolution(ctx context.Context, challengeID uuid.UUID, teamID *uuid.UUID) (*entity.ChallengeSolution, error) {
	challenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolution - ChallengeRepo.GetByID: %w", err)
	}
	if challenge.IsHidden {
		return nil, httperr.ErrChallengeNotFound
	}
	if teamID == nil {
		return nil, httperr.ErrNotAuthenticated
	}
	if uc.deps.TeamRepo != nil {
		team, err := uc.deps.TeamRepo.GetByID(ctx, *teamID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - GetSolution - TeamRepo.GetByID: %w", err)
		}
		if team.IsBanned {
			return nil, httperr.ErrTeamBanned
		}
	}
	if _, err := uc.deps.SolveRepo.GetByTeamAndChallenge(ctx, *teamID, challengeID); err != nil {
		if errors.Is(err, httperr.ErrSolveNotFound) {
			return nil, httperr.ErrSolutionAccessDenied
		}
		return nil, fmt.Errorf("ChallengeUseCase - GetSolution - SolveRepo.GetByTeamAndChallenge: %w", err)
	}
	solution, err := uc.deps.ChallengeRepo.GetSolution(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetSolution - ChallengeRepo.GetSolution: %w", err)
	}
	return solution, nil
}

func (uc *ChallengeUseCase) ListSolutions(ctx context.Context, teamID uuid.UUID) ([]*entity.ChallengeSolutionEntry, error) {
	if uc.deps.TeamRepo != nil {
		team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - ListSolutions - TeamRepo.GetByID: %w", err)
		}
		if team.IsBanned {
			return nil, httperr.ErrTeamBanned
		}
	}
	entries, err := uc.deps.ChallengeRepo.ListSolutions(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - ListSolutions - ChallengeRepo.ListSolutions: %w", err)
	}
	return entries, nil
}

func (uc *ChallengeUseCase) GetFlags(ctx context.Context, challengeID uuid.UUID) (*entity.ChallengeFlags, error) {
	flags, err := uc.deps.ChallengeRepo.GetFlags(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetFlags - ChallengeRepo.GetFlags: %w", err)
	}
	return flags, nil
}

func (uc *ChallengeUseCase) GetTypes(context.Context) ([]string, error) {
	return []string{"standard", "dynamic"}, nil
}

func (uc *ChallengeUseCase) GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Challenge, error) {
	challenges, err := uc.deps.ChallengeRepo.GetMissingChallengesByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetMissingChallengesByTeamID - ChallengeRepo.GetMissingChallengesByTeamID: %w", err)
	}
	return challenges, nil
}

// GetMissingChallengesByUserID returns challenges not yet solved by the user's team.
// Returns an empty list if the user has no team (user.TeamID == nil).
func (uc *ChallengeUseCase) GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Challenge, error) {
	if uc.deps.UserRepo == nil {
		return []*entity.Challenge{}, nil
	}
	user, err := uc.deps.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetMissingChallengesByUserID - UserRepo.GetByID: %w", err)
	}
	if user == nil || user.TeamID == nil {
		return []*entity.Challenge{}, nil
	}
	challenges, err := uc.deps.ChallengeRepo.GetMissingChallengesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetMissingChallengesByUserID - ChallengeRepo.GetMissingChallengesByUserID: %w", err)
	}
	return challenges, nil
}
