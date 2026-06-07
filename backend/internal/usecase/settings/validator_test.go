package settings

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func setupValidatorGetByEntityType(d *fieldTestDeps, entityType domain.EntityType, fields []*domain.Field) {
	d.fieldRepo.EXPECT().GetByEntityType(mock.Anything, entityType).Return(fields, nil)
}

func setupValidatorGetByEntityTypeError(d *fieldTestDeps, entityType domain.EntityType, err error) {
	d.fieldRepo.EXPECT().GetByEntityType(mock.Anything, entityType).Return(nil, err)
}

func TestFieldValidator_ValidateValues_Success(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	f1 := newTestField("name", domain.FieldTypeText, entityType, true, nil, 0)
	f2 := newTestField("age", domain.FieldTypeNumber, entityType, false, nil, 1)
	fields := []*domain.Field{f1, f2}
	values := map[uuid.UUID]any{
		f1.ID: "short",
		f2.ID: float64(42),
	}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	normalized, err := v.ValidateValues(ctx, entityType, values)

	assert.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]string{f1.ID: "short", f2.ID: "42"}, normalized)
}

func TestFieldValidator_ValidateValues_RepoError(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser

	setupValidatorGetByEntityTypeError(d, entityType, assert.AnError)
	v := d.createFieldValidator()
	_, err := v.ValidateValues(ctx, entityType, map[uuid.UUID]any{})

	assert.Error(t, err)
}

func TestFieldValidator_ValidateValues_UnknownField(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	f := newTestField("name", domain.FieldTypeText, entityType, false, nil, 0)
	fields := []*domain.Field{f}
	values := map[uuid.UUID]any{
		uuid.New(): "value",
	}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	_, err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestFieldValidator_ValidateValues_RequiredMissing(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	f := newTestField("required", domain.FieldTypeText, entityType, true, nil, 0)
	fields := []*domain.Field{f}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	_, err := v.ValidateValues(ctx, entityType, map[uuid.UUID]any{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestFieldValidator_ValidateValues_NumberInvalid(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	f := newTestField("age", domain.FieldTypeNumber, entityType, false, nil, 0)
	fields := []*domain.Field{f}
	values := map[uuid.UUID]any{f.ID: 3.14}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	_, err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number")
}

func TestFieldValidator_ValidateValues_BooleanInvalid(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	f := newTestField("flag", domain.FieldTypeBoolean, entityType, false, nil, 0)
	fields := []*domain.Field{f}
	values := map[uuid.UUID]any{f.ID: "yes"}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	_, err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "true or false")
}

func TestFieldValidator_ValidateValues_SelectInvalidOption(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	opts := []string{"a", "b"}
	f := newTestField("choice", domain.FieldTypeSelect, entityType, false, opts, 0)
	fields := []*domain.Field{f}
	values := map[uuid.UUID]any{f.ID: "c"}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	_, err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid option")
}

func TestFieldValidator_ValidateValues_TextTooLong(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	f := newTestField("desc", domain.FieldTypeText, entityType, false, nil, 0)
	fields := []*domain.Field{f}

	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}

	values := map[uuid.UUID]any{f.ID: string(long)}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	_, err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max 500")
}

func TestFieldValidator_ValidateValues_SelectSuccess(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeTeam
	opts := []string{"a", "b"}
	f := newTestField("choice", domain.FieldTypeSelect, entityType, true, opts, 0)
	fields := []*domain.Field{f}
	values := map[uuid.UUID]any{f.ID: "a"}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	normalized, err := v.ValidateValues(ctx, entityType, values)

	assert.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]string{f.ID: "a"}, normalized)
}

func TestFieldValidator_ValidateValues_BooleanSuccess(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	f := newTestField("flag", domain.FieldTypeBoolean, entityType, false, nil, 0)
	fields := []*domain.Field{f}
	values := map[uuid.UUID]any{f.ID: true}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	normalized, err := v.ValidateValues(ctx, entityType, values)

	assert.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]string{f.ID: "true"}, normalized)
}

func TestFieldValidator_ValidateValues_JSONSuccess(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	f := newTestField("metadata", domain.FieldTypeJSON, entityType, false, nil, 0)
	fields := []*domain.Field{f}
	values := map[uuid.UUID]any{
		f.ID: map[string]any{
			"rank":  float64(3),
			"roles": []any{"captain", "pwner"},
		},
	}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	normalized, err := v.ValidateValues(ctx, entityType, values)

	assert.NoError(t, err)
	assert.JSONEq(t, `{"rank":3,"roles":["captain","pwner"]}`, normalized[f.ID])
}

func TestFieldValidator_ValidateValues_RequiredJSONNullRejected(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	f := newTestField("metadata", domain.FieldTypeJSON, entityType, true, nil, 0)

	setupValidatorGetByEntityType(d, entityType, []*domain.Field{f})
	v := d.createFieldValidator()
	_, err := v.ValidateValues(ctx, entityType, map[uuid.UUID]any{f.ID: nil})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestFieldValidator_ValidateEditableValues_SuccessDoesNotRequireMissingRequiredFields(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	required := newTestField("required", domain.FieldTypeText, entityType, true, nil, 0)
	editable := newTestField("editable", domain.FieldTypeNumber, entityType, false, nil, 1)
	required.Editable = true
	editable.Editable = true
	fields := []*domain.Field{required, editable}
	values := map[uuid.UUID]any{editable.ID: float64(7)}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	normalized, err := v.ValidateEditableValues(ctx, entityType, values)

	assert.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]string{editable.ID: "7"}, normalized)
}

func TestFieldValidator_ValidateEditableValues_NonEditable(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	field := newTestField("locked", domain.FieldTypeText, entityType, false, nil, 0)
	field.Editable = false

	setupValidatorGetByEntityType(d, entityType, []*domain.Field{field})
	v := d.createFieldValidator()
	_, err := v.ValidateEditableValues(ctx, entityType, map[uuid.UUID]any{field.ID: "value"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not editable")
}

func TestFieldValidator_ValidateEditableValues_RequiredTextCannotBeEmptyWhenProvided(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser
	field := newTestField("required", domain.FieldTypeText, entityType, true, nil, 0)
	field.Editable = true

	setupValidatorGetByEntityType(d, entityType, []*domain.Field{field})
	v := d.createFieldValidator()
	_, err := v.ValidateEditableValues(ctx, entityType, map[uuid.UUID]any{field.ID: ""})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestFieldValidator_ValidateEditableValues_UnknownField(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := domain.EntityTypeUser

	setupValidatorGetByEntityType(d, entityType, []*domain.Field{})
	v := d.createFieldValidator()
	_, err := v.ValidateEditableValues(ctx, entityType, map[uuid.UUID]any{uuid.New(): "value"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}
