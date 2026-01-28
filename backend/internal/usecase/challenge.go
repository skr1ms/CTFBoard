package usecase

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
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/crypto"
	"github.com/skr1ms/CTFBoard/pkg/websocket"
)

type ChallengeUseCase struct {
	challengeRepo repo.ChallengeRepository
	solveRepo     repo.SolveRepository
	txRepo        repo.TxRepository
	compRepo      repo.CompetitionRepository
	redis         *redis.Client
	hub           *websocket.Hub
	auditLogRepo  repo.AuditLogRepository
	crypto        crypto.Service
	regexCache    sync.Map
}

func NewChallengeUseCase(
	challengeRepo repo.ChallengeRepository,
	solveRepo repo.SolveRepository,
	txRepo repo.TxRepository,
	compRepo repo.CompetitionRepository,
	redis *redis.Client,
	hub *websocket.Hub,
	auditLogRepo repo.AuditLogRepository,
	crypto crypto.Service,
) *ChallengeUseCase {
	return &ChallengeUseCase{
		challengeRepo: challengeRepo,
		solveRepo:     solveRepo,
		txRepo:        txRepo,
		compRepo:      compRepo,
		redis:         redis,
		hub:           hub,
		auditLogRepo:  auditLogRepo,
		crypto:        crypto,
	}
}

func (uc *ChallengeUseCase) GetAll(ctx context.Context, teamId *uuid.UUID) ([]*repo.ChallengeWithSolved, error) {
	challenges, err := uc.challengeRepo.GetAll(ctx, teamId)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - GetAll: %w", err)
	}
	return challenges, nil
}

func (uc *ChallengeUseCase) Create(ctx context.Context, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool) (*entity.Challenge, error) {
	var flagHash string
	var flagRegex string

	if isRegex {
		if uc.crypto == nil {
			return nil, errors.New("encryption service not configured")
		}
		encrypted, err := uc.crypto.Encrypt(flag)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt regex flag: %w", err)
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
	}

	err := uc.challengeRepo.Create(ctx, challenge)
	if err != nil {
		return nil, fmt.Errorf("ChallengeUseCase - Create: %w", err)
	}
	return challenge, nil
}

