package competition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/cache"
	redisKeys "github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/skr1ms/CTFBoard/pkg/websocket"
)

type SolveUseCase struct {
	solveRepo       repo.SolveRepository
	challengeRepo   repo.ChallengeRepository
	competitionRepo repo.CompetitionRepository
	userRepo        repo.UserRepository
	teamRepo        repo.TeamRepository
	txRepo          repo.TxRepository
	cache           *cache.Cache
	broadcaster     *websocket.Broadcaster
}

func NewSolveUseCase(
	solveRepo repo.SolveRepository,
	challengeRepo repo.ChallengeRepository,
	competitionRepo repo.CompetitionRepository,
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	txRepo repo.TxRepository,
	c *cache.Cache,
	broadcaster *websocket.Broadcaster,
) *SolveUseCase {
	return &SolveUseCase{
		solveRepo:       solveRepo,
		challengeRepo:   challengeRepo,
		competitionRepo: competitionRepo,
		userRepo:        userRepo,
		teamRepo:        teamRepo,
		txRepo:          txRepo,
		cache:           c,
		broadcaster:     broadcaster,
	}
}

//nolint:gocognit,gocyclo
func (uc *SolveUseCase) Create(ctx context.Context, solve *entity.Solve) error {
	var isFirstBlood bool
	var solvedChallenge *entity.Challenge

	err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if solve.TeamID == uuid.Nil {
			if err := uc.txRepo.LockUserTx(ctx, tx, solve.UserID); err != nil {
				return fmt.Errorf("SolveUseCase - Create - LockUserTx: %w", err)
			}
			user, err := uc.userRepo.GetByID(ctx, solve.UserID)
			if err != nil {
				return fmt.Errorf("SolveUseCase - Create - GetByID: %w", err)
			}

			if user.TeamID == nil {
				return entityError.ErrNoTeamSelected
			}
			solve.TeamID = *user.TeamID
		}

		challenge, err := uc.txRepo.GetChallengeByIDTx(ctx, tx, solve.ChallengeID)
		if err != nil {
			return fmt.Errorf("SolveUseCase - Create - GetChallengeByIDTx: %w", err)
		}

		existing, err := uc.txRepo.GetSolveByTeamAndChallengeTx(ctx, tx, solve.TeamID, solve.ChallengeID)
		if err == nil && existing != nil {
			return entityError.ErrAlreadySolved
		}

		if challenge.SolveCount == 0 {
			isFirstBlood = true
		}

		if err := uc.txRepo.CreateSolveTx(ctx, tx, solve); err != nil {
			return fmt.Errorf("SolveUseCase - Create - CreateSolveTx: %w", err)
		}

		newCount, err := uc.txRepo.IncrementChallengeSolveCountTx(ctx, tx, solve.ChallengeID)
		if err != nil {
			return fmt.Errorf("SolveUseCase - Create - IncrementChallengeSolveCountTx: %w", err)
		}

		if challenge.Decay > 0 {
			newPoints := CalculateDynamicScore(challenge.InitialValue, challenge.MinValue, challenge.Decay, newCount)
			if newPoints != challenge.Points {
				if err := uc.txRepo.UpdateChallengePointsTx(ctx, tx, challenge.ID, newPoints); err != nil {
					return fmt.Errorf("SolveUseCase - Create - UpdateChallengePointsTx: %w", err)
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

	uc.invalidateScoreboardCache(ctx, solve.TeamID)

	if solvedChallenge != nil {
		uc.broadcaster.NotifySolve(solve.TeamID, solvedChallenge.Title, solvedChallenge.Points, isFirstBlood)
	}

	return nil
}

func (uc *SolveUseCase) GetScoreboard(ctx context.Context, bracketID *uuid.UUID) ([]*repo.ScoreboardEntry, error) {
	comp, err := uc.competitionRepo.Get(ctx)
	if err != nil && !errors.Is(err, entityError.ErrCompetitionNotFound) {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard - GetCompetition: %w", err)
	}

	cacheKey, frozen := uc.getScoreboardCacheKey(comp, bracketID)

	return cache.GetOrLoad(uc.cache, ctx, cacheKey, 15*time.Second, func() ([]*repo.ScoreboardEntry, error) {
		var entries []*repo.ScoreboardEntry
		if frozen {
			if comp != nil && comp.FreezeTime != nil {
				entries, err = uc.solveRepo.GetScoreboardByBracketFrozen(ctx, *comp.FreezeTime, bracketID)
			} else {
				entries, err = uc.solveRepo.GetScoreboardByBracketFrozen(ctx, time.Time{}, bracketID)
			}
		} else {
			entries, err = uc.solveRepo.GetScoreboardByBracket(ctx, bracketID)
		}
		if err != nil {
			return nil, fmt.Errorf("SolveUseCase - GetScoreboard: %w", err)
		}
		return entries, nil
	})
}

func (uc *SolveUseCase) getScoreboardCacheKey(comp *entity.Competition, bracketID *uuid.UUID) (string, bool) {
	frozen := comp != nil && comp.FreezeTime != nil && time.Now().After(*comp.FreezeTime)
	if bracketID == nil || *bracketID == uuid.Nil {
		if frozen {
			return redisKeys.KeyScoreboardFrozen, true
		}
		return redisKeys.KeyScoreboard, false
	}
	idStr := bracketID.String()
	if frozen {
		return redisKeys.KeyScoreboardBracketFrozen(idStr), true
	}
	return redisKeys.KeyScoreboardBracket(idStr), false
}

func (uc *SolveUseCase) invalidateScoreboardCache(ctx context.Context, teamID uuid.UUID) {
	uc.cache.Del(ctx, redisKeys.KeyScoreboard, redisKeys.KeyScoreboardFrozen)
	team, err := uc.teamRepo.GetByID(ctx, teamID)
	if err != nil || team == nil {
		return
	}
	if team.BracketID != nil {
		idStr := team.BracketID.String()
		uc.cache.Del(ctx, redisKeys.KeyScoreboardBracket(idStr), redisKeys.KeyScoreboardBracketFrozen(idStr))
	}
}

func (uc *SolveUseCase) GetFirstBlood(ctx context.Context, challengeID uuid.UUID) (*repo.FirstBloodEntry, error) {
	entry, err := uc.solveRepo.GetFirstBlood(ctx, challengeID)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetFirstBlood: %w", err)
	}
	return entry, nil
}
