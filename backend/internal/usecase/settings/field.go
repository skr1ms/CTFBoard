package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type FieldUseCase struct {
	deps FieldDeps
}

type FieldDeps struct {
	FieldRepo repo.FieldRepository
}

var _ usecase.FieldUseCase = (*FieldUseCase)(nil)

func NewFieldUseCase(deps FieldDeps) *FieldUseCase {
	return &FieldUseCase{deps: deps}
}

func (uc *FieldUseCase) GetByEntityType(ctx context.Context, entityType domain.EntityType) ([]*domain.Field, error) {
	list, err := uc.deps.FieldRepo.GetByEntityType(ctx, entityType)
	if err != nil {
		return nil, fmt.Errorf("FieldUseCase - GetByEntityType - FieldRepo.GetByEntityType: %w", err)
	}

	return list, nil
}

func (uc *FieldUseCase) Create(ctx context.Context, name string, fieldType domain.FieldType, entityType domain.EntityType, required bool, options []string, orderIndex int) (*domain.Field, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, httperr.NewValidationErrorf("name is required")
	}

	field := &domain.Field{
		ID:         uuid.New(),
		Name:       name,
		FieldType:  fieldType,
		EntityType: entityType,
		Required:   required,
		Options:    options,
		OrderIndex: orderIndex,
	}

	err := uc.deps.FieldRepo.Create(ctx, field)
	if err != nil {
		return nil, fmt.Errorf("FieldUseCase - Create - FieldRepo.Create: %w", err)
	}

	return field, nil
}

func (uc *FieldUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Field, error) {
	field, err := uc.deps.FieldRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("FieldUseCase - GetByID - FieldRepo.GetByID: %w", err)
	}

	return field, nil
}

func (uc *FieldUseCase) GetAll(ctx context.Context) ([]*domain.Field, error) {
	list, err := uc.deps.FieldRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("FieldUseCase - GetAll - FieldRepo.GetAll: %w", err)
	}

	return list, nil
}

func (uc *FieldUseCase) Update(ctx context.Context, ID uuid.UUID, name string, fieldType domain.FieldType, required bool, options []string, orderIndex int) (*domain.Field, error) {
	field, err := uc.deps.FieldRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("FieldUseCase - Update - FieldRepo.GetByID: %w", err)
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, httperr.NewValidationErrorf("name is required")
	}

	field.Name = name
	field.FieldType = fieldType
	field.Required = required
	field.Options = options

	field.OrderIndex = orderIndex
	if err := uc.deps.FieldRepo.Update(ctx, field); err != nil {
		return nil, fmt.Errorf("FieldUseCase - Update - FieldRepo.Update: %w", err)
	}

	return field, nil
}

func (uc *FieldUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	err := uc.deps.FieldRepo.Delete(ctx, ID)
	if err != nil {
		return fmt.Errorf("FieldUseCase - Delete - FieldRepo.Delete: %w", err)
	}

	return nil
}
