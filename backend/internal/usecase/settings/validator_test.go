package settings

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
)

func TestFieldValidator_ValidateValues_Success(t *testing.T) {
	h := NewFieldTestHelper(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f1 := h.NewField("name", entity.FieldTypeText, entityType, true, nil, 0)
	f2 := h.NewField("age", entity.FieldTypeNumber, entityType, false, nil, 1)
	fields := []*entity.Field{f1, f2}
	values := map[uuid.UUID]string{
		f1.ID: "short",
		f2.ID: "42",
	}

	h.SetupGetByEntityType(entityType, fields)
	v := h.CreateFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.NoError(t, err)
}

func TestFieldValidator_ValidateValues_RepoError(t *testing.T) {
	h := NewFieldTestHelper(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser

	h.SetupGetByEntityTypeError(entityType, assert.AnError)
	v := h.CreateFieldValidator()
	err := v.ValidateValues(ctx, entityType, map[uuid.UUID]string{})

	assert.Error(t, err)
}

func TestFieldValidator_ValidateValues_UnknownField(t *testing.T) {
	h := NewFieldTestHelper(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := h.NewField("name", entity.FieldTypeText, entityType, false, nil, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{
		uuid.New(): "value",
	}

	h.SetupGetByEntityType(entityType, fields)
	v := h.CreateFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestFieldValidator_ValidateValues_RequiredMissing(t *testing.T) {
	h := NewFieldTestHelper(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := h.NewField("required", entity.FieldTypeText, entityType, true, nil, 0)
	fields := []*entity.Field{f}

	h.SetupGetByEntityType(entityType, fields)
	v := h.CreateFieldValidator()
	err := v.ValidateValues(ctx, entityType, map[uuid.UUID]string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestFieldValidator_ValidateValues_NumberInvalid(t *testing.T) {
	h := NewFieldTestHelper(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := h.NewField("age", entity.FieldTypeNumber, entityType, false, nil, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{f.ID: "not-a-number"}

	h.SetupGetByEntityType(entityType, fields)
	v := h.CreateFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number")
}

func TestFieldValidator_ValidateValues_BooleanInvalid(t *testing.T) {
	h := NewFieldTestHelper(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := h.NewField("flag", entity.FieldTypeBoolean, entityType, false, nil, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{f.ID: "yes"}

	h.SetupGetByEntityType(entityType, fields)
	v := h.CreateFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "true or false")
}

func TestFieldValidator_ValidateValues_SelectInvalidOption(t *testing.T) {
	h := NewFieldTestHelper(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	opts := []string{"a", "b"}
	f := h.NewField("choice", entity.FieldTypeSelect, entityType, false, opts, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{f.ID: "c"}

	h.SetupGetByEntityType(entityType, fields)
	v := h.CreateFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid option")
}

func TestFieldValidator_ValidateValues_TextTooLong(t *testing.T) {
	h := NewFieldTestHelper(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := h.NewField("desc", entity.FieldTypeText, entityType, false, nil, 0)
	fields := []*entity.Field{f}
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}
	values := map[uuid.UUID]string{f.ID: string(long)}

	h.SetupGetByEntityType(entityType, fields)
	v := h.CreateFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max 500")
}

func TestFieldValidator_ValidateValues_SelectSuccess(t *testing.T) {
	h := NewFieldTestHelper(t)
	ctx := context.Background()
	entityType := entity.EntityTypeTeam
	opts := []string{"a", "b"}
	f := h.NewField("choice", entity.FieldTypeSelect, entityType, true, opts, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{f.ID: "a"}

	h.SetupGetByEntityType(entityType, fields)
	v := h.CreateFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.NoError(t, err)
}

func TestFieldValidator_ValidateValues_BooleanSuccess(t *testing.T) {
	h := NewFieldTestHelper(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := h.NewField("flag", entity.FieldTypeBoolean, entityType, false, nil, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{f.ID: "true"}

	h.SetupGetByEntityType(entityType, fields)
	v := h.CreateFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.NoError(t, err)
}
