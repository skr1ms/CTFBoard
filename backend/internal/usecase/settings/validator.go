package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	validation "github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

const maxFieldTextLen = 500
const maxSafeJSONInteger = float64(1<<53 - 1)

type FieldValidator struct {
	fieldRepo FieldRepository
}

func NewFieldValidator(
	fieldRepo FieldRepository,
) *FieldValidator {
	return &FieldValidator{fieldRepo: fieldRepo}
}

func (v *FieldValidator) ValidateValues(ctx context.Context, entityType domain.EntityType, values map[uuid.UUID]any) (map[uuid.UUID]string, error) {
	fields, err := v.fieldRepo.GetByEntityType(ctx, entityType)
	if err != nil {
		return nil, fmt.Errorf("FieldValidator - ValidateValues - FieldRepo.GetByEntityType: %w", err)
	}

	fieldMap := make(map[uuid.UUID]*domain.Field)
	normalized := make(map[uuid.UUID]string, len(values))

	for _, f := range fields {
		fieldMap[f.ID] = f
	}

	for fieldID, value := range values {
		field, ok := fieldMap[fieldID]
		if !ok {
			return nil, apperr.NewValidationErrorf("unknown field")
		}

		normalizedValue, err := v.validateValue(field, value)
		if err != nil {
			return nil, fmt.Errorf("FieldValidator - ValidateValues - validateValue: %w", err)
		}

		normalized[fieldID] = normalizedValue
	}

	for _, field := range fields {
		if field.Required {
			val, ok := normalized[field.ID]
			if !ok {
				return nil, apperr.NewValidationErrorf("required field missing")
			}

			if requiredFieldValueEmpty(field, val) {
				return nil, apperr.NewValidationErrorf("required field cannot be empty")
			}
		}
	}

	return normalized, nil
}

func (v *FieldValidator) ValidateEditableValues(ctx context.Context, entityType domain.EntityType, values map[uuid.UUID]any) (map[uuid.UUID]string, error) {
	fields, err := v.fieldRepo.GetByEntityType(ctx, entityType)
	if err != nil {
		return nil, fmt.Errorf("FieldValidator - ValidateEditableValues - FieldRepo.GetByEntityType: %w", err)
	}

	fieldMap := make(map[uuid.UUID]*domain.Field)
	normalized := make(map[uuid.UUID]string, len(values))

	for _, f := range fields {
		fieldMap[f.ID] = f
	}

	for fieldID, value := range values {
		field, ok := fieldMap[fieldID]
		if !ok {
			return nil, apperr.NewValidationErrorf("unknown field")
		}

		if !field.Editable {
			return nil, apperr.NewValidationErrorf("field is not editable")
		}

		normalizedValue, err := v.validateValue(field, value)
		if err != nil {
			return nil, fmt.Errorf("FieldValidator - ValidateEditableValues - validateValue: %w", err)
		}

		if field.Required && requiredFieldValueEmpty(field, normalizedValue) {
			return nil, apperr.NewValidationErrorf("required field cannot be empty")
		}

		normalized[fieldID] = normalizedValue
	}

	return normalized, nil
}

func (v *FieldValidator) validateValue(field *domain.Field, value any) (string, error) {
	switch field.FieldType {
	case domain.FieldTypeNumber:
		return v.validateNumber(value)
	case domain.FieldTypeBoolean:
		return v.validateBoolean(value)
	case domain.FieldTypeSelect:
		return v.validateSelect(value, field.Options)
	case domain.FieldTypeText:
		return v.validateText(value)
	case domain.FieldTypeJSON:
		return v.validateJSON(value)
	default:
		return "", apperr.NewValidationErrorf("unsupported field type")
	}
}

func (v *FieldValidator) validateNumber(value any) (string, error) {
	switch typed := value.(type) {
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || typed != math.Trunc(typed) {
			return "", apperr.ErrFieldInvalidNumber
		}

		if typed < -maxSafeJSONInteger || typed > maxSafeJSONInteger {
			return "", apperr.ErrFieldInvalidNumber
		}

		return strconv.FormatInt(int64(typed), 10), nil
	case int:
		return strconv.Itoa(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case json.Number:
		i, err := typed.Int64()
		if err != nil {
			return "", apperr.ErrFieldInvalidNumber
		}

		return strconv.FormatInt(i, 10), nil
	default:
		return "", apperr.ErrFieldInvalidNumber
	}
}

func (v *FieldValidator) validateBoolean(value any) (string, error) {
	typed, ok := value.(bool)
	if !ok {
		return "", apperr.ErrFieldInvalidBoolean
	}

	return strconv.FormatBool(typed), nil
}

func (v *FieldValidator) validateSelect(value any, options []string) (string, error) {
	if len(options) == 0 {
		return "", apperr.NewValidationErrorf("select field has no options configured")
	}

	typed, ok := value.(string)
	if !ok {
		return "", apperr.NewValidationErrorf("select field value must be a string")
	}

	typed = validation.SanitizeCustomFieldValue(typed)

	if slices.Contains(options, typed) {
		return typed, nil
	}

	return "", apperr.NewValidationErrorf("invalid option")
}

func (v *FieldValidator) validateText(value any) (string, error) {
	typed, ok := value.(string)
	if !ok {
		return "", apperr.NewValidationErrorf("text field value must be a string")
	}

	typed = validation.SanitizeCustomFieldValue(typed)

	if len(typed) > maxFieldTextLen {
		return "", apperr.ErrFieldTextTooLong
	}

	return typed, nil
}

func (v *FieldValidator) validateJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", apperr.NewValidationErrorf("json field value must be JSON-serializable")
	}

	if len(raw) > validation.MaxCustomFieldEncodedValueLen {
		return "", apperr.NewValidationErrorf("json field value exceeds maximum length (%d)", validation.MaxCustomFieldEncodedValueLen)
	}

	return string(raw), nil
}

func requiredFieldValueEmpty(field *domain.Field, value string) bool {
	switch field.FieldType {
	case domain.FieldTypeText, domain.FieldTypeSelect:
		return strings.TrimSpace(value) == ""
	case domain.FieldTypeJSON:
		return value == "null"
	case domain.FieldTypeNumber, domain.FieldTypeBoolean:
		return false
	default:
		return false
	}
}
