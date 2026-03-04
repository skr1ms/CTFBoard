package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompetitionParamRepo_GetAll_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.CompetitionParamRepo.GetAll(ctx)
	require.NoError(t, err)
}

func TestCompetitionParamRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.CompetitionParamRepo.GetAll(ctx)
	assert.Error(t, err)
}

func TestCompetitionParamRepo_GetByKey_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	p := &entity.CompetitionParam{Key: "test_key_success", Value: "v", ValueType: entity.CompetitionParamTypeString}
	err := f.CompetitionParamRepo.Upsert(ctx, p)
	require.NoError(t, err)
	got, err := f.CompetitionParamRepo.GetByKey(ctx, p.Key)
	require.NoError(t, err)
	assert.Equal(t, p.Key, got.Key)
	assert.Equal(t, p.Value, got.Value)
}

func TestCompetitionParamRepo_GetByKey_Error_NotFound(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.CompetitionParamRepo.GetByKey(ctx, "nonexistent_key_xyz")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrCompetitionParamNotFound))
}

func TestCompetitionParamRepo_Upsert_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	p := &entity.CompetitionParam{Key: "upsert_key", Value: "v1", ValueType: entity.CompetitionParamTypeString}
	err := f.CompetitionParamRepo.Upsert(ctx, p)
	require.NoError(t, err)
	p.Value = "v2"
	err = f.CompetitionParamRepo.Upsert(ctx, p)
	require.NoError(t, err)
	got, err := f.CompetitionParamRepo.GetByKey(ctx, p.Key)
	require.NoError(t, err)
	assert.Equal(t, "v2", got.Value)
}

func TestCompetitionParamRepo_Upsert_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &entity.CompetitionParam{Key: "bad_ctx", Value: "x", ValueType: entity.CompetitionParamTypeString}
	err := f.CompetitionParamRepo.Upsert(ctx, p)
	assert.Error(t, err)
}

func TestCompetitionParamRepo_Delete_Success(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	p := &entity.CompetitionParam{Key: "del_key", Value: "v", ValueType: entity.CompetitionParamTypeString}
	err := f.CompetitionParamRepo.Upsert(ctx, p)
	require.NoError(t, err)
	err = f.CompetitionParamRepo.Delete(ctx, p.Key)
	require.NoError(t, err)
	_, err = f.CompetitionParamRepo.GetByKey(ctx, p.Key)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrCompetitionParamNotFound))
}

func TestCompetitionParamRepo_Delete_Error_NoRows(t *testing.T) {
	t.Parallel()
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.CompetitionParamRepo.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}
