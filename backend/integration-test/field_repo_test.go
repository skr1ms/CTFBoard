package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestFieldRepo_Create_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := &domain.Field{
		Name:        "bio",
		Description: "Biography",
		FieldType:   domain.FieldTypeText,
		EntityType:  domain.EntityTypeUser,
		Required:    false,
		Public:      true,
		Editable:    true,
		OrderIndex:  0,
	}
	err := f.FieldRepo.Create(ctx, field)
	require.NoError(t, err)
	assert.NotEmpty(t, field.ID)

	got, err := f.FieldRepo.GetByID(ctx, field.ID)
	require.NoError(t, err)
	assert.Equal(t, field.Description, got.Description)
	assert.True(t, got.Public)
	assert.True(t, got.Editable)
}

func TestFieldRepo_Create_JSONField(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := &domain.Field{
		Name:       "metadata",
		FieldType:  domain.FieldTypeJSON,
		EntityType: domain.EntityTypeTeam,
		Public:     true,
		Editable:   true,
	}

	err := f.FieldRepo.Create(ctx, field)
	require.NoError(t, err)

	got, err := f.FieldRepo.GetByID(ctx, field.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.FieldTypeJSON, got.FieldType)
	assert.Equal(t, domain.EntityTypeTeam, got.EntityType)
}

func TestFieldRepo_Create_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	field := &domain.Field{Name: "x", FieldType: domain.FieldTypeText, EntityType: domain.EntityTypeUser}
	err := f.FieldRepo.Create(ctx, field)
	assert.Error(t, err)
}

func TestFieldRepo_GetByID_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := f.CreateField(t, "gbi", domain.EntityTypeUser)
	got, err := f.FieldRepo.GetByID(ctx, field.ID)
	require.NoError(t, err)
	assert.Equal(t, field.ID, got.ID)
	assert.Equal(t, field.Name, got.Name)
	assert.Equal(t, field.Description, got.Description)
	assert.Equal(t, field.Public, got.Public)
	assert.Equal(t, field.Editable, got.Editable)
}

func TestFieldRepo_GetByID_Error_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.FieldRepo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrFieldNotFound)
}

func TestFieldRepo_GetByEntityType_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f1 := f.CreateField(t, "et1", domain.EntityTypeUser)
	f2 := f.CreateField(t, "et2", domain.EntityTypeUser)
	list, err := f.FieldRepo.GetByEntityType(ctx, domain.EntityTypeUser)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, fl := range list {
		ids[fl.ID] = true
	}

	assert.True(t, ids[f1.ID], "field 1 should be in result")
	assert.True(t, ids[f2.ID], "field 2 should be in result")
}

func TestFieldRepo_GetByEntityType_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.FieldRepo.GetByEntityType(ctx, domain.EntityTypeUser)
	assert.Error(t, err)
}

func TestFieldRepo_GetAll_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := f.CreateField(t, "ga1", domain.EntityTypeTeam)
	list, err := f.FieldRepo.GetAll(ctx)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, fl := range list {
		ids[fl.ID] = true
	}

	assert.True(t, ids[field.ID], "field should be in GetAll result")
}

func TestFieldRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.FieldRepo.GetAll(ctx)
	assert.Error(t, err)
}

func TestFieldRepo_Update_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := f.CreateField(t, "upd", domain.EntityTypeUser)
	field.Name = "updated_name"
	field.Description = "updated description"
	field.Public = true
	field.Editable = true
	err := f.FieldRepo.Update(ctx, field)
	require.NoError(t, err)
	got, err := f.FieldRepo.GetByID(ctx, field.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated_name", got.Name)
	assert.Equal(t, "updated description", got.Description)
	assert.True(t, got.Public)
	assert.True(t, got.Editable)
}

func TestFieldRepo_Update_Error_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := &domain.Field{ID: uuid.New(), Name: "x", FieldType: domain.FieldTypeText, EntityType: domain.EntityTypeUser}
	err := f.FieldRepo.Update(ctx, field)
	assert.NoError(t, err)
}

func TestFieldRepo_Delete_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	field := f.CreateField(t, "del", domain.EntityTypeUser)
	err := f.FieldRepo.Delete(ctx, field.ID)
	require.NoError(t, err)
	_, err = f.FieldRepo.GetByID(ctx, field.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrFieldNotFound)
}

func TestFieldRepo_Delete_Error_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.FieldRepo.Delete(ctx, uuid.New())
	assert.NoError(t, err)
}

func TestFieldValueRepo_GetByEntityID_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "fv")
	field := f.CreateField(t, "fv", domain.EntityTypeUser)
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.FieldValueRepo.SetValues(txCtx, user.ID, map[string]string{field.ID.String(): "hello"})
	})
	require.NoError(t, err)
	vals, err := f.FieldValueRepo.GetByEntityID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, vals, 1)
	assert.Equal(t, "hello", vals[0].Value)
}

func TestFieldValueRepo_GetByEntityID_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.FieldValueRepo.GetByEntityID(ctx, uuid.New())
	assert.Error(t, err)
}

func TestFieldValueRepo_SetValues_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "setv")
	field := f.CreateField(t, "setv", domain.EntityTypeUser)
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.FieldValueRepo.SetValues(txCtx, user.ID, map[string]string{field.ID.String(): "value1"})
	})
	require.NoError(t, err)
	vals, err := f.FieldValueRepo.GetByEntityID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, vals, 1)

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.FieldValueRepo.SetValues(txCtx, user.ID, map[string]string{field.ID.String(): "value2"})
	})
	require.NoError(t, err)
	vals, err = f.FieldValueRepo.GetByEntityID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, vals, 1)
	assert.Equal(t, "value2", vals[0].Value)
}

func TestFieldValueRepo_SetValues_Error_InvalidFieldID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "invf")
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.FieldValueRepo.SetValues(txCtx, user.ID, map[string]string{"not-a-uuid": "x"})
	})
	assert.Error(t, err)
}

func TestFieldValueRepo_UpsertValues_PartialDoesNotDeleteOmittedValues(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "upsertv")
	keptField := f.CreateField(t, "upsert_keep", domain.EntityTypeUser)
	updatedField := f.CreateField(t, "upsert_update", domain.EntityTypeUser)
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.FieldValueRepo.SetValues(txCtx, user.ID, map[string]string{
			keptField.ID.String():    "kept",
			updatedField.ID.String(): "old",
		})
	})
	require.NoError(t, err)

	err = f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.FieldValueRepo.UpsertValues(txCtx, user.ID, map[string]string{updatedField.ID.String(): "new"})
	})
	require.NoError(t, err)

	vals, err := f.FieldValueRepo.GetByEntityID(ctx, user.ID)
	require.NoError(t, err)

	got := make(map[uuid.UUID]string, len(vals))
	for _, val := range vals {
		got[val.FieldID] = val.Value
	}

	assert.Equal(t, "kept", got[keptField.ID])
	assert.Equal(t, "new", got[updatedField.ID])
}
