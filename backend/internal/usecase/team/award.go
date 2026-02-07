package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/cache"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
)

type AwardUseCase struct {
	awardRepo       repo.AwardRepository
	txRepo          repo.TxRepository
	scoreboardCache cache.ScoreboardCacheInvalidator
}

func NewAwardUseCase(
	awardRepo repo.AwardRepository,
	txRepo repo.TxRepository,
	scoreboardCache cache.ScoreboardCacheInvalidator,
) *AwardUseCase {
	return &AwardUseCase{
		awardRepo:       awardRepo,
		txRepo:          txRepo,
		scoreboardCache: scoreboardCache,
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

	if err := uc.txRepo.RunTransaction(ctx, func(ctx context.Context, tx repo.Transaction) error {
		return uc.txRepo.CreateAwardTx(ctx, tx, award)
	}); err != nil {
		return nil, usecaseutil.Wrap(err, "AwardUseCase - Create")
	}

	if uc.scoreboardCache != nil {
		uc.scoreboardCache.InvalidateAll(ctx)
	}
	return award, nil
}

func (uc *AwardUseCase) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error) {
	awards, err := uc.awardRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "AwardUseCase - GetByTeamID")
	}
	return awards, nil
}
