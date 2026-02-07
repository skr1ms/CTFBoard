package settings

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
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
			return usecaseutil.Wrap(err, "field "+field.Name)
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
	}
	return nil
}

func (v *FieldValidator) validateNumber(value string) error {
	if _, err := strconv.Atoi(value); err != nil {
		return fmt.Errorf("must be a number")
	}
	return nil
}

func (v *FieldValidator) validateBoolean(value string) error {
	if value != "true" && value != "false" {
		return fmt.Errorf("must be true or false")
	}
	return nil
}

func (v *FieldValidator) validateSelect(value string, options []string) error {
	if len(options) == 0 {
		return nil
	}
	for _, opt := range options {
		if opt == value {
			return nil
		}
	}
	return fmt.Errorf("invalid option: %s", value)
}

func (v *FieldValidator) validateText(value string) error {
	if len(value) > 500 {
		return fmt.Errorf("text too long (max 500)")
	}
	return nil
}
