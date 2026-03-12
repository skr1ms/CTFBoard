package settings

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type fieldTestDeps struct {
	fieldRepo *mocks.MockFieldRepository
}

func newFieldTestDeps(t *testing.T) *fieldTestDeps {
	t.Helper()
	return &fieldTestDeps{fieldRepo: mocks.NewMockFieldRepository(t)}
}

func (d *fieldTestDeps) createUseCase() *FieldUseCase {
	return NewFieldUseCase(FieldDeps{FieldRepo: d.fieldRepo})
}

func (d *fieldTestDeps) createFieldValidator() *FieldValidator {
	return NewFieldValidator(d.fieldRepo)
}

func newTestField(name string, fieldType entity.FieldType, entityType entity.EntityType, required bool, options []string, orderIndex int) *entity.Field {
	return &entity.Field{
		ID:         uuid.New(),
		Name:       name,
		FieldType:  fieldType,
		EntityType: entityType,
		Required:   required,
		Options:    options,
		OrderIndex: orderIndex,
	}
}

func TestFieldUseCase_GetByEntityType_Success(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	list := []*entity.Field{newTestField("name", entity.FieldTypeText, entityType, true, nil, 0)}

	d.fieldRepo.EXPECT().GetByEntityType(mock.Anything, entityType).Return(list, nil)

	uc := d.createUseCase()
	got, err := uc.GetByEntityType(ctx, entityType)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, entityType, got[0].EntityType)
}

func TestFieldUseCase_GetByEntityType_Error(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	entityType := entity.EntityTypeUser

	d.fieldRepo.EXPECT().GetByEntityType(mock.Anything, entityType).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.GetByEntityType(ctx, entityType)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestFieldUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	name := "field1"
	fieldType := entity.FieldTypeText
	entityType := entity.EntityTypeTeam
	required := true
	options := []string{"a", "b"}
	orderIndex := 1

	d.fieldRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, f *entity.Field) {
		assert.Equal(t, name, f.Name)
		assert.Equal(t, fieldType, f.FieldType)
		assert.Equal(t, entityType, f.EntityType)
		assert.Equal(t, required, f.Required)
		assert.Equal(t, options, f.Options)
		assert.Equal(t, orderIndex, f.OrderIndex)
	})

	uc := d.createUseCase()
	got, err := uc.Create(ctx, name, fieldType, entityType, true, options, orderIndex)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, name, got.Name)
}

func TestFieldUseCase_Create_Error(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()

	d.fieldRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createUseCase()
	got, err := uc.Create(ctx, "name", entity.FieldTypeText, entity.EntityTypeUser, false, nil, 0)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestFieldUseCase_GetByID_Success(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	id := uuid.New()
	field := newTestField("f", entity.FieldTypeText, entity.EntityTypeUser, false, nil, 0)
	field.ID = id

	d.fieldRepo.EXPECT().GetByID(mock.Anything, id).Return(field, nil)

	uc := d.createUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestFieldUseCase_GetByID_Error(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.fieldRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestFieldUseCase_GetAll_Success(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	list := []*entity.Field{newTestField("f", entity.FieldTypeText, entity.EntityTypeUser, false, nil, 0)}

	d.fieldRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := d.createUseCase()
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestFieldUseCase_GetAll_Error(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()

	d.fieldRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.GetAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestFieldUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	id := uuid.New()
	field := newTestField("old", entity.FieldTypeText, entity.EntityTypeUser, false, nil, 0)
	field.ID = id
	name := "new"
	fieldType := entity.FieldTypeSelect
	required := true
	options := []string{"x"}
	orderIndex := 2

	d.fieldRepo.EXPECT().GetByID(mock.Anything, id).Return(field, nil)
	d.fieldRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, f *entity.Field) {
		assert.Equal(t, name, f.Name)
		assert.Equal(t, fieldType, f.FieldType)
		assert.Equal(t, required, f.Required)
		assert.Equal(t, options, f.Options)
		assert.Equal(t, orderIndex, f.OrderIndex)
	})

	uc := d.createUseCase()
	got, err := uc.Update(ctx, id, name, fieldType, true, options, orderIndex)

	assert.NoError(t, err)
	assert.Equal(t, name, got.Name)
}

func TestFieldUseCase_Update_Error(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.fieldRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, httperr.ErrFieldNotFound)

	uc := d.createUseCase()
	got, err := uc.Update(ctx, id, "name", entity.FieldTypeText, false, nil, 0)

	assert.ErrorIs(t, err, httperr.ErrFieldNotFound)
	assert.Nil(t, got)
}

func TestFieldUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.fieldRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := d.createUseCase()
	err := uc.Delete(ctx, id)

	assert.NoError(t, err)
}

func TestFieldUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newFieldTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.fieldRepo.EXPECT().Delete(mock.Anything, id).Return(assert.AnError)

	uc := d.createUseCase()
	err := uc.Delete(ctx, id)

	assert.Error(t, err)
}
