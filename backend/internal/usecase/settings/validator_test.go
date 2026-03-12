package settings

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
)

func setupValidatorGetByEntityType(d *fieldTestDeps, entityType entity.EntityType, fields []*entity.Field) {
	d.fieldRepo.EXPECT().GetByEntityType(mock.Anything, entityType).Return(fields, nil)
}

func setupValidatorGetByEntityTypeError(d *fieldTestDeps, entityType entity.EntityType, err error) {
	d.fieldRepo.EXPECT().GetByEntityType(mock.Anything, entityType).Return(nil, err)
}

func TestFieldValidator_ValidateValues_Success(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f1 := newTestField("name", entity.FieldTypeText, entityType, true, nil, 0)
	f2 := newTestField("age", entity.FieldTypeNumber, entityType, false, nil, 1)
	fields := []*entity.Field{f1, f2}
	values := map[uuid.UUID]string{
		f1.ID: "short",
		f2.ID: "42",
	}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.NoError(t, err)
}

func TestFieldValidator_ValidateValues_RepoError(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser

	setupValidatorGetByEntityTypeError(d, entityType, assert.AnError)
	v := d.createFieldValidator()
	err := v.ValidateValues(ctx, entityType, map[uuid.UUID]string{})

	assert.Error(t, err)
}

func TestFieldValidator_ValidateValues_UnknownField(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := newTestField("name", entity.FieldTypeText, entityType, false, nil, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{
		uuid.New(): "value",
	}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestFieldValidator_ValidateValues_RequiredMissing(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := newTestField("required", entity.FieldTypeText, entityType, true, nil, 0)
	fields := []*entity.Field{f}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	err := v.ValidateValues(ctx, entityType, map[uuid.UUID]string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestFieldValidator_ValidateValues_NumberInvalid(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := newTestField("age", entity.FieldTypeNumber, entityType, false, nil, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{f.ID: "not-a-number"}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number")
}

func TestFieldValidator_ValidateValues_BooleanInvalid(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := newTestField("flag", entity.FieldTypeBoolean, entityType, false, nil, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{f.ID: "yes"}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "true or false")
}

func TestFieldValidator_ValidateValues_SelectInvalidOption(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	opts := []string{"a", "b"}
	f := newTestField("choice", entity.FieldTypeSelect, entityType, false, opts, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{f.ID: "c"}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid option")
}

func TestFieldValidator_ValidateValues_TextTooLong(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := newTestField("desc", entity.FieldTypeText, entityType, false, nil, 0)
	fields := []*entity.Field{f}
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}
	values := map[uuid.UUID]string{f.ID: string(long)}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max 500")
}

func TestFieldValidator_ValidateValues_SelectSuccess(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeTeam
	opts := []string{"a", "b"}
	f := newTestField("choice", entity.FieldTypeSelect, entityType, true, opts, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{f.ID: "a"}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.NoError(t, err)
}

func TestFieldValidator_ValidateValues_BooleanSuccess(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	f := newTestField("flag", entity.FieldTypeBoolean, entityType, false, nil, 0)
	fields := []*entity.Field{f}
	values := map[uuid.UUID]string{f.ID: "true"}

	setupValidatorGetByEntityType(d, entityType, fields)
	v := d.createFieldValidator()
	err := v.ValidateValues(ctx, entityType, values)

	assert.NoError(t, err)
}
