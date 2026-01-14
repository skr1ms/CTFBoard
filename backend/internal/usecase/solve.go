package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	pkgRedis "github.com/skr1ms/CTFBoard/pkg/redis"
)

type SolveUseCase struct {
	solveRepo repo.SolveRepository
	redis     pkgRedis.Client
}

func NewSolveUseCase(solveRepo repo.SolveRepository, redis pkgRedis.Client) *SolveUseCase {
	return &SolveUseCase{
		solveRepo: solveRepo,
		redis:     redis,
	}
}

func (uc *SolveUseCase) Create(ctx context.Context, solve *entity.Solve) error {
	_, err := uc.solveRepo.GetByTeamAndChallenge(ctx, solve.TeamId, solve.ChallengeId)
	if err == nil {
		return entityError.ErrAlreadySolved
	}
	if !errors.Is(err, entityError.ErrSolveNotFound) {
		return fmt.Errorf("SolveUseCase - Create - GetByTeamAndChallenge: %w", err)
	}

	err = uc.solveRepo.Create(ctx, solve)
	if err != nil {
		return fmt.Errorf("SolveUseCase - Create: %w", err)
	}

	uc.redis.Del(ctx, "scoreboard")

	return nil
}

func (uc *SolveUseCase) GetScoreboard(ctx context.Context) ([]*repo.ScoreboardEntry, error) {
	val, err := uc.redis.Get(ctx, "scoreboard").Result()
	if err == nil {
		var entries []*repo.ScoreboardEntry
		if err := json.Unmarshal([]byte(val), &entries); err == nil {
			return entries, nil
		}
	}

	entries, err := uc.solveRepo.GetScoreboard(ctx)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetScoreboard: %w", err)
	}

	if bytes, err := json.Marshal(entries); err == nil {
		uc.redis.Set(ctx, "scoreboard", bytes, 15*time.Second)
	}

	return entries, nil
}

func (uc *SolveUseCase) GetFirstBlood(ctx context.Context, challengeId string) (*repo.FirstBloodEntry, error) {
	entry, err := uc.solveRepo.GetFirstBlood(ctx, challengeId)
	if err != nil {
		return nil, fmt.Errorf("SolveUseCase - GetFirstBlood: %w", err)
	}
	return entry, nil
}
