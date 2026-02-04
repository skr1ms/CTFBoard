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

func TestBracketRepo_Create_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	bracket := &entity.Bracket{Name: "Pro", Description: "Pro bracket", IsDefault: false}
	err := f.BracketRepo.Create(ctx, bracket)
	require.NoError(t, err)
	assert.NotEmpty(t, bracket.ID)
}

func TestBracketRepo_Create_Error_DuplicateName(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateBracket(t, "dup")
	bracket2 := &entity.Bracket{Name: "bracket_dup", Description: "x", IsDefault: false}
	err := f.BracketRepo.Create(ctx, bracket2)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrBracketNameConflict))
}

func TestBracketRepo_GetByID_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	bracket := f.CreateBracket(t, "gbi")
	got, err := f.BracketRepo.GetByID(ctx, bracket.ID)
	require.NoError(t, err)
	assert.Equal(t, bracket.ID, got.ID)
	assert.Equal(t, bracket.Name, got.Name)
}

func TestBracketRepo_GetByID_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.BracketRepo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrBracketNotFound))
}

func TestBracketRepo_GetByName_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	bracket := f.CreateBracket(t, "gbn")
	got, err := f.BracketRepo.GetByName(ctx, bracket.Name)
	require.NoError(t, err)
	assert.Equal(t, bracket.ID, got.ID)
}

func TestBracketRepo_GetByName_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.BracketRepo.GetByName(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrBracketNotFound))
}

func TestBracketRepo_GetAll_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateBracket(t, "a")
	f.CreateBracket(t, "b")
	list, err := f.BracketRepo.GetAll(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 2)
}

func TestBracketRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.BracketRepo.GetAll(ctx)
	assert.Error(t, err)
}

func TestBracketRepo_Update_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	bracket := f.CreateBracket(t, "upd")
	bracket.Description = "updated desc"
	err := f.BracketRepo.Update(ctx, bracket)
	require.NoError(t, err)
	got, err := f.BracketRepo.GetByID(ctx, bracket.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated desc", got.Description)
}

func TestBracketRepo_Update_Error_DuplicateName(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	b1 := f.CreateBracket(t, "u1")
	b2 := f.CreateBracket(t, "u2")
	b2.Name = b1.Name
	err := f.BracketRepo.Update(ctx, b2)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrBracketNameConflict))
}

func TestBracketRepo_Delete_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	bracket := f.CreateBracket(t, "del")
	err := f.BracketRepo.Delete(ctx, bracket.ID)
	require.NoError(t, err)
	_, err = f.BracketRepo.GetByID(ctx, bracket.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrBracketNotFound))
}

func TestBracketRepo_Delete_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.BracketRepo.Delete(ctx, uuid.New())
	assert.NoError(t, err)
}
