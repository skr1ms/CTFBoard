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

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/pkg/cache"
	"github.com/skr1ms/CTFBoard/pkg/crypto"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
	"github.com/skr1ms/CTFBoard/pkg/websocket"
	"golang.org/x/sync/singleflight"
)

type ChallengeUseCase struct {
	challengeRepo   repo.ChallengeRepository
	tagRepo         repo.TagRepository
	solveRepo       repo.SolveRepository
	txRepo          repo.TxRepository
	compRepo        repo.CompetitionRepository
	teamRepo        repo.TeamRepository
	redis           *redis.Client
	scoreboardCache cache.ScoreboardCacheInvalidator
	broadcaster     websocket.SolveBroadcaster
	auditLogRepo    repo.AuditLogRepository
	crypto          crypto.Service
	regexCache      *cache.BoundedCache[string, *regexp.Regexp]
	regexSf         singleflight.Group
}

func NewChallengeUseCase(challengeRepo repo.ChallengeRepository, opts ...ChallengeUCOption) *ChallengeUseCase {
	uc := &ChallengeUseCase{
		challengeRepo: challengeRepo,
		regexCache:    cache.NewBoundedCache[string, *regexp.Regexp](cache.DefaultBoundedCacheSize),
	}
	for _, opt := range opts {
		opt(uc)
	}
	return uc
}

func (uc *ChallengeUseCase) GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*usecase.ChallengeWithTags, error) {
	challenges, err := uc.challengeRepo.GetAll(ctx, teamID, tagID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "ChallengeUseCase - GetAll")
	}
	if uc.tagRepo == nil {
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
	tagsMap, err := uc.tagRepo.GetByChallengeIDs(ctx, ids)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "ChallengeUseCase - GetAll - GetTags")
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

func (uc *ChallengeUseCase) Create(ctx context.Context, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error) {
	var flagHash string
	var flagRegex string

	if isRegex {
		if uc.crypto == nil {
			return nil, usecaseutil.Wrap(crypto.ErrServiceNotConfigured, "ChallengeUseCase - Create")
		}
		encrypted, err := uc.crypto.Encrypt(flag)
		if err != nil {
			return nil, usecaseutil.Wrap(err, "ChallengeUseCase - Create - Encrypt")
		}
		flagRegex = encrypted
		flagHash = "REGEX_CHALLENGE"
	} else {
		userInput := flag
		if isCaseInsensitive {
			userInput = strings.ToLower(strings.TrimSpace(flag))
		}
		hash := sha256.Sum256([]byte(userInput))
		flagHash = hex.EncodeToString(hash[:])
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

	err := uc.challengeRepo.Create(ctx, challenge)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "ChallengeUseCase - Create")
	}
	if uc.tagRepo != nil && len(tagIDs) > 0 {
		if err := uc.tagRepo.SetChallengeTags(ctx, challenge.ID, tagIDs); err != nil {
			return nil, usecaseutil.Wrap(err, "ChallengeUseCase - Create - SetTags")
		}
	}
	return challenge, nil
}

func (uc *ChallengeUseCase) Update(ctx context.Context, ID uuid.UUID, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error) {
	challenge, err := uc.challengeRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "ChallengeUseCase - Update - GetByID")
	}
	uc.challengeUpdateApplyBasic(challenge, title, description, category, points, initialValue, minValue, decay, isHidden, isRegex, isCaseInsensitive, flagFormatRegex)
	if err := uc.challengeUpdateApplyFlag(challenge, flag, isRegex, isCaseInsensitive); err != nil {
		return nil, err
	}
	if err := uc.challengeRepo.Update(ctx, challenge); err != nil {
		return nil, usecaseutil.Wrap(err, "ChallengeUseCase - Update")
	}
	if uc.tagRepo != nil {
		if err := uc.tagRepo.SetChallengeTags(ctx, ID, tagIDs); err != nil {
			return nil, usecaseutil.Wrap(err, "ChallengeUseCase - Update - SetTags")
		}
	}
	if uc.scoreboardCache != nil {
		uc.scoreboardCache.InvalidateAll(ctx)
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
		return nil
	}
	if isRegex {
		if uc.crypto == nil {
			return usecaseutil.Wrap(crypto.ErrServiceNotConfigured, "ChallengeUseCase - Update")
		}
		encrypted, err := uc.crypto.Encrypt(flag)
		if err != nil {
			return usecaseutil.Wrap(err, "ChallengeUseCase - Update - Encrypt")
		}
		c.FlagRegex = encrypted
		c.FlagHash = "REGEX_CHALLENGE"
		return nil
	}
	userInput := flag
	if isCaseInsensitive {
		userInput = strings.ToLower(strings.TrimSpace(flag))
	}
	hash := sha256.Sum256([]byte(userInput))
	c.FlagHash = hex.EncodeToString(hash[:])
	c.FlagRegex = ""
	return nil
}

