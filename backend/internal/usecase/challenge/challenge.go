package challenge

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/scoring"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

const (
	// challengeListCachePrefix is kept only for backward-compatible invalidation of old keys.
	challengeListCachePrefix = "challenges:list:"

	// Two-layer cache: shared base (challenges without per-team solve status) + lightweight per-team solved-ID set.
	challengeBaseCachePrefix   = "challenges:base:"
	challengeBaseTTL           = 30 * time.Second
	challengeSolvedCachePrefix = "challenges:solved:"
	challengeSolvedTTL         = 10 * time.Second
	submitMinCheckDuration     = 75 * time.Millisecond
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
	if uc.deps.ListCache == nil {
		return uc.getAllInner(ctx, teamID, tagID)
	}

	baseKey := challengeBaseCacheKey(tagID)
	base, err := cache.GetOrLoad(uc.deps.ListCache, ctx, baseKey, challengeBaseTTL, func() ([]*usecase.ChallengeWithTags, error) {
		return uc.getAllInner(ctx, nil, tagID)
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - cache.GetOrLoad: %w", err)
	}

	if teamID == nil {
		return base, nil
	}

	// Layer 2: fetch and cache the set of solved challenge IDs for this team.
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
		return base, nil
	}

	solvedSet := make(map[uuid.UUID]struct{}, len(solvedIDs))
	for _, id := range solvedIDs {
		solvedSet[id] = struct{}{}
	}

	// Return a shallow copy with Solved flags applied; do not mutate cached base entries.
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
	return out, nil
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

	return &usecase.ChallengeDetail{
		Challenge:  challenge,
		Tags:       tags,
		Files:      files,
		Hints:      hints,
		FirstBlood: firstBlood,
		SolvedByMe: solvedByMe,
		SolveCount: challenge.SolveCount,
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

func validateFlagFormatRegex(flagFormatRegex *string) error {
	if flagFormatRegex == nil || *flagFormatRegex == "" {
		return nil
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

func (uc *ChallengeUseCase) Update(ctx context.Context, ID uuid.UUID, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error) {
	if err := validateDynamicScoringRange(initialValue, minValue); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update - validateDynamicScoringRange: %w", err)
	}
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

func (uc *ChallengeUseCase) challengeUpdatePersist(ctx context.Context, ID uuid.UUID, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error) {
	var challenge *entity.Challenge
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err2 error
		challenge, err2 = uc.deps.ChallengeRepo.GetByID(ctx, ID)
		if err2 != nil {
			return fmt.Errorf("ChallengeUseCase - Update - ChallengeRepo.GetByID: %w", err2)
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

func (uc *ChallengeUseCase) challengeUpdateApplyBasic(c *entity.Challenge, title, description, category string, points, initialValue, minValue, decay int, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string) {
	c.Title = title
	c.Description = description
	c.Category = category
	c.Points = points
	c.InitialValue = initialValue
	c.MinValue = minValue
	c.Decay = decay
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
	fileLocations := uc.collectFileLocations(ctx, ID)

	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if _, err := uc.deps.ChallengeRepo.GetByID(ctx, ID); err != nil {
			return fmt.Errorf("ChallengeUseCase - Delete - ChallengeRepo.GetByID: %w", err)
		}

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
	var locations []string
	for _, ft := range []entity.FileType{entity.FileTypeChallenge, entity.FileTypeWriteup} {
		files, err := uc.deps.FileRepo.GetByChallengeID(ctx, challengeID, ft)
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - collectFileLocations - GetByChallengeID")
			continue
		}
		for _, f := range files {
			locations = append(locations, f.Location)
		}
	}
	return locations
}

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
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (uc *ChallengeUseCase) getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	if re, ok := uc.regexCache.Get(pattern); ok {
		return re, nil
	}
	v, err, _ := uc.regexSf.Do(pattern, func() (any, error) {
		if re, ok := uc.regexCache.Get(pattern); ok {
			return re, nil
		}
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - getCompiledRegex - regexp.Compile: %w", err)
		}
		uc.regexCache.Set(pattern, compiled)
		return compiled, nil
	})
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - getCompiledRegex: %w", err)
	}
	re, ok := v.(*regexp.Regexp)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - getCompiledRegex: invalid type from singleflight")
	}
	return re, nil
}

type submitContext struct {
	ctx         context.Context
	challengeID uuid.UUID
	flag        string
	userID      uuid.UUID
	teamID      uuid.UUID
	team        *entity.Team
	comp        *entity.Competition
}

//nolint:gocognit,gocyclo // submit flow: validation chain + solve record
func (uc *ChallengeUseCase) SubmitFlag(ctx context.Context, challengeID uuid.UUID, flag string, userID uuid.UUID, teamID *uuid.UUID) (bool, error) {
	if teamID == nil {
		return false, httperr.ErrUserMustBeInTeam
	}

	comp, err := uc.submitCheckCompetitionTime(ctx)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitCheckCompetitionTime: %w", err)
	}

	sc := &submitContext{ctx: ctx, challengeID: challengeID, flag: strings.TrimSpace(flag), userID: userID, teamID: *teamID, comp: comp}
	if sc.flag == "" {
		return false, httperr.ErrInvalidFlagFormat
	}

	if uc.deps.TeamRepo != nil {
		// Bypass the BoundedCache for the ban/mode check: the cache has no TTL,
		// so a team banned after caching would pass the check until the entry is
		// evicted. A direct DB read is cheap and prevents banned teams from
		// submitting even incorrect flags.
		team, err := uc.deps.TeamRepo.GetByID(ctx, sc.teamID)
		if err != nil {
			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - TeamRepo.GetByID: %w", err)
		}
		sc.team = team
		if team.IsBanned {
			return false, httperr.ErrTeamBanned
		}
		if comp != nil {
			if comp.Mode == entity.ModeTeamsOnly && team.IsSolo {
				return false, httperr.ErrTeamModeRequired
			}
			if comp.Mode == entity.ModeSoloOnly && !team.IsSolo {
				return false, httperr.ErrSoloModeRequired
			}
		}
	}

	var challenge *entity.Challenge
	eg, egCtx := errgroup.WithContext(sc.ctx)
	scEg := *sc
	scEg.ctx = egCtx
	eg.Go(func() error {
		var e error
		challenge, e = uc.submitGetChallenge(&scEg)
		return e
	})
	eg.Go(func() error {
		return uc.submitCheckRequirements(&scEg)
	})
	if err := eg.Wait(); err != nil {
		if errors.Is(err, httperr.ErrChallengeNotFound) {
			return false, httperr.ErrChallengeNotFound
		}
		if errors.Is(err, httperr.ErrRequirementsNotMet) {
			return false, httperr.ErrRequirementsNotMet
		}
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - parallel checks: %w", err)
	}
	if err := uc.submitValidateFlagFormat(sc, challenge); err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitValidateFlagFormat: %w", err)
	}
	checkStart := time.Now()
	correct, err := uc.submitCheckFlag(sc, challenge)
	if elapsed := time.Since(checkStart); elapsed < submitMinCheckDuration {
		select {
		case <-time.After(submitMinCheckDuration - elapsed):
		case <-sc.ctx.Done():
			return false, sc.ctx.Err()
		}
	}
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - checkFlag: %w", err)
	}
	if !correct {
		return false, nil
	}
	solvedChallenge, solveCount, err := uc.submitRecordSolve(sc, challenge)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - submitRecordSolve: %w", err)
	}
	uc.submitInvalidateCache(sc.ctx, sc.teamID)
	uc.submitNotifySolve(sc.teamID, solvedChallenge, solveCount == 1)
	return true, nil
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

