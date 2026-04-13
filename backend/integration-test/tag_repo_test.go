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

func TestTagRepo_Create_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tag := &domain.Tag{Name: "web", Color: "#00ff00"}
	err := f.TagRepo.Create(ctx, tag)
	require.NoError(t, err)
	assert.NotEmpty(t, tag.ID)
}

func TestTagRepo_Create_Error_DuplicateName(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	t1 := f.CreateTag(t, "dup")
	tag2 := &domain.Tag{Name: t1.Name, Color: "#111"}
	err := f.TagRepo.Create(ctx, tag2)
	assert.Error(t, err)
}

func TestTagRepo_GetByID_Success(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.TagRepo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTagNotFound)
}

func TestTagRepo_GetByName_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tag := f.CreateTag(t, "getname")
	got, err := f.TagRepo.GetByName(ctx, tag.Name)
	require.NoError(t, err)
	assert.Equal(t, tag.Name, got.Name)
}

func TestTagRepo_GetByName_Error_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.TagRepo.GetByName(ctx, "nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTagNotFound)
}

func TestTagRepo_GetAll_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	t1 := f.CreateTag(t, "a")
	t2 := f.CreateTag(t, "b")
	list, err := f.TagRepo.GetAll(ctx)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, tag := range list {
		ids[tag.ID] = true
	}

	assert.True(t, ids[t1.ID], "tag 1 should be in GetAll result")
	assert.True(t, ids[t2.ID], "tag 2 should be in GetAll result")
}

func TestTagRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.TagRepo.GetAll(ctx)
	assert.Error(t, err)
}

func TestTagRepo_Update_Success(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	tag := f.CreateTag(t, "upderr")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.TagRepo.Update(ctx, tag)
	assert.Error(t, err)
}

func TestTagRepo_Delete_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	tag := f.CreateTag(t, "del")
	err := f.TagRepo.Delete(ctx, tag.ID)
	require.NoError(t, err)
	_, err = f.TagRepo.GetByID(ctx, tag.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTagNotFound)
}

func TestTagRepo_Delete_Error_NoRows(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.TagRepo.Delete(ctx, uuid.New())
	assert.NoError(t, err)
}

func TestTagRepo_GetByChallengeID_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "chtag", 100)
	tag := f.CreateTag(t, "ch")
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.SetTags(txCtx, challenge.ID, []uuid.UUID{tag.ID})
	})
	require.NoError(t, err)
	tags, err := f.TagRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 1)
	assert.Equal(t, tag.ID, tags[0].ID)
}

func TestTagRepo_GetByChallengeID_Error_Empty(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "nochtag", 100)
	tags, err := f.TagRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestTagRepo_SetChallengeTags_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "settags", 100)
	tag1 := f.CreateTag(t, "s1")
	tag2 := f.CreateTag(t, "s2")
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.SetTags(txCtx, challenge.ID, []uuid.UUID{tag1.ID, tag2.ID})
	})
	require.NoError(t, err)
	tags, err := f.TagRepo.GetByChallengeID(ctx, challenge.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 2)
}

func TestTagRepo_SetChallengeTags_Error_InvalidTagID(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	challenge := f.CreateChallenge(t, "invalidtag", 100)
	err := f.TM.Run(ctx, func(txCtx context.Context) error {
		return f.ChallengeRepo.SetTags(txCtx, challenge.ID, []uuid.UUID{uuid.New()})
	})
	assert.Error(t, err)
}
