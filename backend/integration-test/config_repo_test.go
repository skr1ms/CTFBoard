package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigRepo_GetAll_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.ConfigRepo.GetAll(ctx)
	require.NoError(t, err)
}

func TestConfigRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.ConfigRepo.GetAll(ctx)
	assert.Error(t, err)
}

func TestConfigRepo_GetByKey_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	cfg := &entity.Config{Key: "test_key_success", Value: "v", ValueType: entity.ConfigTypeString}
	err := f.ConfigRepo.Upsert(ctx, cfg)
	require.NoError(t, err)
	got, err := f.ConfigRepo.GetByKey(ctx, cfg.Key)
	require.NoError(t, err)
	assert.Equal(t, cfg.Key, got.Key)
	assert.Equal(t, cfg.Value, got.Value)
}

func TestConfigRepo_GetByKey_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.ConfigRepo.GetByKey(ctx, "nonexistent_key_xyz")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrConfigNotFound))
}

func TestConfigRepo_Upsert_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	cfg := &entity.Config{Key: "upsert_key", Value: "v1", ValueType: entity.ConfigTypeString}
	err := f.ConfigRepo.Upsert(ctx, cfg)
	require.NoError(t, err)
	cfg.Value = "v2"
	err = f.ConfigRepo.Upsert(ctx, cfg)
	require.NoError(t, err)
	got, err := f.ConfigRepo.GetByKey(ctx, cfg.Key)
	require.NoError(t, err)
	assert.Equal(t, "v2", got.Value)
}

func TestConfigRepo_Upsert_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := &entity.Config{Key: "bad_ctx", Value: "x", ValueType: entity.ConfigTypeString}
	err := f.ConfigRepo.Upsert(ctx, cfg)
	assert.Error(t, err)
}

func TestConfigRepo_Delete_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	cfg := &entity.Config{Key: "del_key", Value: "v", ValueType: entity.ConfigTypeString}
	err := f.ConfigRepo.Upsert(ctx, cfg)
	require.NoError(t, err)
	err = f.ConfigRepo.Delete(ctx, cfg.Key)
	require.NoError(t, err)
	_, err = f.ConfigRepo.GetByKey(ctx, cfg.Key)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrConfigNotFound))
}

func TestConfigRepo_Delete_Error_NoRows(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.ConfigRepo.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}
