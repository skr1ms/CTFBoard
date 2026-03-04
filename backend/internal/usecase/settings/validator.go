package settings

import (
	"context"
	"fmt"
	"strconv"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
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

func (v *FieldValidator) ValidateValues(ctx context.Context, entityType entity.EntityType, values map[uuid.UUID]string) error {
	fields, err := v.fieldRepo.GetByEntityType(ctx, entityType)
	if err != nil {
		return fmt.Errorf("FieldValidator - ValidateValues - FieldRepo.GetByEntityType: %w", err)
	}
	fieldMap := make(map[uuid.UUID]*entity.Field)
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
			if _, ok := values[field.ID]; !ok {
				return httperr.NewValidationErrorf("required field missing")
			}
		}
	}
	return nil
}

func (v *FieldValidator) validateValue(field *entity.Field, value string) error {
	switch field.FieldType {
	case entity.FieldTypeNumber:
		return v.validateNumber(value)
	case entity.FieldTypeBoolean:
		return v.validateBoolean(value)
	case entity.FieldTypeSelect:
		return v.validateSelect(value, field.Options)
	case entity.FieldTypeText:
		return v.validateText(value)
	default:
		return httperr.NewValidationErrorf("unsupported field type: %s", string(field.FieldType))
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