func (uc *ChallengeUseCase) Update(ctx context.Context, id uuid.UUID, title, description, category string, points, initialValue, minValue, decay int, flag string, isHidden, isRegex, isCaseInsensitive bool) (*entity.Challenge, error) {
	challenge, err := uc.challengeRepo.GetByID(ctx, id)
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

	if flag != "" {
		if isRegex {
			if uc.crypto == nil {
				return nil, errors.New("encryption service not configured")
			}
			encrypted, err := uc.crypto.Encrypt(flag)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt regex flag: %w", err)
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

	uc.redis.Del(ctx, "scoreboard")

	return challenge, nil
}

func (uc *ChallengeUseCase) Delete(ctx context.Context, id uuid.UUID, actorId uuid.UUID, clientIP string) error {
	err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := uc.challengeRepo.GetByID(ctx, id); err != nil {
			return err
		}

		if err := uc.challengeRepo.Delete(ctx, id); err != nil {
			return fmt.Errorf("Delete: %w", err)
		}

		auditLog := &entity.AuditLog{
			UserId:     &actorId,
			Action:     entity.AuditActionDelete,
			EntityType: entity.AuditEntityChallenge,
			EntityId:   id.String(),
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

	uc.redis.Del(ctx, "scoreboard")

	return nil
}

func (uc *ChallengeUseCase) SubmitFlag(ctx context.Context, challengeId uuid.UUID, flag string, userId uuid.UUID, teamId *uuid.UUID) (bool, error) {
	if teamId == nil {
		return false, entityError.ErrUserMustBeInTeam
	}

	if uc.compRepo != nil {
		comp, err := uc.compRepo.Get(ctx)
		if err == nil && comp.FlagRegex != nil && *comp.FlagRegex != "" {
			matched, _ := regexp.MatchString(*comp.FlagRegex, flag)
			if !matched {
				return false, entityError.ErrInvalidFlagFormat
			}
		}
	}

	challenge, err := uc.challengeRepo.GetByID(ctx, challengeId)
	if err != nil {
		if errors.Is(err, entityError.ErrChallengeNotFound) {
			return false, entityError.ErrChallengeNotFound
		}
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - GetByID: %w", err)
	}

	var isValid bool

	flag = strings.TrimSpace(flag)

	if challenge.IsRegex {
		if uc.crypto == nil {
			return false, errors.New("encryption service not configured")
		}
		pattern, err := uc.crypto.Decrypt(challenge.FlagRegex)
		if err != nil {
			return false, nil
		}

		if challenge.IsCaseInsensitive {
			pattern = "(?i)" + pattern
		}

		var compiledRegex *regexp.Regexp
		if v, ok := uc.regexCache.Load(pattern); ok {
			compiledRegex = v.(*regexp.Regexp)
		} else {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				return false, fmt.Errorf("failed to compile regex: %w", err)
			}
			uc.regexCache.Store(pattern, compiled)
			compiledRegex = compiled
		}

		if compiledRegex.MatchString(flag) {
			isValid = true
		}
	} else {
		userInput := flag
		if challenge.IsCaseInsensitive {
			userInput = strings.ToLower(userInput)
		}

		hash := sha256.Sum256([]byte(userInput))
		hashStr := hex.EncodeToString(hash[:])
		if subtle.ConstantTimeCompare([]byte(hashStr), []byte(challenge.FlagHash)) == 1 {
			isValid = true
		}
	}

	if !isValid {
		return false, nil
	}

	tx, err := uc.txRepo.BeginTx(ctx)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - BeginTx: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	_, err = uc.txRepo.GetSolveByTeamAndChallengeTx(ctx, tx, *teamId, challengeId)
	if err == nil {
		err = entityError.ErrAlreadySolved
		return true, err
	}
	if !errors.Is(err, entityError.ErrSolveNotFound) {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - GetByTeamAndChallengeTx: %w", err)
	}

	challenge, err = uc.txRepo.GetChallengeByIDTx(ctx, tx, challengeId)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - GetChallengeByIDTx: %w", err)
	}

	solve := &entity.Solve{
		UserId:      userId,
		TeamId:      *teamId,
		ChallengeId: challengeId,
	}

	err = uc.txRepo.CreateSolveTx(ctx, tx, solve)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return true, entityError.ErrAlreadySolved
		}
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - CreateTx: %w", err)
	}

	solveCount, err := uc.txRepo.IncrementChallengeSolveCountTx(ctx, tx, challengeId)
	if err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - IncrementSolveCountTx: %w", err)
	}

	if challenge.InitialValue > 0 && challenge.Decay > 0 {
		newPoints := CalculateDynamicScore(challenge.InitialValue, challenge.MinValue, challenge.Decay, solveCount)
		if err = uc.txRepo.UpdateChallengePointsTx(ctx, tx, challengeId, newPoints); err != nil {
			return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - UpdatePointsTx: %w", err)
		}
		challenge.Points = newPoints
	}

	if err = tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("ChallengeUseCase - SubmitFlag - Commit: %w", err)
	}

	uc.redis.Del(ctx, "scoreboard")

	if uc.hub != nil {
		payload := websocket.ScoreboardUpdate{
			Type:      websocket.EventTypeSolve,
			TeamID:    teamId.String(),
			Challenge: challenge.Title,
			Points:    challenge.Points,
			Timestamp: time.Now(),
		}
		uc.hub.BroadcastEvent(websocket.Event{
			Type:      "scoreboard_update",
			Payload:   payload,
			Timestamp: time.Now(),
		})

		if solveCount == 1 {
			fbPayload := websocket.ScoreboardUpdate{
				Type:      websocket.EventTypeFirstBlood,
				TeamID:    teamId.String(),
				Challenge: challenge.Title,
				Points:    challenge.Points,
				Timestamp: time.Now(),
			}
			uc.hub.BroadcastEvent(websocket.Event{
				Type:      "scoreboard_update",
				Payload:   fbPayload,
				Timestamp: time.Now(),
			})
		}
	}

	return true, nil
}
