package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type AwardUseCase struct {
	repo  repo.AwardRepository
	redis *redis.Client
}

func NewAwardUseCase(repo repo.AwardRepository, redis *redis.Client) *AwardUseCase {
	return &AwardUseCase{
		repo:  repo,
		redis: redis,
	}
}

func (uc *AwardUseCase) Create(ctx context.Context, teamID uuid.UUID, value int, description string, createdBy uuid.UUID) (*entity.Award, error) {
	if value == 0 {
		return nil, fmt.Errorf("value cannot be 0")
	}

	award := &entity.Award{
		TeamID:      teamID,
		Value:       value,
		Description: description,
		CreatedBy:   &createdBy,
	}

	if err := uc.repo.Create(ctx, award); err != nil {
		return nil, fmt.Errorf("AwardUseCase - Create: %w", err)
	}

	uc.redis.Del(ctx, "scoreboard")
	uc.redis.Del(ctx, "scoreboard:frozen")

	return award, nil
}

func (uc *AwardUseCase) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error) {
	awards, err := uc.repo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("AwardUseCase - GetByTeamID: %w", err)
	}
	return awards, nil
}
