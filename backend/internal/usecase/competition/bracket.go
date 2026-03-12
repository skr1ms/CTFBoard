package competition

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type BracketUseCase struct {
	deps BracketDeps
}

type BracketDeps struct {
	BracketRepo repo.BracketRepository
	TM          repo.TransactionManager
}

var _ usecase.BracketUseCase = (*BracketUseCase)(nil)

func NewBracketUseCase(deps BracketDeps) *BracketUseCase {
	return &BracketUseCase{deps: deps}
}

func (uc *BracketUseCase) Create(ctx context.Context, name, description string, isDefault bool) (*entity.Bracket, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, httperr.ErrBracketNameRequired
	}
	bracket := &entity.Bracket{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		IsDefault:   isDefault,
	}
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		if isDefault {
			if err := uc.deps.BracketRepo.ClearAllDefaults(ctx); err != nil {
				return fmt.Errorf("BracketUseCase - Create - BracketRepo.ClearAllDefaults: %w", err)
			}
		}
		if err := uc.deps.BracketRepo.Create(ctx, bracket); err != nil {
			return fmt.Errorf("BracketUseCase - Create - BracketRepo.Create: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("BracketUseCase - Create - TM.Run: %w", err)
	}
	return bracket, nil
}

func (uc *BracketUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Bracket, error) {
	bracket, err := uc.deps.BracketRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("BracketUseCase - GetByID - BracketRepo.GetByID: %w", err)
	}
	return bracket, nil
}

func (uc *BracketUseCase) GetAll(ctx context.Context) ([]*entity.Bracket, error) {
	list, err := uc.deps.BracketRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("BracketUseCase - GetAll - BracketRepo.GetAll: %w", err)
	}
	return list, nil
}

func (uc *BracketUseCase) Update(ctx context.Context, ID uuid.UUID, name, description string, isDefault bool) (*entity.Bracket, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, httperr.ErrBracketNameRequired
	}
	var bracket *entity.Bracket
	err := uc.deps.TM.Run(ctx, func(ctx context.Context) error {
		var err error
		bracket, err = uc.deps.BracketRepo.GetByID(ctx, ID)
		if err != nil {
			return fmt.Errorf("BracketUseCase - Update - BracketRepo.GetByID: %w", err)
		}
		if isDefault {
			if err := uc.deps.BracketRepo.ClearAllDefaults(ctx); err != nil {
				return fmt.Errorf("BracketUseCase - Update - BracketRepo.ClearAllDefaults: %w", err)
			}
		}
		bracket.Name = name
		bracket.Description = description
		bracket.IsDefault = isDefault
		if err := uc.deps.BracketRepo.Update(ctx, bracket); err != nil {
			return fmt.Errorf("BracketUseCase - Update - BracketRepo.Update: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("BracketUseCase - Update - TM.Run: %w", err)
	}
	return bracket, nil
}

func (uc *BracketUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := uc.deps.BracketRepo.Delete(ctx, ID); err != nil {
		return fmt.Errorf("BracketUseCase - Delete - BracketRepo.Delete: %w", err)
	}
	return nil
}
