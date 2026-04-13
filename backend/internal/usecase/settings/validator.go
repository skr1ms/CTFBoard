package settings

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

const maxFieldTextLen = 500

type FieldValidator struct {
	fieldRepo repo.FieldRepository
}

func NewFieldValidator(
	fieldRepo repo.FieldRepository,
) *FieldValidator {
	return &FieldValidator{fieldRepo: fieldRepo}
}

func (v *FieldValidator) ValidateValues(ctx context.Context, entityType domain.EntityType, values map[uuid.UUID]string) error {
	fields, err := v.fieldRepo.GetByEntityType(ctx, entityType)
	if err != nil {
		return fmt.Errorf("FieldValidator - ValidateValues - FieldRepo.GetByEntityType: %w", err)
	}

	fieldMap := make(map[uuid.UUID]*domain.Field)

	for _, f := range fields {
		fieldMap[f.ID] = f
	}

	for fieldID, value := range values {
		field, ok := fieldMap[fieldID]
		if !ok {
			return apperr.NewValidationErrorf("unknown field")
		}

		err := v.validateValue(field, value)
		if err != nil {
			return fmt.Errorf("FieldValidator - ValidateValues - validateValue: %w", err)
		}
	}

	for _, field := range fields {
		if field.Required {
			val, ok := values[field.ID]
			if !ok {
				return apperr.NewValidationErrorf("required field missing")
			}

			if field.FieldType == domain.FieldTypeText && val == "" {
				return apperr.NewValidationErrorf("required field cannot be empty")
			}
		}
	}

	return nil
}

func (v *FieldValidator) validateValue(field *domain.Field, value string) error {
	switch field.FieldType {
	case domain.FieldTypeNumber:
		return v.validateNumber(value)
	case domain.FieldTypeBoolean:
		return v.validateBoolean(value)
	case domain.FieldTypeSelect:
		return v.validateSelect(value, field.Options)
	case domain.FieldTypeText:
		return v.validateText(value)
	default:
		return apperr.NewValidationErrorf("unsupported field type")
	}
}

func (v *FieldValidator) validateNumber(value string) error {
	if _, err := strconv.Atoi(value); err != nil {
		return apperr.ErrFieldInvalidNumber
	}

	return nil
}

func (v *FieldValidator) validateBoolean(value string) error {
	if value != "true" && value != "false" {
		return apperr.ErrFieldInvalidBoolean
	}

	return nil
}

func (v *FieldValidator) validateSelect(value string, options []string) error {
	if len(options) == 0 {
		return apperr.NewValidationErrorf("select field has no options configured")
	}

	value = strings.TrimSpace(value)

	if slices.Contains(options, value) {
		return nil
	}

	return apperr.NewValidationErrorf("invalid option")
}

func (v *FieldValidator) validateText(value string) error {
	if len(value) > maxFieldTextLen {
		return apperr.ErrFieldTextTooLong
	}

	return nil
}
