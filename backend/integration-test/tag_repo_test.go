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

func TestTagRepo_Create_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tag := &entity.Tag{Name: "web", Color: "#00ff00"}
	err := f.TagRepo.Create(ctx, tag)
	require.NoError(t, err)
	assert.NotEmpty(t, tag.ID)
}

func TestTagRepo_Create_Error_DuplicateName(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateTag(t, "dup")
	tag2 := &entity.Tag{Name: "tag_dup", Color: "#111"}
	err := f.TagRepo.Create(ctx, tag2)
	assert.Error(t, err)
}

func TestTagRepo_GetByID_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tag := f.CreateTag(t, "getid")
	got, err := f.TagRepo.GetByID(ctx, tag.ID)
	require.NoError(t, err)
	assert.Equal(t, tag.ID, got.ID)
	assert.Equal(t, tag.Name, got.Name)
}

func TestTagRepo_GetByID_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.TagRepo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTagNotFound))
}

func TestTagRepo_GetByName_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tag := f.CreateTag(t, "getname")
	got, err := f.TagRepo.GetByName(ctx, tag.Name)
	require.NoError(t, err)
	assert.Equal(t, tag.Name, got.Name)
}

func TestTagRepo_GetByName_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.TagRepo.GetByName(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTagNotFound))
}

func TestTagRepo_GetAll_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateTag(t, "a")
	f.CreateTag(t, "b")
	list, err := f.TagRepo.GetAll(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 2)
}

func TestTagRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.TagRepo.GetAll(ctx)
	assert.Error(t, err)
}

func TestTagRepo_Update_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tag := f.CreateTag(t, "upd")
	tag.Color = "#111111"
	err := f.TagRepo.Update(ctx, tag)
	require.NoError(t, err)
	got, err := f.TagRepo.GetByID(ctx, tag.ID)
	require.NoError(t, err)
	assert.Equal(t, "#111111", got.Color)
}

func TestTagRepo_Update_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	tag := f.CreateTag(t, "upderr")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.TagRepo.Update(ctx, tag)
	assert.Error(t, err)
}

func TestTagRepo_Delete_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tag := f.CreateTag(t, "del")
	err := f.TagRepo.Delete(ctx, tag.ID)
	require.NoError(t, err)
	_, err = f.TagRepo.GetByID(ctx, tag.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrTagNotFound))
}

func TestTagRepo_Delete_Error_NoRows(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TagRepo.Delete(ctx, uuid.New())
	assert.NoError(t, err)
}

func TestTagRepo_GetByChallengeID_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "chtag", 100)
	tag := f.CreateTag(t, "ch")
	err := f.TagRepo.SetChallengeTags(ctx, challenge.ID, []uuid.UUID{tag.ID})
	require.NoError(t, err)
	tags, err := f.TagRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 1)
	assert.Equal(t, tag.ID, tags[0].ID)
}

func TestTagRepo_GetByChallengeID_Error_Empty(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "nochtag", 100)
	tags, err := f.TagRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestTagRepo_SetChallengeTags_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "settags", 100)
	tag1 := f.CreateTag(t, "s1")
	tag2 := f.CreateTag(t, "s2")
	err := f.TagRepo.SetChallengeTags(ctx, challenge.ID, []uuid.UUID{tag1.ID, tag2.ID})
	require.NoError(t, err)
	tags, err := f.TagRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 2)
}

func TestTagRepo_SetChallengeTags_Error_InvalidTagID(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "invalidtag", 100)
	err := f.TagRepo.SetChallengeTags(ctx, challenge.ID, []uuid.UUID{uuid.New()})
	assert.Error(t, err)
}
