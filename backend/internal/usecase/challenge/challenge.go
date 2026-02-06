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
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition"
	"github.com/skr1ms/CTFBoard/pkg/crypto"
	redisKeys "github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/skr1ms/CTFBoard/pkg/websocket"
	"golang.org/x/sync/singleflight"
)

type ChallengeUseCase struct {
	challengeRepo repo.ChallengeRepository
	tagRepo       repo.TagRepository
	solveRepo     repo.SolveRepository
	txRepo        repo.TxRepository
	compRepo      repo.CompetitionRepository
	teamRepo      repo.TeamRepository
	redis         *redis.Client
	broadcaster   *websocket.Broadcaster
	auditLogRepo  repo.AuditLogRepository
	crypto        crypto.Service
	regexCache    sync.Map
	regexSf       singleflight.Group
}

func NewChallengeUseCase(challengeRepo repo.ChallengeRepository, opts ...ChallengeUCOption) *ChallengeUseCase {
	uc := &ChallengeUseCase{
		challengeRepo: challengeRepo,
	}
	for _, opt := range opts {
		opt(uc)
	}
	return uc
}

func (uc *ChallengeUseCase) GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*usecase.ChallengeWithTags, error) {
	challenges, err := uc.challengeRepo.GetAll(ctx, teamID, tagID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll: %w", err)
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
		return nil, fmt.Errorf("ChallengeUseCase - GetAll - GetTags: %w", err)
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
			return nil, fmt.Errorf("ChallengeUseCase - Create: %w", crypto.ErrServiceNotConfigured)
		}
		encrypted, err := uc.crypto.Encrypt(flag)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - Create - Encrypt: %w", err)
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
		return nil, fmt.Errorf("ChallengeUseCase - Create: %w", err)
	}
	if uc.tagRepo != nil && len(tagIDs) > 0 {
		if err := uc.tagRepo.SetChallengeTags(ctx, challenge.ID, tagIDs); err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - Create - SetTags: %w", err)
		}
	}
	return challenge, nil
}

//nolint:gocognit // many branches for flag and tags
func (uc *ChallengeUseCase) Update(ctx context.Context, ID uuid.UUID, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool, flagFormatRegex *string, tagIDs []uuid.UUID) (*entity.Challenge, error) {
	challenge, err := uc.challengeRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update - GetByID: %w", err)
	}

	challenge.Title = title
	challenge.Description = description
	challenge.Category = category
	challenge.Points = points
	challenge.InitialValue = initialValue
	challenge.MinValue = minValue
	challenge.Decay = decay
	challenge.IsHidden = isHidden
	challenge.IsRegex = isRegex
	challenge.IsCaseInsensitive = isCaseInsensitive
	challenge.FlagFormatRegex = flagFormatRegex

	if flag != "" {
		if isRegex {
			if uc.crypto == nil {
				return nil, fmt.Errorf("ChallengeUseCase - Update: %w", crypto.ErrServiceNotConfigured)
			}
			encrypted, err := uc.crypto.Encrypt(flag)
			if err != nil {
				return nil, fmt.Errorf("ChallengeUseCase - Update - Encrypt: %w", err)
			}
			challenge.FlagRegex = encrypted
			challenge.FlagHash = "REGEX_CHALLENGE"
		} else {
			userInput := flag
			if isCaseInsensitive {
				userInput = strings.ToLower(strings.TrimSpace(flag))
			}
			hash := sha256.Sum256([]byte(userInput))
			challenge.FlagHash = hex.EncodeToString(hash[:])
			challenge.FlagRegex = ""
		}
	}

	err = uc.challengeRepo.Update(ctx, challenge)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Update: %w", err)
	}
	if uc.tagRepo != nil {
		if err := uc.tagRepo.SetChallengeTags(ctx, ID, tagIDs); err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - Update - SetTags: %w", err)
		}
	}

	uc.redis.Del(ctx, redisKeys.KeyScoreboard)

	return challenge, nil
}