func (uc *ChallengeUseCase) AdminUpsertSolution(ctx context.Context, challengeID uuid.UUID, content string) (*entity.ChallengeSolution, error) {
	if _, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID); err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - AdminUpsertSolution - ChallengeRepo.GetByID: %w", err)
	}
	solution, err := uc.deps.ChallengeRepo.UpsertSolution(ctx, challengeID, content)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - AdminUpsertSolution - ChallengeRepo.UpsertSolution: %w", err)
	}
	return solution, nil
}

func (uc *ChallengeUseCase) AdminDeleteSolution(ctx context.Context, challengeID uuid.UUID) error {
	if err := uc.deps.ChallengeRepo.DeleteSolution(ctx, challengeID); err != nil {
		return fmt.Errorf("ChallengeUseCase - AdminDeleteSolution - ChallengeRepo.DeleteSolution: %w", err)
	}
	return nil
}

func (uc *ChallengeUseCase) GetFlags(ctx context.Context, challengeID uuid.UUID) (*entity.ChallengeFlags, error) {
	flags, err := uc.deps.ChallengeRepo.GetFlags(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetFlags - ChallengeRepo.GetFlags: %w", err)
	}
	return flags, nil
}

func (uc *ChallengeUseCase) GetTypes(ctx context.Context) ([]string, error) {
	return []string{"standard", "dynamic"}, nil
}

