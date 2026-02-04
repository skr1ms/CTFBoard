package page

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPageUseCase_GetPublishedList_Success(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	list := []*entity.PageListItem{h.NewPageListItem(uuid.New(), "t", "s", 0)}

	deps.pageRepo.EXPECT().GetPublishedList(mock.Anything).Return(list, nil)

	uc := h.CreateUseCase()
	got, err := uc.GetPublishedList(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, list[0].Slug, got[0].Slug)
}

func TestPageUseCase_GetPublishedList_Error(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.pageRepo.EXPECT().GetPublishedList(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.GetPublishedList(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_GetBySlug_Success(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	slug := "about"
	page := h.NewPage("About", slug, "content", false, 0)

	deps.pageRepo.EXPECT().GetBySlug(mock.Anything, slug).Return(page, nil)

	uc := h.CreateUseCase()
	got, err := uc.GetBySlug(ctx, slug)

	assert.NoError(t, err)
	assert.Equal(t, page.ID, got.ID)
	assert.Equal(t, slug, got.Slug)
}

func TestPageUseCase_GetBySlug_Error(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	slug := "about"

	deps.pageRepo.EXPECT().GetBySlug(mock.Anything, slug).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.GetBySlug(ctx, slug)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_GetByID_Success(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()
	page := h.NewPage("T", "s", "c", false, 0)
	page.ID = id

	deps.pageRepo.EXPECT().GetByID(mock.Anything, id).Return(page, nil)

	uc := h.CreateUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestPageUseCase_GetByID_Error(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.pageRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_GetAllList_Success(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	list := []*entity.Page{h.NewPage("T", "s", "c", false, 0)}

	deps.pageRepo.EXPECT().GetAllList(mock.Anything).Return(list, nil)

	uc := h.CreateUseCase()
	got, err := uc.GetAllList(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestPageUseCase_GetAllList_Error(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.pageRepo.EXPECT().GetAllList(mock.Anything).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.GetAllList(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_Create_Success(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	title, slug, content := "Title", "slug", "content"
	isDraft := false
	orderIndex := 1

	deps.pageRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, p *entity.Page) {
		assert.Equal(t, title, p.Title)
		assert.Equal(t, slug, p.Slug)
		assert.Equal(t, content, p.Content)
		assert.Equal(t, isDraft, p.IsDraft)
		assert.Equal(t, orderIndex, p.OrderIndex)
	})

	uc := h.CreateUseCase()
	got, err := uc.Create(ctx, title, slug, content, isDraft, orderIndex)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, title, got.Title)
}

func TestPageUseCase_Create_Error(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.pageRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.Create(ctx, "T", "s", "c", false, 0)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_Update_Success(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()
	page := h.NewPage("Old", "old", "c", false, 0)
	page.ID = id
	title, slug, content := "New", "new", "body"
	isDraft := true
	orderIndex := 2

	deps.pageRepo.EXPECT().GetByID(mock.Anything, id).Return(page, nil)
	deps.pageRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, p *entity.Page) {
		assert.Equal(t, title, p.Title)
		assert.Equal(t, slug, p.Slug)
		assert.Equal(t, orderIndex, p.OrderIndex)
	})

	uc := h.CreateUseCase()
	got, err := uc.Update(ctx, id, title, slug, content, isDraft, orderIndex)

	assert.NoError(t, err)
	assert.Equal(t, title, got.Title)
}

func TestPageUseCase_Update_Error(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.pageRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.Update(ctx, id, "T", "s", "c", false, 0)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_Delete_Success(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.pageRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := h.CreateUseCase()
	err := uc.Delete(ctx, id)

	assert.NoError(t, err)
}

func TestPageUseCase_Delete_Error(t *testing.T) {
	h := NewPageTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.pageRepo.EXPECT().Delete(mock.Anything, id).Return(assert.AnError)

	uc := h.CreateUseCase()
	err := uc.Delete(ctx, id)

	assert.Error(t, err)
}
