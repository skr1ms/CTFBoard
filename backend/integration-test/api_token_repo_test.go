package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPITokenRepo_Create_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "apitok")
	token := &entity.APIToken{UserID: user.ID, TokenHash: "hash_" + uuid.New().String(), Description: "test"}
	err := f.APITokenRepo.Create(ctx, token)
	require.NoError(t, err)
	assert.NotEmpty(t, token.ID)
}

func TestAPITokenRepo_Create_Error_InvalidUserID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	token := &entity.APIToken{UserID: uuid.New(), TokenHash: "hash_xyz", Description: "x"}
	err := f.APITokenRepo.Create(ctx, token)
	assert.Error(t, err)
}

func TestAPITokenRepo_GetByUserID_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "gbu")
	token := &entity.APIToken{UserID: user.ID, TokenHash: "hash_" + uuid.New().String()}
	err := f.APITokenRepo.Create(ctx, token)
	require.NoError(t, err)
	list, err := f.APITokenRepo.GetByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, token.ID, list[0].ID)
}

func TestAPITokenRepo_GetByUserID_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	user := f.CreateUser(t, "gbuerr")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.APITokenRepo.GetByUserID(ctx, user.ID)
	assert.Error(t, err)
}

func TestAPITokenRepo_GetByTokenHash_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "gbh")
	hash := "hash_" + uuid.New().String()
	token := &entity.APIToken{UserID: user.ID, TokenHash: hash}
	err := f.APITokenRepo.Create(ctx, token)
	require.NoError(t, err)
	got, err := f.APITokenRepo.GetByTokenHash(ctx, hash)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, token.ID, got.ID)
}

func TestAPITokenRepo_GetByTokenHash_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	got, err := f.APITokenRepo.GetByTokenHash(ctx, "nonexistent_hash_xyz")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestAPITokenRepo_Delete_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "del")
	token := &entity.APIToken{UserID: user.ID, TokenHash: "hash_" + uuid.New().String()}
	err := f.APITokenRepo.Create(ctx, token)
	require.NoError(t, err)
	err = f.APITokenRepo.Delete(ctx, token.ID, user.ID)
	require.NoError(t, err)
	list, err := f.APITokenRepo.GetByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, list, 0)
}

func TestAPITokenRepo_Delete_Error_WrongUser(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "delerr")
	token := &entity.APIToken{UserID: user.ID, TokenHash: "hash_" + uuid.New().String()}
	err := f.APITokenRepo.Create(ctx, token)
	require.NoError(t, err)
	err = f.APITokenRepo.Delete(ctx, token.ID, uuid.New())
	require.NoError(t, err)
	list, err := f.APITokenRepo.GetByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestAPITokenRepo_UpdateLastUsedAt_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "updlast")
	token := &entity.APIToken{UserID: user.ID, TokenHash: "hash_" + uuid.New().String()}
	err := f.APITokenRepo.Create(ctx, token)
	require.NoError(t, err)
	err = f.APITokenRepo.UpdateLastUsedAt(ctx, token.ID, f.GetDefaultAppSettings(t).UpdatedAt)
	require.NoError(t, err)
}

func TestAPITokenRepo_UpdateLastUsedAt_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.APITokenRepo.UpdateLastUsedAt(ctx, uuid.New(), f.GetDefaultAppSettings(t).UpdatedAt)
	assert.NoError(t, err)
}