func (uc *ChallengeUseCase) GetMissingChallengesByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Challenge, error) {
	challenges, err := uc.deps.ChallengeRepo.GetMissingChallengesByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetMissingChallengesByTeamID - ChallengeRepo.GetMissingChallengesByTeamID: %w", err)
	}
	return challenges, nil
}

func (uc *ChallengeUseCase) GetMissingChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Challenge, error) {
	challenges, err := uc.deps.ChallengeRepo.GetMissingChallengesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetMissingChallengesByUserID - ChallengeRepo.GetMissingChallengesByUserID: %w", err)
	}
	return challenges, nil
}

func (uc *ChallengeUseCase) submitCheckCompetitionTime(ctx context.Context) (*entity.Competition, error) {
	if uc.deps.CompUC == nil && uc.deps.CompRepo == nil {
		return nil, nil
	}
	var comp *entity.Competition
	var err error
	if uc.deps.CompUC != nil {
		comp, err = uc.deps.CompUC.Get(ctx)
	} else {
		comp, err = uc.deps.CompRepo.Get(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - SubmitFlag - CompetitionRepo.Get: %w", err)
	}
	if !comp.IsSubmissionAllowed() {
		return nil, httperr.ErrSubmissionNotAllowed
	}
	return comp, nil
}

func (uc *ChallengeUseCase) submitGetChallenge(sc *submitContext) (*entity.Challenge, error) {
	key := sc.challengeID.String()
	v, err, _ := uc.challengeFetchSf.Do(key, func() (any, error) {
		return uc.deps.ChallengeRepo.GetByID(context.WithoutCancel(sc.ctx), sc.challengeID)
	})
	if err != nil {
		if errors.Is(err, httperr.ErrChallengeNotFound) {
			return nil, httperr.ErrChallengeNotFound
		}
		return nil, fmt.Errorf("ChallengeUseCase - SubmitFlag - ChallengeRepo.GetByID: %w", err)
	}
	challenge, ok := v.(*entity.Challenge)
	if !ok {
		return nil, fmt.Errorf("ChallengeUseCase - SubmitFlag - ChallengeRepo.GetByID: unexpected type")
	}
	if challenge.IsHidden {
		return nil, httperr.ErrChallengeNotFound
	}
	return challenge, nil
}

func (uc *ChallengeUseCase) submitCheckRequirements(sc *submitContext) error {
	key := sc.challengeID.String() + ":req"
	v, err, _ := uc.requirementsSf.Do(key, func() (any, error) {
		return uc.deps.ChallengeRepo.GetRequirements(context.WithoutCancel(sc.ctx), sc.challengeID)
	})
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - SubmitFlag - GetRequirements: %w", err)
	}
	requirements, ok := v.([]*entity.ChallengeRequirement)
	if !ok {
		return fmt.Errorf("ChallengeUseCase - SubmitFlag - GetRequirements: unexpected type")
	}
	if uc.deps.SolveRepo == nil || len(requirements) == 0 {
		return nil
	}
	requirementIDs := make([]uuid.UUID, 0, len(requirements))
	for _, req := range requirements {
		requirementIDs = append(requirementIDs, req.ChallengeID)
	}
	solvedIDs, err := uc.deps.SolveRepo.GetSolvedChallengeIDsByTeam(sc.ctx, sc.teamID, requirementIDs)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - SubmitFlag - SolveRepo.GetSolvedChallengeIDsByTeam: %w", err)
	}
	solvedSet := make(map[uuid.UUID]struct{}, len(solvedIDs))
	for _, id := range solvedIDs {
		solvedSet[id] = struct{}{}
	}
	for _, req := range requirements {
		if _, ok := solvedSet[req.ChallengeID]; !ok {
			return httperr.ErrRequirementsNotMet
		}
	}
	return nil
}

