package team

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
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

func (uc *AwardUseCase) Create(ctx context.Context, teamID uuid.UUID, value int, description string, createdBy uuid.UUID) (*domain.Award, error) {
	if teamID == uuid.Nil {
		return nil, apperr.ErrAwardTeamIDRequired
	}

	if value == 0 {
		return nil, apperr.ErrAwardValueCannotBeZero
	}

	award := &domain.Award{
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
				return apperr.ErrTeamBanned
			}
		}

		if err := uc.deps.AwardRepo.Create(ctx, award); err != nil {
			return fmt.Errorf("AwardUseCase - Create - AwardRepo.Create: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("AwardUseCase - Create - TM.Run: %w", err)
	}

	var comp *domain.Competition

	if uc.deps.CompRepo != nil {
		comp, _ = uc.deps.CompRepo.Get(ctx)
	}

	frozen := comp != nil && comp.IsFreezeActive()
	cache.InvalidateWithFreezeAwareness(ctx, uc.deps.ScoreboardCache, teamID, frozen)

	return award, nil
}

func (uc *AwardUseCase) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error) {
	awards, err := uc.deps.AwardRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("AwardUseCase - GetByTeamID - AwardRepo.GetByTeamID: %w", err)
	}

	return awards, nil
}

func (uc *AwardUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Award, error) {
	award, err := uc.deps.AwardRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("AwardUseCase - GetByID - AwardRepo.GetByID: %w", err)
	}

	return award, nil
}

func (uc *AwardUseCase) GetAll(ctx context.Context) ([]*domain.Award, error) {
	awards, err := uc.deps.AwardRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("AwardUseCase - GetAll - AwardRepo.GetAll: %w", err)
	}

	return awards, nil
}

func (uc *AwardUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	award, err := uc.deps.AwardRepo.GetByID(ctx, ID)
	if err != nil {
		if errors.Is(err, apperr.ErrAwardNotFound) {
			return nil
		}

		return fmt.Errorf("AwardUseCase - Delete - AwardRepo.GetByID: %w", err)
	}

	if err := uc.deps.AwardRepo.Delete(ctx, ID); err != nil {
		return fmt.Errorf("AwardUseCase - Delete - AwardRepo.Delete: %w", err)
	}

	var comp *domain.Competition

	if uc.deps.CompRepo != nil {
		comp, _ = uc.deps.CompRepo.Get(ctx)
	}

	frozen := comp != nil && comp.IsFreezeActive()
	cache.InvalidateWithFreezeAwareness(ctx, uc.deps.ScoreboardCache, award.TeamID, frozen)

	return nil
}
