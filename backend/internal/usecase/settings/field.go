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

const maxFieldDescriptionLen = 500

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
	normalized, err := normalizeFieldParams(params.Name, params.Description, params.FieldType, params.EntityType, params.Options)
	if err != nil {
		return nil, fmt.Errorf("FieldUseCase - Create - normalizeFieldParams: %w", err)
	}

	field := &domain.Field{
		ID:          uuid.New(),
		Name:        normalized.name,
		Description: normalized.description,
		FieldType:   params.FieldType,
		EntityType:  params.EntityType,
		Required:    params.Required,
		Public:      params.Public,
		Editable:    params.Editable,
		Options:     normalized.options,
		OrderIndex:  params.OrderIndex,
	}

	err = uc.deps.FieldRepo.Create(ctx, field)
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

	normalized, err := normalizeFieldParams(params.Name, params.Description, params.FieldType, field.EntityType, params.Options)
	if err != nil {
		return nil, fmt.Errorf("FieldUseCase - Update - normalizeFieldParams: %w", err)
	}

	field.Name = normalized.name
	field.Description = normalized.description
	field.FieldType = params.FieldType
	field.Required = params.Required
	field.Public = params.Public
	field.Editable = params.Editable
	field.Options = normalized.options

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

type normalizedFieldParams struct {
	name        string
	description string
	options     []string
}

func normalizeFieldParams(name, description string, fieldType domain.FieldType, entityType domain.EntityType, options []string) (normalizedFieldParams, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return normalizedFieldParams{}, apperr.NewValidationErrorf("name is required")
	}

	description = strings.TrimSpace(description)
	if len(description) > maxFieldDescriptionLen {
		return normalizedFieldParams{}, apperr.NewValidationErrorf("description exceeds maximum length (%d)", maxFieldDescriptionLen)
	}

	if !fieldType.IsValid() {
		return normalizedFieldParams{}, apperr.NewValidationErrorf("unsupported field type")
	}

	if !entityType.IsValid() {
		return normalizedFieldParams{}, apperr.NewValidationErrorf("unsupported entity type")
	}

	out := normalizedFieldParams{name: name, description: description}

	if fieldType != domain.FieldTypeSelect {
		return out, nil
	}

	seen := make(map[string]struct{}, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option == "" {
			continue
		}

		if _, ok := seen[option]; ok {
			continue
		}

		seen[option] = struct{}{}
		out.options = append(out.options, option)
	}

	if len(out.options) == 0 {
		return normalizedFieldParams{}, apperr.NewValidationErrorf("select field requires options")
	}

	return out, nil
}
