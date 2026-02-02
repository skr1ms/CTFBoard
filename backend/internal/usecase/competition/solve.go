package competition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	redisKeys "github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/skr1ms/CTFBoard/pkg/websocket"
)

type SolveUseCase struct {
	solveRepo       repo.SolveRepository
	challengeRepo   repo.ChallengeRepository
	competitionRepo repo.CompetitionRepository
	userRepo        repo.UserRepository
	txRepo          repo.TxRepository
	redis           *redis.Client
	hub             *websocket.Hub
}

func NewSolveUseCase(
	solveRepo repo.SolveRepository,
	challengeRepo repo.ChallengeRepository,
	competitionRepo repo.CompetitionRepository,
	userRepo repo.UserRepository,
	txRepo repo.TxRepository,
	redis *redis.Client,
	hub *websocket.Hub,
) *SolveUseCase {
	return &SolveUseCase{
		solveRepo:       solveRepo,
		challengeRepo:   challengeRepo,
		competitionRepo: competitionRepo,
		userRepo:        userRepo,
		txRepo:          txRepo,
		redis:           redis,
		hub:             hub,
	}
}

//nolint:gocognit,gocyclo,funlen
func (uc *SolveUseCase) Create(ctx context.Context, solve *entity.Solve) error {
	var isFirstBlood bool
	var solvedChallenge *entity.Challenge

	err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if solve.TeamID == uuid.Nil {
			if err := uc.txRepo.LockUserTx(ctx, tx, solve.UserID); err != nil {
				return fmt.Errorf("LockUserTx: %w", err)
			}
			user, err := uc.userRepo.GetByID(ctx, solve.UserID)
			if err != nil {
				return fmt.Errorf("GetByID: %w", err)
			}

			if user.TeamID == nil {
				return entityError.ErrNoTeamSelected
			}
			solve.TeamID = *user.TeamID
		}

		challenge, err := uc.txRepo.GetChallengeByIDTx(ctx, tx, solve.ChallengeID)
		if err != nil {
			return fmt.Errorf("GetChallengeByIDTx: %w", err)
		}

		existing, err := uc.txRepo.GetSolveByTeamAndChallengeTx(ctx, tx, solve.TeamID, solve.ChallengeID)
		if err == nil && existing != nil {
			return entityError.ErrAlreadySolved
		}

		if challenge.SolveCount == 0 {
			isFirstBlood = true
		}

		if err := uc.txRepo.CreateSolveTx(ctx, tx, solve); err != nil {
			return fmt.Errorf("CreateSolveTx: %w", err)
		}

		newCount, err := uc.txRepo.IncrementChallengeSolveCountTx(ctx, tx, solve.ChallengeID)
		if err != nil {
			return fmt.Errorf("IncrementChallengeSolveCountTx: %w", err)
		}

		if challenge.Decay > 0 {
			newPoints := CalculateDynamicScore(challenge.InitialValue, challenge.MinValue, challenge.Decay, newCount)
			if newPoints != challenge.Points {
				if err := uc.txRepo.UpdateChallengePointsTx(ctx, tx, challenge.ID, newPoints); err != nil {
					return fmt.Errorf("UpdateChallengePointsTx: %w", err)
				}
				challenge.Points = newPoints
			}
		}

		solvedChallenge = challenge
		return nil
	})
	if err != nil {
		if errors.Is(err, entityError.ErrAlreadySolved) || errors.Is(err, entityError.ErrNoTeamSelected) {
			return err
		}
		return fmt.Errorf("SolveUseCase - Create - Transaction: %w", err)
	}

	uc.redis.Del(ctx, redisKeys.KeyScoreboard)

	if uc.hub != nil && solvedChallenge != nil {
		payload := websocket.ScoreboardUpdate{
			Type:      websocket.EventTypeSolve,
			TeamID:    solve.TeamID.String(),
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
				TeamID:    solve.TeamID.String(),
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

	cacheKey, frozen := uc.getScoreboardCacheKey(comp)

	if entries := uc.tryGetCachedScoreboard(ctx, cacheKey); entries != nil {
		return entries, nil
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

func (uc *SolveUseCase) getScoreboardCacheKey(comp *entity.Competition) (string, bool) {
	if comp != nil && comp.FreezeTime != nil && time.Now().After(*comp.FreezeTime) {
		return redisKeys.KeyScoreboardFrozen, true
	}
	return redisKeys.KeyScoreboard, false
}

func (uc *SolveUseCase) tryGetCachedScoreboard(ctx context.Context, cacheKey string) []*repo.ScoreboardEntry {
	val, err := uc.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil
	}

	var entries []*repo.ScoreboardEntry
	if err := json.Unmarshal([]byte(val), &entries); err == nil {
		return entries
	}
	return nil
}

func (uc *SolveUseCase) GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*repo.FirstBloodEntry, error) {
	entry, err := uc.solveRepo.GetFirstBlood(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetFirstBlood: %w", err)
	}
	return entry, nil
}