func (uc *ChallengeUseCase) Delete(ctx context.Context, ID, actorID uuid.UUID, clientIP string) error {
	err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx repo.Transaction) error {
		if _, err := uc.txRepo.GetChallengeByIDTx(ctx, tx, ID); err != nil {
			return err
		}

		if err := uc.txRepo.DeleteChallengeTx(ctx, tx, ID); err != nil {
			return usecaseutil.Wrap(err, "DeleteChallengeTx")
		}

		auditLog := &entity.AuditLog{
			UserID:     &actorID,
			Action:     entity.AuditActionDelete,
			EntityType: entity.AuditEntityChallenge,
			EntityID:   ID.String(),
			IP:         clientIP,
		}
		if err := uc.txRepo.CreateAuditLogTx(ctx, tx, auditLog); err != nil {
			return usecaseutil.Wrap(err, "CreateAuditLogTx")
		}
		return nil
	})
	if err != nil {
		return usecaseutil.Wrap(err, "ChallengeUseCase - Delete - Transaction")
	}

	if uc.scoreboardCache != nil {
		uc.scoreboardCache.InvalidateAll(ctx)
	}
	return nil
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
			return nil, usecaseutil.Wrap(err, "ChallengeUseCase - getCompiledRegex - Compile")
		}
		uc.regexCache.Set(pattern, compiled)
		return compiled, nil
	})
	if err != nil {
		return nil, err
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
}

func (uc *ChallengeUseCase) SubmitFlag(ctx context.Context, challengeID uuid.UUID, flag string, userID uuid.UUID, teamID *uuid.UUID) (bool, error) {
	if teamID == nil {
		return false, entityError.ErrUserMustBeInTeam
	}
	sc := &submitContext{ctx: ctx, challengeID: challengeID, flag: strings.TrimSpace(flag), userID: userID, teamID: *teamID}

	if err := uc.submitValidateTeam(sc); err != nil {
		return false, err
	}
	challenge, err := uc.submitGetChallenge(sc)
	if err != nil {
		return false, err
	}
	if err := uc.submitValidateFlagFormat(sc, challenge); err != nil {
		return false, err
	}
	if !uc.submitCheckFlag(sc, challenge) {
		return false, nil
	}
	solvedChallenge, solveCount, err := uc.submitRecordSolve(sc, challenge)
	if err != nil {
		return errors.Is(err, entityError.ErrAlreadySolved), err
	}
	uc.submitInvalidateCache(sc.ctx)
	uc.submitNotifySolve(sc.teamID, solvedChallenge, solveCount == 1)
	return true, nil
}

func (uc *ChallengeUseCase) submitValidateTeam(sc *submitContext) error {
	if uc.teamRepo == nil {
		return nil
	}
	team, err := uc.teamRepo.GetByID(sc.ctx, sc.teamID)
	if err != nil {
		return usecaseutil.Wrap(err, "ChallengeUseCase - SubmitFlag - GetTeam")
	}
	if team.IsBanned {
		return entityError.ErrTeamBanned
	}
	return nil
}

func (uc *ChallengeUseCase) submitGetChallenge(sc *submitContext) (*entity.Challenge, error) {
	challenge, err := uc.challengeRepo.GetByID(sc.ctx, sc.challengeID)
	if err != nil {
		if errors.Is(err, entityError.ErrChallengeNotFound) {
			return nil, entityError.ErrChallengeNotFound
		}
		return nil, usecaseutil.Wrap(err, "ChallengeUseCase - SubmitFlag - GetByID")
	}
	return challenge, nil
}

func (uc *ChallengeUseCase) submitValidateFlagFormat(sc *submitContext, challenge *entity.Challenge) error {
	formatRegex := ""
	if challenge.FlagFormatRegex != nil && *challenge.FlagFormatRegex != "" {
		formatRegex = *challenge.FlagFormatRegex
	} else if uc.compRepo != nil {
		comp, err := uc.compRepo.Get(sc.ctx)
		if err == nil && comp.FlagRegex != nil && *comp.FlagRegex != "" {
			formatRegex = *comp.FlagRegex
		}
	}
	if formatRegex == "" {
		return nil
	}
	matched, err := regexp.MatchString(formatRegex, sc.flag)
	if err != nil {
		return usecaseutil.Wrap(err, "ChallengeUseCase - SubmitFlag - MatchString")
	}
	if !matched {
		return entityError.ErrInvalidFlagFormat
	}
	return nil
}

