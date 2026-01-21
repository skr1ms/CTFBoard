package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	pkgRedis "github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/skr1ms/CTFBoard/pkg/websocket"
)

type SolveUseCase struct {
	solveRepo       repo.SolveRepository
	challengeRepo   repo.ChallengeRepository
	competitionRepo repo.CompetitionRepository
	txRepo          repo.TxRepository
	redis           pkgRedis.Client
	hub             *websocket.Hub
}

func NewSolveUseCase(
	solveRepo repo.SolveRepository,
	challengeRepo repo.ChallengeRepository,
	competitionRepo repo.CompetitionRepository,
	txRepo repo.TxRepository,
	redis pkgRedis.Client,
	hub *websocket.Hub,
) *SolveUseCase {
	return &SolveUseCase{
		solveRepo:       solveRepo,
		challengeRepo:   challengeRepo,
		competitionRepo: competitionRepo,
		txRepo:          txRepo,
		redis:           redis,
		hub:             hub,
	}
}

func (uc *SolveUseCase) Create(ctx context.Context, solve *entity.Solve) error {
	var isFirstBlood bool
	var solvedChallenge *entity.Challenge

	err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		challenge, err := uc.txRepo.GetChallengeByIDTx(ctx, tx, solve.ChallengeId)
		if err != nil {
			return fmt.Errorf("GetChallengeByIDTx: %w", err)
		}

		existing, err := uc.txRepo.GetSolveByTeamAndChallengeTx(ctx, tx, solve.TeamId, solve.ChallengeId)
		if err == nil && existing != nil {
			return entityError.ErrAlreadySolved
		}

		if challenge.SolveCount == 0 {
			isFirstBlood = true
		}

		if err := uc.txRepo.CreateSolveTx(ctx, tx, solve); err != nil {
			return fmt.Errorf("CreateSolveTx: %w", err)
		}

		newCount, err := uc.txRepo.IncrementChallengeSolveCountTx(ctx, tx, solve.ChallengeId)
		if err != nil {
			return fmt.Errorf("IncrementChallengeSolveCountTx: %w", err)
		}

		if challenge.Decay > 0 {
			newPoints := CalculateDynamicScore(challenge.InitialValue, challenge.MinValue, challenge.Decay, newCount)
			if newPoints != challenge.Points {
				if err := uc.txRepo.UpdateChallengePointsTx(ctx, tx, challenge.Id, newPoints); err != nil {
					return fmt.Errorf("UpdateChallengePointsTx: %w", err)
				}
				challenge.Points = newPoints
			}
		}

		solvedChallenge = challenge
		return nil
	})

	if err != nil {
		if errors.Is(err, entityError.ErrAlreadySolved) {
			return err
		}
		return fmt.Errorf("SolveUseCase - Create - Transaction: %w", err)
	}

	uc.redis.Del(ctx, "scoreboard")

	if uc.hub != nil && solvedChallenge != nil {
		payload := websocket.ScoreboardUpdate{
			Type:      websocket.EventTypeSolve,
			TeamID:    solve.TeamId.String(),
			Challenge: solvedChallenge.Title,
			Points:    solvedChallenge.Points,
			Timestamp: time.Now(),
		}
		uc.hub.BroadcastEvent(websocket.Event{
			Type:      "scoreboard_update",
			Payload:   payload,
			Timestamp: time.Now(),
		})

		if isFirstBlood {
			fbPayload := websocket.ScoreboardUpdate{
				Type:      websocket.EventTypeFirstBlood,
				TeamID:    solve.TeamId.String(),
				Challenge: solvedChallenge.Title,
				Points:    solvedChallenge.Points,
				Timestamp: time.Now(),
			}
			uc.hub.BroadcastEvent(websocket.Event{
				Type:      "scoreboard_update",
				Payload:   fbPayload,
				Timestamp: time.Now(),
			})
		}
	}

	return nil
}

func (uc *SolveUseCase) GetScoreboard(ctx context.Context) ([]*repo.ScoreboardEntry, error) {
	comp, err := uc.competitionRepo.Get(ctx)
	if err != nil && !errors.Is(err, entityError.ErrCompetitionNotFound) {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard - GetCompetition: %w", err)
	}

	var cacheKey string
	var frozen bool
	if comp != nil && comp.FreezeTime != nil && time.Now().After(*comp.FreezeTime) {
		cacheKey = "scoreboard:frozen"
		frozen = true
	} else {
		cacheKey = "scoreboard"
		frozen = false
	}

	val, err := uc.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var entries []*repo.ScoreboardEntry
		if err := json.Unmarshal([]byte(val), &entries); err == nil {
			return entries, nil
		}
	}

	var entries []*repo.ScoreboardEntry
	if frozen {
		entries, err = uc.solveRepo.GetScoreboardFrozen(ctx, *comp.FreezeTime)
	} else {
		entries, err = uc.solveRepo.GetScoreboard(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard: %w", err)
	}

	if bytes, err := json.Marshal(entries); err == nil {
		uc.redis.Set(ctx, cacheKey, bytes, 15*time.Second)
	}

	return entries, nil
}

func (uc *SolveUseCase) GetFirstBlood(ctx context.Context, challengeId uuid.UUID) (*repo.FirstBloodEntry, error) {
	entry, err := uc.solveRepo.GetFirstBlood(ctx, challengeId)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetFirstBlood: %w", err)
	}
	return entry, nil
}
