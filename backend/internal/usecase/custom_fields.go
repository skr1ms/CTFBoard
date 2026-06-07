package usecase

import (
	"encoding/json"
	"strconv"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type CustomFieldValues = map[string]any

func CustomFieldStorageValuesToMap(
	fields []*domain.Field,
	values []*domain.FieldValue,
	include func(*domain.Field) bool,
) CustomFieldValues {
	if len(fields) == 0 || len(values) == 0 {
		return nil
	}

	fieldByID := make(map[uuid.UUID]*domain.Field, len(fields))
	for _, field := range fields {
		if field != nil && (include == nil || include(field)) {
			fieldByID[field.ID] = field
		}
	}

	if len(fieldByID) == 0 {
		return nil
	}

	out := make(CustomFieldValues)

	for _, value := range values {
		if value == nil {
			continue
		}

		field, ok := fieldByID[value.FieldID]
		if !ok {
			continue
		}

		out[value.FieldID.String()] = DecodeCustomFieldStorageValue(field.FieldType, value.Value)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func DecodeCustomFieldStorageValue(fieldType domain.FieldType, value string) any {
	switch fieldType {
	case domain.FieldTypeNumber:
		i, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return i
		}
	case domain.FieldTypeBoolean:
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	case domain.FieldTypeJSON:
		var decoded any
		if err := json.Unmarshal([]byte(value), &decoded); err == nil {
			return decoded
		}
	case domain.FieldTypeText, domain.FieldTypeSelect:
		return value
	}

	return value
}

func CustomFieldStorageValuesToStringKeyMap(values map[uuid.UUID]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	out := make(map[string]string, len(values))
	for fieldID, value := range values {
		out[fieldID.String()] = value
	}

	return out
}