func (uc *ChallengeUseCase) submitCheckFlag(sc *submitContext, challenge *entity.Challenge) bool {
	if challenge.IsRegex {
		return uc.submitCheckRegexFlag(sc, challenge)
	}
	return uc.submitCheckHashFlag(sc, challenge)
}

func (uc *ChallengeUseCase) submitCheckRegexFlag(sc *submitContext, challenge *entity.Challenge) bool {
	if uc.crypto == nil {
		return false
	}
	pattern, err := uc.crypto.Decrypt(challenge.FlagRegex)
	if err != nil {
		return false
	}
	if challenge.IsCaseInsensitive {
		pattern = "(?i)" + pattern
	}
	compiled, err := uc.getCompiledRegex(pattern)
	if err != nil {
		return false
	}
	return compiled.MatchString(sc.flag)
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

func (uc *ChallengeUseCase) submitRecordSolve(sc *submitContext, _ *entity.Challenge) (*entity.Challenge, int, error) {
	var solvedChallenge *entity.Challenge
	var solveCount int
	err := uc.txRepo.RunTransaction(sc.ctx, func(ctx context.Context, tx repo.Transaction) error {
		if err := uc.submitRecordSolveCheckExisting(ctx, tx, sc); err != nil {
			return err
		}
		var err2 error
		solvedChallenge, err2 = uc.txRepo.GetChallengeByIDTx(ctx, tx, sc.challengeID)
		if err2 != nil {
			return usecaseutil.Wrap(err2, "GetChallengeByIDTx")
		}
		solve := &entity.Solve{UserID: sc.userID, TeamID: sc.teamID, ChallengeID: sc.challengeID}
		if err2 = uc.txRepo.CreateSolveTx(ctx, tx, solve); err2 != nil {
			return usecaseutil.Wrap(err2, "CreateSolveTx")
		}
		solveCount, err2 = uc.txRepo.IncrementChallengeSolveCountTx(ctx, tx, sc.challengeID)
		if err2 != nil {
			return usecaseutil.Wrap(err2, "IncrementChallengeSolveCountTx")
		}
		return uc.submitRecordSolveUpdatePointsIfDecay(ctx, tx, sc.challengeID, solvedChallenge, solveCount)
	})
	if err != nil {
		return nil, 0, err
	}
	return solvedChallenge, solveCount, nil
}

func (uc *ChallengeUseCase) submitRecordSolveCheckExisting(ctx context.Context, tx repo.Transaction, sc *submitContext) error {
	_, err := uc.txRepo.GetSolveByTeamAndChallengeTx(ctx, tx, sc.teamID, sc.challengeID)
	if err == nil {
		return entityError.ErrAlreadySolved
	}
	if !errors.Is(err, entityError.ErrSolveNotFound) {
		return usecaseutil.Wrap(err, "GetSolveByTeamAndChallengeTx")
	}
	return nil
}

func (uc *ChallengeUseCase) submitRecordSolveUpdatePointsIfDecay(ctx context.Context, tx repo.Transaction, challengeID uuid.UUID, solvedChallenge *entity.Challenge, solveCount int) error {
	if solvedChallenge.InitialValue <= 0 || solvedChallenge.Decay <= 0 {
		return nil
	}
	newPoints := competition.CalculateDynamicScore(solvedChallenge.InitialValue, solvedChallenge.MinValue, solvedChallenge.Decay, solveCount)
	if newPoints == solvedChallenge.Points {
		return nil
	}
	if err := uc.txRepo.UpdateChallengePointsTx(ctx, tx, challengeID, newPoints); err != nil {
		return usecaseutil.Wrap(err, "UpdateChallengePointsTx")
	}
	solvedChallenge.Points = newPoints
	return nil
}

func (uc *ChallengeUseCase) submitInvalidateCache(ctx context.Context) {
	if uc.scoreboardCache != nil {
		uc.scoreboardCache.InvalidateAll(ctx)
	}
}

func (uc *ChallengeUseCase) submitNotifySolve(teamID uuid.UUID, challenge *entity.Challenge, isFirstBlood bool) {
	if uc.broadcaster != nil && challenge != nil {
		uc.broadcaster.NotifySolve(teamID, challenge.Title, challenge.Points, isFirstBlood)
	}
}