func (uc *ChallengeUseCase) submitValidateFlagFormat(sc *submitContext, challenge *entity.Challenge) error {
	formatRegex := ""
	if challenge.FlagFormatRegex != nil && *challenge.FlagFormatRegex != "" {
		formatRegex = *challenge.FlagFormatRegex
	} else if sc.comp != nil && sc.comp.FlagRegex != nil && *sc.comp.FlagRegex != "" {
		formatRegex = *sc.comp.FlagRegex
	}
	if formatRegex == "" {
		return nil
	}
	compiled, err := uc.getCompiledRegex(formatRegex)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - SubmitFlag - CompileFormatRegex: %w", err)
	}
	if !compiled.MatchString(sc.flag) {
		return httperr.ErrInvalidFlagFormat
	}
	return nil
}

func (uc *ChallengeUseCase) submitCheckFlag(sc *submitContext, challenge *entity.Challenge) (bool, error) {
	if challenge.IsRegex {
		return uc.submitCheckRegexFlag(sc, challenge)
	}
	return uc.submitCheckHashFlag(sc, challenge), nil
}

func (uc *ChallengeUseCase) submitCheckRegexFlag(sc *submitContext, challenge *entity.Challenge) (bool, error) {
	if uc.deps.Crypto == nil {
		return false, fmt.Errorf("ChallengeUseCase - submitCheckRegexFlag - crypto not configured")
	}
	pattern, err := uc.deps.Crypto.Decrypt(challenge.FlagRegex)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - submitCheckRegexFlag - crypto.Decrypt: %w", err)
	}
	if challenge.IsCaseInsensitive {
		pattern = "(?i)" + pattern
	}
	compiled, err := uc.getCompiledRegex(pattern)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - submitCheckRegexFlag - regexp.Compile: %w", err)
	}
	return compiled.MatchString(sc.flag), nil
}

func (uc *ChallengeUseCase) submitCheckHashFlag(sc *submitContext, challenge *entity.Challenge) bool {
	userInput := sc.flag
	if challenge.IsCaseInsensitive {
		userInput = strings.ToLower(userInput)
	}
	hash := sha256.Sum256([]byte(userInput))
	hashStr := hex.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(hashStr), []byte(challenge.FlagHash)) == 1
}

