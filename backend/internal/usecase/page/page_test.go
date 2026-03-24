package page

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	pageMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page/mock"
)

type pageTestDeps struct {
	pageRepo *pageMock.MockPageRepository
}

func newPageTestDeps(t *testing.T) *pageTestDeps {
	t.Helper()
	return &pageTestDeps{pageRepo: pageMock.NewMockPageRepository(t)}
}

func (d *pageTestDeps) createUseCase() *PageUseCase {
	return NewPageUseCase(PageDeps{PageRepo: d.pageRepo})
}

func newTestPage(title, slug, content string, isDraft bool, orderIndex int) *domain.Page {
	return &domain.Page{
		ID:         uuid.New(),
		Title:      title,
		Slug:       slug,
		Content:    content,
		IsDraft:    isDraft,
		OrderIndex: orderIndex,
	}
}

func newTestPageListItem(id uuid.UUID, title, slug string, orderIndex int) *domain.PageListItem {
	return &domain.PageListItem{
		ID:         id,
		Title:      title,
		Slug:       slug,
		OrderIndex: orderIndex,
	}
}

func TestPageUseCase_GetPublishedList_Success(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	list := []*domain.PageListItem{newTestPageListItem(uuid.New(), "t", "s", 0)}

	d.pageRepo.EXPECT().GetPublishedList(mock.Anything).Return(list, nil)

	uc := d.createUseCase()
	got, err := uc.GetPublishedList(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, list[0].Slug, got[0].Slug)
}

func TestPageUseCase_GetPublishedList_Error(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()

	d.pageRepo.EXPECT().GetPublishedList(mock.Anything).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.GetPublishedList(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_GetBySlug_Success(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	slug := "about"
	page := newTestPage("About", slug, "content", false, 0)

	d.pageRepo.EXPECT().GetBySlug(mock.Anything, slug).Return(page, nil)

	uc := d.createUseCase()
	got, err := uc.GetBySlug(ctx, slug)

	assert.NoError(t, err)
	assert.Equal(t, page.ID, got.ID)
	assert.Equal(t, slug, got.Slug)
}

func TestPageUseCase_GetBySlug_Error(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	slug := "about"

	d.pageRepo.EXPECT().GetBySlug(mock.Anything, slug).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.GetBySlug(ctx, slug)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_GetByID_Success(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	id := uuid.New()
	page := newTestPage("T", "s", "c", false, 0)
	page.ID = id

	d.pageRepo.EXPECT().GetByID(mock.Anything, id).Return(page, nil)

	uc := d.createUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestPageUseCase_GetByID_Error(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.pageRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.GetByID(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_GetAllList_Success(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	list := []*domain.Page{newTestPage("T", "s", "c", false, 0)}

	d.pageRepo.EXPECT().GetAllList(mock.Anything).Return(list, nil)

	uc := d.createUseCase()
	got, err := uc.GetAllList(ctx)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestPageUseCase_GetAllList_Error(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()

	d.pageRepo.EXPECT().GetAllList(mock.Anything).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.GetAllList(ctx)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_Create_Success(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	title, slug, content := "Title", "slug", "content"
	isDraft := false
	orderIndex := 1

	d.pageRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, p *domain.Page) {
		assert.Equal(t, title, p.Title)
		assert.Equal(t, slug, p.Slug)
		assert.Equal(t, content, p.Content)
		assert.Equal(t, isDraft, p.IsDraft)
		assert.Equal(t, orderIndex, p.OrderIndex)
	})

	uc := d.createUseCase()
	got, err := uc.Create(ctx, title, slug, content, false, orderIndex)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, title, got.Title)
}

func TestPageUseCase_Create_Error(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()

	d.pageRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createUseCase()
	got, err := uc.Create(ctx, "T", "s", "c", false, 0)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	id := uuid.New()
	page := newTestPage("Old", "old", "c", false, 0)
	page.ID = id
	title, slug, content := "New", "new", "body"
	orderIndex := 2

	d.pageRepo.EXPECT().GetByID(mock.Anything, id).Return(page, nil)
	d.pageRepo.EXPECT().GetBySlug(mock.Anything, slug).Return(nil, nil)
	d.pageRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, p *domain.Page) {
		assert.Equal(t, title, p.Title)
		assert.Equal(t, slug, p.Slug)
		assert.Equal(t, orderIndex, p.OrderIndex)
	})

	uc := d.createUseCase()
	got, err := uc.Update(ctx, id, title, slug, content, true, orderIndex)

	assert.NoError(t, err)
	assert.Equal(t, title, got.Title)
}

func TestPageUseCase_Update_Error(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.pageRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.Update(ctx, id, "T", "s", "c", false, 0)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestPageUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.pageRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := d.createUseCase()
	err := uc.Delete(ctx, id)

	assert.NoError(t, err)
}

func TestPageUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newPageTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.pageRepo.EXPECT().Delete(mock.Anything, id).Return(assert.AnError)

	uc := d.createUseCase()
	err := uc.Delete(ctx, id)

	assert.Error(t, err)
}
