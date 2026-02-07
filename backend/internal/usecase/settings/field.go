package settings

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

type FieldUseCase struct {
	fieldRepo repo.FieldRepository
}

func NewFieldUseCase(
	fieldRepo repo.FieldRepository,
) *FieldUseCase {
	return &FieldUseCase{fieldRepo: fieldRepo}
}

func (uc *FieldUseCase) GetByEntityType(ctx context.Context, entityType entity.EntityType) ([]*entity.Field, error) {
	list, err := uc.fieldRepo.GetByEntityType(ctx, entityType)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "FieldUseCase - GetByEntityType")
	}
	return list, nil
}

func (uc *FieldUseCase) Create(ctx context.Context, name string, fieldType entity.FieldType, entityType entity.EntityType, required bool, options []string, orderIndex int) (*entity.Field, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("FieldUseCase - Create: name is required")
	}
	field := &entity.Field{
		ID:         uuid.New(),
		Name:       name,
		FieldType:  fieldType,
		EntityType: entityType,
		Required:   required,
		Options:    options,
		OrderIndex: orderIndex,
	}
	if err := uc.fieldRepo.Create(ctx, field); err != nil {
		return nil, usecaseutil.Wrap(err, "FieldUseCase - Create")
	}
	return field, nil
}

func (uc *FieldUseCase) GetByID(ctx context.Context, id uuid.UUID) (*entity.Field, error) {
	field, err := uc.fieldRepo.GetByID(ctx, id)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "FieldUseCase - GetByID")
	}
	return field, nil
}

func (uc *FieldUseCase) GetAll(ctx context.Context) ([]*entity.Field, error) {
	list, err := uc.fieldRepo.GetAll(ctx)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "FieldUseCase - GetAll")
	}
	return list, nil
}

func (uc *FieldUseCase) Update(ctx context.Context, id uuid.UUID, name string, fieldType entity.FieldType, required bool, options []string, orderIndex int) (*entity.Field, error) {
	field, err := uc.fieldRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, entityError.ErrFieldNotFound) {
			return nil, err
		}
		return nil, usecaseutil.Wrap(err, "FieldUseCase - Update - GetByID")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, usecaseutil.Wrap(err, "FieldUseCase - Update: name is required")
	}
	field.Name = name
	field.FieldType = fieldType
	field.Required = required
	field.Options = options
	field.OrderIndex = orderIndex
	if err := uc.fieldRepo.Update(ctx, field); err != nil {
		return nil, usecaseutil.Wrap(err, "FieldUseCase - Update")
	}
	return field, nil
}

func (uc *FieldUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := uc.fieldRepo.Delete(ctx, id); err != nil {
		return usecaseutil.Wrap(err, "FieldUseCase - Delete")
	}
	return nil
}