//nolint:gocognit,gocyclo // submit flow with team/solve checks
func (uc *ChallengeUseCase) submitRecordSolve(sc *submitContext, challenge *entity.Challenge) (*entity.Challenge, int, error) {
	// Competition time check uses pre-fetched sc.comp; the CompetitionActive middleware already
	// enforced this before the handler ran, so a stale check here is acceptable.
	if sc.comp != nil && !sc.comp.IsSubmissionAllowed() {
		return nil, 0, httperr.ErrSubmissionNotAllowed
	}

	// Copy the struct so concurrent solves on the same challenge don't race on Points mutation.
	challengeCopy := *challenge
	solvedChallenge := &challengeCopy
	var solveCount int
	err := uc.deps.TM.Run(sc.ctx, func(ctx context.Context) error {
		if err := uc.submitRecordSolveCheckExisting(ctx, sc); err != nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve - submitRecordSolveCheckExisting: %w", err)
		}
		// Lock user first (matches the lock order used by kick/leave/join, preventing
		// deadlocks) and verify the user is still a member of sc.teamID. Without this
		// check, a user kicked between the RequireTeam middleware and this transaction
		// would have the solve credited to a team they no longer belong to (TOCTOU).
		if uc.deps.UserRepo != nil {
			if err := uc.deps.UserRepo.Lock(ctx, sc.userID); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - UserRepo.Lock: %w", err)
			}
			freshUser, err := uc.deps.UserRepo.GetByID(ctx, sc.userID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - UserRepo.GetByID: %w", err)
			}
			if freshUser.TeamID == nil || *freshUser.TeamID != sc.teamID {
				return httperr.ErrTeamMemberNotFound
			}
			if freshUser.IsBanned {
				return httperr.ErrUserBanned
			}
		}
		// Lock the team row with FOR UPDATE so this transaction blocks concurrent
		// BanTeam and vice-versa. Without the lock, a ban committed between the
		// read and the solve insert would be missed (TOCTOU).
		if uc.deps.TeamRepo != nil {
			if err := uc.deps.TeamRepo.Lock(ctx, sc.teamID); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - TeamRepo.Lock: %w", err)
			}
			freshTeam, err := uc.deps.TeamRepo.GetByID(ctx, sc.teamID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolve - TeamRepo.GetByID: %w", err)
			}
			if freshTeam.IsBanned {
				return httperr.ErrTeamBanned
			}
			if sc.comp != nil {
				if sc.comp.Mode == entity.ModeTeamsOnly && freshTeam.IsSolo {
					return httperr.ErrTeamModeRequired
				}
				if sc.comp.Mode == entity.ModeSoloOnly && !freshTeam.IsSolo {
					return httperr.ErrSoloModeRequired
				}
				if sc.comp.MinTeamSize > 0 && !freshTeam.IsSolo {
					count, err := uc.deps.TeamRepo.CountTeamMembers(ctx, sc.teamID)
					if err != nil {
						return fmt.Errorf("ChallengeUseCase - submitRecordSolve - TeamRepo.CountTeamMembers: %w", err)
					}
					if count < sc.comp.MinTeamSize {
						return httperr.ErrTeamBelowMinSize
					}
				}
			}
		}
		if solvedChallenge.IsHidden {
			return httperr.ErrChallengeNotFound
		}
		var err error
		solveCount, err = uc.deps.ChallengeRepo.IncrementSolveCount(ctx, sc.challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve - ChallengeRepo.IncrementSolveCount: %w", err)
		}
		pointsAtSolve, err := scoring.ApplySolveScore(ctx,
			solvedChallenge.InitialValue, solvedChallenge.MinValue, solvedChallenge.Decay, solvedChallenge.Points, solveCount,
			func(ctx context.Context, pts int) error {
				if err := uc.deps.ChallengeRepo.UpdatePoints(ctx, sc.challengeID, pts); err != nil {
					return fmt.Errorf("ChallengeUseCase - submitRecordSolve - ChallengeRepo.UpdatePoints: %w", err)
				}
				solvedChallenge.Points = pts
				return nil
			},
		)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve - ApplySolveScore: %w", err)
		}
		solve := &entity.Solve{UserID: sc.userID, TeamID: sc.teamID, ChallengeID: sc.challengeID, PointsAtSolve: pointsAtSolve}
		if err := uc.deps.SolveRepo.Create(ctx, solve); err != nil {
			return fmt.Errorf("ChallengeUseCase - submitRecordSolve - SolveRepo.Create: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("ChallengeUseCase - submitRecordSolve - TM.Run: %w", err)
	}
	return solvedChallenge, solveCount, nil
}