func (uc *ChallengeUseCase) Delete(ctx context.Context, ID, actorID uuid.UUID, clientIP string) error {
	err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := uc.txRepo.GetChallengeByIDTx(ctx, tx, ID); err != nil {
			return err
		}

		if err := uc.txRepo.DeleteChallengeTx(ctx, tx, ID); err != nil {
			return fmt.Errorf("DeleteChallengeTx: %w", err)
		}

		auditLog := &entity.AuditLog{
			UserID:     &actorID,
			Action:     entity.AuditActionDelete,
			EntityType: entity.AuditEntityChallenge,
			EntityID:   ID.String(),
			IP:         clientIP,
		}
		if err := uc.txRepo.CreateAuditLogTx(ctx, tx, auditLog); err != nil {
			return fmt.Errorf("CreateAuditLogTx: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("ChallengeUseCase - Delete - Transaction: %w", err)
	}

	uc.redis.Del(ctx, redisKeys.KeyScoreboard)

	return nil
}

func (uc *ChallengeUseCase) getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	if v, ok := uc.regexCache.Load(pattern); ok {
		if re, ok := v.(*regexp.Regexp); ok {
			return re, nil
		}
		return nil, fmt.Errorf("ChallengeUseCase - getCompiledRegex: invalid cached regex type")
	}
	v, err, _ := uc.regexSf.Do(pattern, func() (any, error) {
		if v, ok := uc.regexCache.Load(pattern); ok {
			if re, ok := v.(*regexp.Regexp); ok {
				return re, nil
			}
			return nil, fmt.Errorf("ChallengeUseCase - getCompiledRegex: invalid cached regex type")
		}
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("ChallengeUseCase - getCompiledRegex - Compile: %w", err)
		}
		uc.regexCache.Store(pattern, compiled)
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
		return fmt.Errorf("ChallengeUseCase - SubmitFlag - GetTeam: %w", err)
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
		return nil, fmt.Errorf("ChallengeUseCase - SubmitFlag - GetByID: %w", err)
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
		return fmt.Errorf("ChallengeUseCase - SubmitFlag - MatchString: %w", err)
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

//nolint:gocognit
func (uc *ChallengeUseCase) submitRecordSolve(sc *submitContext, _ *entity.Challenge) (*entity.Challenge, int, error) {
	var solveCount int
	var solvedChallenge *entity.Challenge
	err := uc.txRepo.RunTransaction(sc.ctx, func(ctx context.Context, tx pgx.Tx) error {
		_, err := uc.txRepo.GetSolveByTeamAndChallengeTx(ctx, tx, sc.teamID, sc.challengeID)
		if err == nil {
			return entityError.ErrAlreadySolved
		}
		if !errors.Is(err, entityError.ErrSolveNotFound) {
			return fmt.Errorf("GetSolveByTeamAndChallengeTx: %w", err)
		}
		var err2 error
		solvedChallenge, err2 = uc.txRepo.GetChallengeByIDTx(ctx, tx, sc.challengeID)
		if err2 != nil {
			return fmt.Errorf("GetChallengeByIDTx: %w", err2)
		}
		solve := &entity.Solve{
			UserID:      sc.userID,
			TeamID:      sc.teamID,
			ChallengeID: sc.challengeID,
		}
		if err2 = uc.txRepo.CreateSolveTx(ctx, tx, solve); err2 != nil {
			return fmt.Errorf("CreateSolveTx: %w", err2)
		}
		solveCount, err2 = uc.txRepo.IncrementChallengeSolveCountTx(ctx, tx, sc.challengeID)
		if err2 != nil {
			return fmt.Errorf("IncrementChallengeSolveCountTx: %w", err2)
		}
		if solvedChallenge.InitialValue > 0 && solvedChallenge.Decay > 0 {
			newPoints := competition.CalculateDynamicScore(solvedChallenge.InitialValue, solvedChallenge.MinValue, solvedChallenge.Decay, solveCount)
			if newPoints != solvedChallenge.Points {
				if err2 = uc.txRepo.UpdateChallengePointsTx(ctx, tx, sc.challengeID, newPoints); err2 != nil {
					return fmt.Errorf("UpdateChallengePointsTx: %w", err2)
				}
				solvedChallenge.Points = newPoints
			}
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return solvedChallenge, solveCount, nil
}

func (uc *ChallengeUseCase) submitInvalidateCache(ctx context.Context) {
	if uc.redis != nil {
		uc.redis.Del(ctx, redisKeys.KeyScoreboard)
	}
}

func (uc *ChallengeUseCase) submitNotifySolve(teamID uuid.UUID, challenge *entity.Challenge, isFirstBlood bool) {
	if uc.broadcaster != nil && challenge != nil {
		uc.broadcaster.NotifySolve(teamID, challenge.Title, challenge.Points, isFirstBlood)
	}
}
