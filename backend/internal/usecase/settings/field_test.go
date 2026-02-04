package settings

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFieldUseCase_GetByEntityType_Success(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	entityType := entity.EntityTypeUser
	list := []*entity.Field{h.NewField("name", entity.FieldTypeText, entityType, true, nil, 0)}

	deps.fieldRepo.EXPECT().GetByEntityType(mock.Anything, entityType).Return(list, nil)

	uc := h.CreateUseCase()
	got, err := uc.GetByEntityType(ctx, entityType)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, entityType, got[0].EntityType)
}

func TestFieldUseCase_GetByEntityType_Error(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	entityType := entity.EntityTypeUser

	deps.fieldRepo.EXPECT().GetByEntityType(mock.Anything, entityType).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.GetByEntityType(ctx, entityType)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestFieldUseCase_Create_Success(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	name := "field1"
	fieldType := entity.FieldTypeText
	entityType := entity.EntityTypeTeam
	required := true
	options := []string{"a", "b"}
	orderIndex := 1

	deps.fieldRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, f *entity.Field) {
		assert.Equal(t, name, f.Name)
		assert.Equal(t, fieldType, f.FieldType)
		assert.Equal(t, entityType, f.EntityType)
		assert.Equal(t, required, f.Required)
		assert.Equal(t, options, f.Options)
		assert.Equal(t, orderIndex, f.OrderIndex)
	})

	uc := h.CreateUseCase()
	got, err := uc.Create(ctx, name, fieldType, entityType, required, options, orderIndex)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, name, got.Name)
}

func TestFieldUseCase_Create_Error(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.fieldRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.Create(ctx, "name", entity.FieldTypeText, entity.EntityTypeUser, false, nil, 0)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestFieldUseCase_GetByID_Success(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()
	field := h.NewField("f", entity.FieldTypeText, entity.EntityTypeUser, false, nil, 0)
	field.ID = id

	deps.fieldRepo.EXPECT().GetByID(mock.Anything, id).Return(field, nil)

	uc := h.CreateUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestFieldUseCase_GetByID_Error(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.fieldRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestFieldUseCase_GetAll_Success(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	list := []*entity.Field{h.NewField("f", entity.FieldTypeText, entity.EntityTypeUser, false, nil, 0)}

	deps.fieldRepo.EXPECT().GetAll(mock.Anything).Return(list, nil)

	uc := h.CreateUseCase()
	got, err := uc.GetAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestFieldUseCase_GetAll_Error(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.fieldRepo.EXPECT().GetAll(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.GetAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestFieldUseCase_Update_Success(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()
	field := h.NewField("old", entity.FieldTypeText, entity.EntityTypeUser, false, nil, 0)
	field.ID = id
	name := "new"
	fieldType := entity.FieldTypeSelect
	required := true
	options := []string{"x"}
	orderIndex := 2

	deps.fieldRepo.EXPECT().GetByID(mock.Anything, id).Return(field, nil)
	deps.fieldRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, f *entity.Field) {
		assert.Equal(t, name, f.Name)
		assert.Equal(t, fieldType, f.FieldType)
		assert.Equal(t, required, f.Required)
		assert.Equal(t, options, f.Options)
		assert.Equal(t, orderIndex, f.OrderIndex)
	})

	uc := h.CreateUseCase()
	got, err := uc.Update(ctx, id, name, fieldType, required, options, orderIndex)

	assert.NoError(t, err)
	assert.Equal(t, name, got.Name)
}

func TestFieldUseCase_Update_Error(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.fieldRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, entityError.ErrFieldNotFound)

	uc := h.CreateUseCase()
	got, err := uc.Update(ctx, id, "name", entity.FieldTypeText, false, nil, 0)

	assert.ErrorIs(t, err, entityError.ErrFieldNotFound)
	assert.Nil(t, got)
}

func TestFieldUseCase_Delete_Success(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.fieldRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := h.CreateUseCase()
	err := uc.Delete(ctx, id)

	assert.NoError(t, err)
}

func TestFieldUseCase_Delete_Error(t *testing.T) {
	h := NewFieldTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.fieldRepo.EXPECT().Delete(mock.Anything, id).Return(assert.AnError)

	uc := h.CreateUseCase()
	err := uc.Delete(ctx, id)

	assert.Error(t, err)
}