func (uc *ChallengeUseCase) submitRecordSolveCheckExisting(ctx context.Context, sc *submitContext) error {
	_, err := uc.deps.SolveRepo.GetByTeamAndChallengeForUpdate(ctx, sc.teamID, sc.challengeID)
	if err == nil {
		return httperr.ErrAlreadySolved
	}
	if !errors.Is(err, httperr.ErrSolveNotFound) {
		return fmt.Errorf("ChallengeUseCase - submitRecordSolveCheckExisting - SolveRepo.GetByTeamAndChallengeForUpdate: %w", err)
	}
	return nil
}

func (uc *ChallengeUseCase) submitRecordSolveUpdatePointsIfDecay(ctx context.Context, challengeID uuid.UUID, solvedChallenge *entity.Challenge, solveCount int) error {
	_, err := scoring.ApplySolveScore(ctx,
		solvedChallenge.InitialValue, solvedChallenge.MinValue, solvedChallenge.Decay, solvedChallenge.Points, solveCount,
		func(ctx context.Context, pts int) error {
			if err := uc.deps.ChallengeRepo.UpdatePoints(ctx, challengeID, pts); err != nil {
				return fmt.Errorf("ChallengeUseCase - submitRecordSolveUpdatePointsIfDecay - ChallengeRepo.UpdatePoints: %w", err)
			}
			solvedChallenge.Points = pts
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - submitRecordSolveUpdatePointsIfDecay - ApplySolveScore: %w", err)
	}
	return nil
}

func (uc *ChallengeUseCase) submitInvalidateCache(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	}
	uc.InvalidateChallengeListCacheForTeam(ctx, teamID)
}

func (uc *ChallengeUseCase) InvalidateScoreboardCache(ctx context.Context) {
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	}
}

func (uc *ChallengeUseCase) InvalidateScoreboardCacheForTeam(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	}
}

func (uc *ChallengeUseCase) InvalidateChallengeListCache(ctx context.Context) {
	if uc.deps.ListCache == nil {
		return
	}
	// Invalidate both old per-team keys (backward compat) and new two-layer keys.
	_ = uc.deps.ListCache.DeleteByPrefix(ctx, challengeListCachePrefix)   //nolint:errcheck // best-effort
	_ = uc.deps.ListCache.DeleteByPrefix(ctx, challengeBaseCachePrefix)   //nolint:errcheck // best-effort
	_ = uc.deps.ListCache.DeleteByPrefix(ctx, challengeSolvedCachePrefix) //nolint:errcheck // best-effort
}

func (uc *ChallengeUseCase) InvalidateChallengeListCacheForTeam(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ListCache == nil {
		return
	}
	// Invalidate old per-team key (backward compat) and new per-team solved-IDs key.
	_ = uc.deps.ListCache.DeleteByPrefix(ctx, challengeListCachePrefix+teamID.String()+":") //nolint:errcheck // best-effort
	_ = uc.deps.ListCache.Del(ctx, challengeSolvedCachePrefix+teamID.String())              //nolint:errcheck // best-effort
	// Invalidate base cache too so updated solve counts are reflected promptly.
	_ = uc.deps.ListCache.DeleteByPrefix(ctx, challengeBaseCachePrefix) //nolint:errcheck // best-effort
}

// InvalidateAll implements cache.ChallengeListCacheInvalidator.
func (uc *ChallengeUseCase) InvalidateAll(ctx context.Context) { uc.InvalidateChallengeListCache(ctx) }

