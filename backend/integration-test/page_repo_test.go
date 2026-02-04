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

func TestPageRepo_Create_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	page := &entity.Page{
		Title:      "About",
		Slug:       "about",
		Content:    "Content",
		IsDraft:    false,
		OrderIndex: 0,
	}
	err := f.PageRepo.Create(ctx, page)
	require.NoError(t, err)
	assert.NotEmpty(t, page.ID)
}

func TestPageRepo_Create_Error_DuplicateSlug(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreatePage(t, "dup", false)
	page2 := &entity.Page{Title: "Other", Slug: "page-dup", Content: "x", IsDraft: false, OrderIndex: 0}
	err := f.PageRepo.Create(ctx, page2)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrPageSlugConflict))
}

func TestPageRepo_GetByID_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	page := f.CreatePage(t, "gbi", false)
	got, err := f.PageRepo.GetByID(ctx, page.ID)
	require.NoError(t, err)
	assert.Equal(t, page.ID, got.ID)
	assert.Equal(t, page.Slug, got.Slug)
}

func TestPageRepo_GetByID_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.PageRepo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrPageNotFound))
}

func TestPageRepo_GetBySlug_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	page := f.CreatePage(t, "gbs", false)
	got, err := f.PageRepo.GetBySlug(ctx, page.Slug)
	require.NoError(t, err)
	assert.Equal(t, page.ID, got.ID)
}

func TestPageRepo_GetBySlug_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.PageRepo.GetBySlug(ctx, "nonexistent-slug-xyz")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrPageNotFound))
}

func TestPageRepo_GetPublishedList_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreatePage(t, "pub", false)
	list, err := f.PageRepo.GetPublishedList(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestPageRepo_GetPublishedList_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.PageRepo.GetPublishedList(ctx)
	assert.Error(t, err)
}

func TestPageRepo_GetAllList_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreatePage(t, "gal", true)
	list, err := f.PageRepo.GetAllList(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestPageRepo_GetAllList_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.PageRepo.GetAllList(ctx)
	assert.Error(t, err)
}

func TestPageRepo_Update_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	page := f.CreatePage(t, "upd", false)
	page.Title = "Updated Title"
	err := f.PageRepo.Update(ctx, page)
	require.NoError(t, err)
	got, err := f.PageRepo.GetByID(ctx, page.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", got.Title)
}

func TestPageRepo_Update_Error_DuplicateSlug(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	p1 := f.CreatePage(t, "u1", false)
	p2 := f.CreatePage(t, "u2", false)
	p2.Slug = p1.Slug
	err := f.PageRepo.Update(ctx, p2)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrPageSlugConflict))
}

func TestPageRepo_Delete_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	page := f.CreatePage(t, "del", false)
	err := f.PageRepo.Delete(ctx, page.ID)
	require.NoError(t, err)
	_, err = f.PageRepo.GetByID(ctx, page.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrPageNotFound))
}

func TestPageRepo_Delete_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.PageRepo.Delete(ctx, uuid.New())
	assert.NoError(t, err)
}
