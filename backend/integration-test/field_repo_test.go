package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldRepo_Create_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := &entity.Field{
		Name:       "bio",
		FieldType:  entity.FieldTypeText,
		EntityType: entity.EntityTypeUser,
		Required:   false,
		OrderIndex: 0,
	}
	err := f.FieldRepo.Create(ctx, field)
	require.NoError(t, err)
	assert.NotEmpty(t, field.ID)
}

func TestFieldRepo_Create_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	field := &entity.Field{Name: "x", FieldType: entity.FieldTypeText, EntityType: entity.EntityTypeUser}
	err := f.FieldRepo.Create(ctx, field)
	assert.Error(t, err)
}

func TestFieldRepo_GetByID_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := f.CreateField(t, "gbi", entity.EntityTypeUser)
	got, err := f.FieldRepo.GetByID(ctx, field.ID)
	require.NoError(t, err)
	assert.Equal(t, field.ID, got.ID)
	assert.Equal(t, field.Name, got.Name)
}

func TestFieldRepo_GetByID_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.FieldRepo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrFieldNotFound))
}

func TestFieldRepo_GetByEntityType_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateField(t, "et1", entity.EntityTypeUser)
	f.CreateField(t, "et2", entity.EntityTypeUser)
	list, err := f.FieldRepo.GetByEntityType(ctx, entity.EntityTypeUser)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 2)
}

func TestFieldRepo_GetByEntityType_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.FieldRepo.GetByEntityType(ctx, entity.EntityTypeUser)
	assert.Error(t, err)
}

func TestFieldRepo_GetAll_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateField(t, "ga1", entity.EntityTypeTeam)
	list, err := f.FieldRepo.GetAll(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestFieldRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.FieldRepo.GetAll(ctx)
	assert.Error(t, err)
}

func TestFieldRepo_Update_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := f.CreateField(t, "upd", entity.EntityTypeUser)
	field.Name = "updated_name"
	err := f.FieldRepo.Update(ctx, field)
	require.NoError(t, err)
	got, err := f.FieldRepo.GetByID(ctx, field.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated_name", got.Name)
}

func TestFieldRepo_Update_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := &entity.Field{ID: uuid.New(), Name: "x", FieldType: entity.FieldTypeText, EntityType: entity.EntityTypeUser}
	err := f.FieldRepo.Update(ctx, field)
	assert.NoError(t, err)
}

func TestFieldRepo_Delete_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := f.CreateField(t, "del", entity.EntityTypeUser)
	err := f.FieldRepo.Delete(ctx, field.ID)
	require.NoError(t, err)
	_, err = f.FieldRepo.GetByID(ctx, field.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrFieldNotFound))
}

func TestFieldRepo_Delete_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.FieldRepo.Delete(ctx, uuid.New())
	assert.NoError(t, err)
}

func TestFieldValueRepo_GetByEntityID_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "fv")
	field := f.CreateField(t, "fv", entity.EntityTypeUser)
	err := f.FieldValueRepo.SetValues(ctx, user.ID, map[string]string{field.ID.String(): "hello"})
	require.NoError(t, err)
	vals, err := f.FieldValueRepo.GetByEntityID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, vals, 1)
	assert.Equal(t, "hello", vals[0].Value)
}

func TestFieldValueRepo_GetByEntityID_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.FieldValueRepo.GetByEntityID(ctx, uuid.New())
	assert.Error(t, err)
}

func TestFieldValueRepo_SetValues_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "setv")
	field := f.CreateField(t, "setv", entity.EntityTypeUser)
	err := f.FieldValueRepo.SetValues(ctx, user.ID, map[string]string{field.ID.String(): "value1"})
	require.NoError(t, err)
	vals, err := f.FieldValueRepo.GetByEntityID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, vals, 1)
	err = f.FieldValueRepo.SetValues(ctx, user.ID, map[string]string{field.ID.String(): "value2"})
	require.NoError(t, err)
	vals, err = f.FieldValueRepo.GetByEntityID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, vals, 1)
	assert.Equal(t, "value2", vals[0].Value)
}

func TestFieldValueRepo_SetValues_Error_InvalidFieldID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "invf")
	err := f.FieldValueRepo.SetValues(ctx, user.ID, map[string]string{"not-a-uuid": "x"})
	assert.Error(t, err)
}