// InvalidateForTeam implements cache.ChallengeListCacheInvalidator.
func (uc *ChallengeUseCase) InvalidateForTeam(ctx context.Context, teamID uuid.UUID) {
	uc.InvalidateChallengeListCacheForTeam(ctx, teamID)
}

func (uc *ChallengeUseCase) submitNotifySolve(teamID uuid.UUID, challenge *entity.Challenge, isFirstBlood bool) {
	if uc.deps.Broadcaster != nil && challenge != nil {
		uc.deps.Broadcaster.NotifySolve(teamID, challenge.Title, challenge.Points, isFirstBlood)
	}
}

func (uc *ChallengeUseCase) AdminCreateSolve(ctx context.Context, userID, teamID, challengeID uuid.UUID) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if uc.deps.TeamRepo != nil {
			if err := uc.deps.TeamRepo.Lock(ctx, teamID); err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - TeamRepo.Lock: %w", err)
			}
			team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - TeamRepo.GetByID: %w", err)
			}
			if team.IsBanned {
				return httperr.ErrTeamBanned
			}
		}
		solvedChallenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - ChallengeRepo.GetByID: %w", err)
		}
		if uc.deps.UserRepo != nil {
			user, err := uc.deps.UserRepo.GetByID(ctx, userID)
			if err != nil {
				return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - UserRepo.GetByID: %w", err)
			}
			if user.TeamID == nil || *user.TeamID != teamID {
				return httperr.ErrUserNotInTeam
			}
			if user.IsBanned {
				return httperr.ErrUserBanned
			}
		}
		if _, err := uc.deps.SolveRepo.GetByTeamAndChallengeForUpdate(ctx, teamID, challengeID); err == nil {
			return nil
		} else if !errors.Is(err, httperr.ErrSolveNotFound) {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - SolveRepo.GetByTeamAndChallengeForUpdate: %w", err)
		}
		solveCount, err := uc.deps.ChallengeRepo.IncrementSolveCount(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - ChallengeRepo.IncrementSolveCount: %w", err)
		}
		pointsAtSolve, err := scoring.ApplySolveScore(ctx,
			solvedChallenge.InitialValue, solvedChallenge.MinValue, solvedChallenge.Decay, solvedChallenge.Points, solveCount,
			func(ctx context.Context, pts int) error {
				if err := uc.deps.ChallengeRepo.UpdatePoints(ctx, challengeID, pts); err != nil {
					return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - ChallengeRepo.UpdatePoints: %w", err)
				}
				solvedChallenge.Points = pts
				return nil
			},
		)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - ApplySolveScore: %w", err)
		}
		solve := &entity.Solve{UserID: userID, TeamID: teamID, ChallengeID: challengeID, PointsAtSolve: pointsAtSolve}
		if err = uc.deps.SolveRepo.Create(ctx, solve); err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - SolveRepo.Create: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - AdminCreateSolve - TM.Run: %w", err)
	}
	uc.submitInvalidateCache(ctx, teamID)
	return nil
}

func (uc *ChallengeUseCase) AdminDeleteSolve(ctx context.Context, teamID, challengeID uuid.UUID) error {
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		solvedChallenge, err := uc.deps.ChallengeRepo.GetByID(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - ChallengeRepo.GetByID: %w", err)
		}
		if err = uc.deps.SolveRepo.DeleteByTeamAndChallenge(ctx, teamID, challengeID); err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - SolveRepo.DeleteByTeamAndChallenge: %w", err)
		}
		solveCount, err := uc.deps.ChallengeRepo.DecrementSolveCount(ctx, challengeID)
		if err != nil {
			return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - ChallengeRepo.DecrementSolveCount: %w", err)
		}
		return uc.submitRecordSolveUpdatePointsIfDecay(ctx, challengeID, solvedChallenge, solveCount)
	})
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - AdminDeleteSolve - TM.Run: %w", err)
	}
	uc.submitInvalidateCache(ctx, teamID)
	return nil
}
