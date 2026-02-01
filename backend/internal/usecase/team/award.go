package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type AwardUseCase struct {
	awardRepo repo.AwardRepository
	txRepo    repo.TxRepository
	redis     *redis.Client
}

func NewAwardUseCase(
	awardRepo repo.AwardRepository,
	txRepo repo.TxRepository,
	redis *redis.Client,
) *AwardUseCase {
	return &AwardUseCase{
		awardRepo: awardRepo,
		txRepo:    txRepo,
		redis:     redis,
	}
}

func (uc *AwardUseCase) Create(ctx context.Context, teamID uuid.UUID, value int, description string, createdBy uuid.UUID) (*entity.Award, error) {
	if value == 0 {
		return nil, fmt.Errorf("AwardUseCase - Create: value cannot be 0")
	}

	award := &entity.Award{
		TeamID:      teamID,
		Value:       value,
		Description: description,
		CreatedBy:   &createdBy,
	}

	if err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return uc.txRepo.CreateAwardTx(ctx, tx, award)
	}); err != nil {
		return nil, fmt.Errorf("AwardUseCase - Create: %w", err)
	}

	uc.redis.Del(ctx, "scoreboard")
	uc.redis.Del(ctx, "scoreboard:frozen")

	return award, nil
}

func (uc *AwardUseCase) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error) {
	awards, err := uc.awardRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("AwardUseCase - GetByTeamID: %w", err)
	}
	return awards, nil
}
