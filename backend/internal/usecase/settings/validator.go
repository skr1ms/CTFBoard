package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
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
			return httperr.NewValidationErrorf("unknown field")
		}
		if err := v.validateValue(field, value); err != nil {
			return fmt.Errorf("FieldValidator - ValidateValues - validateValue: %w", err)
		}
	}
	for _, field := range fields {
		if field.Required {
			val, ok := values[field.ID]
			if !ok {
				return httperr.NewValidationErrorf("required field missing")
			}
			if field.FieldType == domain.FieldTypeText && val == "" {
				return httperr.NewValidationErrorf("required field cannot be empty")
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
		return httperr.NewValidationErrorf("unsupported field type")
	}
}

func (v *FieldValidator) validateNumber(value string) error {
	if _, err := strconv.Atoi(value); err != nil {
		return httperr.ErrFieldInvalidNumber
	}
	return nil
}

func (v *FieldValidator) validateBoolean(value string) error {
	if value != "true" && value != "false" {
		return httperr.ErrFieldInvalidBoolean
	}
	return nil
}

func (v *FieldValidator) validateSelect(value string, options []string) error {
	if len(options) == 0 {
		return httperr.NewValidationErrorf("select field has no options configured")
	}
	value = strings.TrimSpace(value)
	for _, opt := range options {
		if opt == value {
			return nil
		}
	}
	return httperr.NewValidationErrorf("invalid option")
}

func (v *FieldValidator) validateText(value string) error {
	if len(value) > maxFieldTextLen {
		return httperr.ErrFieldTextTooLong
	}
	return nil
}
