package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type AwardUseCase struct {
	deps AwardDeps
}

type AwardDeps struct {
	AwardRepo       repo.AwardRepository
	TeamRepo        repo.TeamRepository
	TM              repo.TransactionManager
	ScoreboardCache cache.ScoreboardCacheInvalidator
	CompRepo        repo.CompetitionRepository
}

var _ usecase.AwardUseCase = (*AwardUseCase)(nil)

func NewAwardUseCase(deps AwardDeps) *AwardUseCase {
	return &AwardUseCase{deps: deps}
}

func (uc *AwardUseCase) Create(ctx context.Context, teamID uuid.UUID, value int, description string, createdBy uuid.UUID) (*entity.Award, error) {
	if teamID == uuid.Nil {
		return nil, httperr.ErrAwardTeamIDRequired
	}
	if value == 0 {
		return nil, httperr.ErrAwardValueCannotBeZero
	}

	award := &entity.Award{
		TeamID:      teamID,
		Value:       value,
		Description: description,
		CreatedBy:   &createdBy,
	}

	if err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if uc.deps.TeamRepo != nil {
			team, err := uc.deps.TeamRepo.GetByID(ctx, teamID)
			if err != nil {
				return fmt.Errorf("AwardUseCase - Create - TeamRepo.GetByID: %w", err)
			}
			if team.IsBanned {
				return httperr.ErrTeamBanned
			}
		}
		if err := uc.deps.AwardRepo.Create(ctx, award); err != nil {
			return fmt.Errorf("AwardUseCase - Create - AwardRepo.Create: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("AwardUseCase - Create - TM.Run: %w", err)
	}

	if uc.deps.ScoreboardCache != nil {
		if uc.deps.CompRepo != nil {
			comp, err := uc.deps.CompRepo.Get(ctx)
			if err == nil && comp != nil && comp.IsFreezeActive() {
				uc.deps.ScoreboardCache.InvalidateLiveOnly(ctx, teamID)
				return award, nil
			}
		}
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	}
	return award, nil
}

func (uc *AwardUseCase) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.Award, error) {
	awards, err := uc.deps.AwardRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("AwardUseCase - GetByTeamID - AwardRepo.GetByTeamID: %w", err)
	}
	return awards, nil
}

func (uc *AwardUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Award, error) {
	award, err := uc.deps.AwardRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("AwardUseCase - GetByID - AwardRepo.GetByID: %w", err)
	}
	return award, nil
}

func (uc *AwardUseCase) GetAll(ctx context.Context) ([]*entity.Award, error) {
	awards, err := uc.deps.AwardRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("AwardUseCase - GetAll - AwardRepo.GetAll: %w", err)
	}
	return awards, nil
}

func (uc *AwardUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	award, err := uc.deps.AwardRepo.GetByID(ctx, ID)
	if err != nil {
		return fmt.Errorf("AwardUseCase - Delete - AwardRepo.GetByID: %w", err)
	}
	if err := uc.deps.AwardRepo.Delete(ctx, ID); err != nil {
		return fmt.Errorf("AwardUseCase - Delete - AwardRepo.Delete: %w", err)
	}
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, award.TeamID)
	}
	return nil
}
