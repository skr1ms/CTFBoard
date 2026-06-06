package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type FieldUseCase struct {
	deps FieldDeps
}

type FieldDeps struct {
	FieldRepo FieldRepository
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

func (uc *FieldUseCase) Create(ctx context.Context, params usecase.FieldCreateParams) (*domain.Field, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, apperr.NewValidationErrorf("name is required")
	}

	field := &domain.Field{
		ID:         uuid.New(),
		Name:       name,
		FieldType:  params.FieldType,
		EntityType: params.EntityType,
		Required:   params.Required,
		Options:    params.Options,
		OrderIndex: params.OrderIndex,
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

func (uc *FieldUseCase) Update(ctx context.Context, ID uuid.UUID, params usecase.FieldUpdateParams) (*domain.Field, error) {
	field, err := uc.deps.FieldRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("FieldUseCase - Update - FieldRepo.GetByID: %w", err)
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, apperr.NewValidationErrorf("name is required")
	}

	field.Name = name
	field.FieldType = params.FieldType
	field.Required = params.Required
	field.Options = params.Options

	field.OrderIndex = params.OrderIndex
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
