package settings

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

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
		return err
	}
	fieldMap := make(map[uuid.UUID]*entity.Field)
	for _, f := range fields {
		fieldMap[f.ID] = f
	}
	for fieldID, value := range values {
		field, ok := fieldMap[fieldID]
		if !ok {
			return fmt.Errorf("unknown field: %s", fieldID)
		}
		if err := v.validateValue(field, value); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}
	for _, field := range fields {
		if field.Required {
			if _, ok := values[field.ID]; !ok {
				return fmt.Errorf("field %s is required", field.Name)
			}
		}
	}
	return nil
}

//nolint:gocognit,gocyclo // switch over field types
func (v *FieldValidator) validateValue(field *entity.Field, value string) error {
	switch field.FieldType {
	case entity.FieldTypeNumber:
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("must be a number")
		}
	case entity.FieldTypeBoolean:
		if value != "true" && value != "false" {
			return fmt.Errorf("must be true or false")
		}
	case entity.FieldTypeSelect:
		if len(field.Options) > 0 {
			valid := false
			for _, opt := range field.Options {
				if opt == value {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid option: %s", value)
			}
		}
	case entity.FieldTypeText:
		if len(value) > 500 {
			return fmt.Errorf("text too long (max 500)")
		}
	}
	return nil
}
