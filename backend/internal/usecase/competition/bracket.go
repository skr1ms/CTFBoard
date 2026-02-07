package competition

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
)

type BracketUseCase struct {
	bracketRepo repo.BracketRepository
}

func NewBracketUseCase(
	bracketRepo repo.BracketRepository,
) *BracketUseCase {
	return &BracketUseCase{bracketRepo: bracketRepo}
}

func (uc *BracketUseCase) Create(ctx context.Context, name, description string, isDefault bool) (*entity.Bracket, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("BracketUseCase - Create: name is required")
	}
	bracket := &entity.Bracket{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		IsDefault:   isDefault,
	}
	if err := uc.bracketRepo.Create(ctx, bracket); err != nil {
		return nil, usecaseutil.Wrap(err, "BracketUseCase - Create")
	}
	return bracket, nil
}

func (uc *BracketUseCase) GetByID(ctx context.Context, id uuid.UUID) (*entity.Bracket, error) {
	bracket, err := uc.bracketRepo.GetByID(ctx, id)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "BracketUseCase - GetByID")
	}
	return bracket, nil
}

func (uc *BracketUseCase) GetAll(ctx context.Context) ([]*entity.Bracket, error) {
	list, err := uc.bracketRepo.GetAll(ctx)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "BracketUseCase - GetAll")
	}
	return list, nil
}

func (uc *BracketUseCase) Update(ctx context.Context, id uuid.UUID, name, description string, isDefault bool) (*entity.Bracket, error) {
	bracket, err := uc.bracketRepo.GetByID(ctx, id)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "BracketUseCase - Update - GetByID")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("BracketUseCase - Update: name is required")
	}
	bracket.Name = name
	bracket.Description = description
	bracket.IsDefault = isDefault
	if err := uc.bracketRepo.Update(ctx, bracket); err != nil {
		if errors.Is(err, entityError.ErrBracketNameConflict) {
			return nil, err
		}
		return nil, usecaseutil.Wrap(err, "BracketUseCase - Update")
	}
	return bracket, nil
}

func (uc *BracketUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := uc.bracketRepo.Delete(ctx, id); err != nil {
		return usecaseutil.Wrap(err, "BracketUseCase - Delete")
	}
	return nil
}
