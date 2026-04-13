package page

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	pageMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page/mock"
)

func newUC(t *testing.T) (*PageUseCase, *pageMock.MockPageRepository) {
	t.Helper()

	repo := pageMock.NewMockPageRepository(t)
	uc := NewPageUseCase(PageDeps{PageRepo: repo})

	return uc, repo
}

func makePage(slug string, isDraft bool) *domain.Page {
	return &domain.Page{
		ID:      uuid.New(),
		Title:   "title",
		Slug:    slug,
		Content: "content",
		IsDraft: isDraft,
	}
}

func TestGetPublishedList_Success(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	want := []*domain.PageListItem{{ID: uuid.New(), Slug: "hello"}}
	repo.EXPECT().GetPublishedList(mock.Anything).Return(want, nil)

	got, err := uc.GetPublishedList(context.Background())

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetPublishedList_RepoError(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	repo.EXPECT().GetPublishedList(mock.Anything).Return(nil, errors.New("db error"))

	_, err := uc.GetPublishedList(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestGetBySlug_Success(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	page := makePage("my-page", false)
	repo.EXPECT().GetBySlug(mock.Anything, "my-page").Return(page, nil)

	got, err := uc.GetBySlug(context.Background(), "my-page")

	require.NoError(t, err)
	assert.Equal(t, "my-page", got.Slug)
}

func TestGetBySlug_EmptySlug_Error(t *testing.T) {
	t.Parallel()

	uc, _ := newUC(t)

	_, err := uc.GetBySlug(context.Background(), "   ")

	require.ErrorIs(t, err, apperr.ErrPageSlugRequired)
}

func TestGetBySlug_NotFound_NilPage_Error(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	repo.EXPECT().GetBySlug(mock.Anything, "missing").Return(nil, nil)

	_, err := uc.GetBySlug(context.Background(), "missing")

	require.ErrorIs(t, err, apperr.ErrPageNotFound)
}

func TestGetBySlug_Draft_Error(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	page := makePage("draft-page", true)
	repo.EXPECT().GetBySlug(mock.Anything, "draft-page").Return(page, nil)

	_, err := uc.GetBySlug(context.Background(), "draft-page")

	require.ErrorIs(t, err, apperr.ErrPageNotFound)
}

func TestGetBySlug_RepoError(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	repo.EXPECT().GetBySlug(mock.Anything, "slug").Return(nil, errors.New("db error"))

	_, err := uc.GetBySlug(context.Background(), "slug")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestGetByID_Success(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	id := uuid.New()
	page := makePage("slug", false)
	page.ID = id
	repo.EXPECT().GetByID(mock.Anything, id).Return(page, nil)

	got, err := uc.GetByID(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestGetAllList_Success(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	want := []*domain.Page{makePage("p1", false), makePage("p2", true)}
	repo.EXPECT().GetAllList(mock.Anything).Return(want, nil)

	got, err := uc.GetAllList(context.Background())

	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(p *domain.Page) bool {
		return p.Title == "My Page" && p.Slug == "my-page" && !p.IsDraft
	})).Return(nil)

	got, err := uc.Create(context.Background(), "My Page", "my-page", "content", false, 0)

	require.NoError(t, err)
	assert.Equal(t, "My Page", got.Title)
	assert.Equal(t, "my-page", got.Slug)
}

func TestCreate_TitleTrimmed(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(p *domain.Page) bool {
		return p.Title == "Trimmed Title"
	})).Return(nil)

	_, err := uc.Create(context.Background(), "  Trimmed Title  ", "valid-slug", "content", false, 0)

	require.NoError(t, err)
}

func TestCreate_EmptyTitle_Error(t *testing.T) {
	t.Parallel()

	uc, _ := newUC(t)

	_, err := uc.Create(context.Background(), "  ", "slug", "content", false, 0)

	require.ErrorIs(t, err, apperr.ErrPageTitleRequired)
}

func TestCreate_EmptySlug_Error(t *testing.T) {
	t.Parallel()

	uc, _ := newUC(t)

	_, err := uc.Create(context.Background(), "title", "  ", "content", false, 0)

	require.ErrorIs(t, err, apperr.ErrPageSlugRequired)
}

func TestCreate_InvalidSlug_Error(t *testing.T) {
	t.Parallel()

	uc, _ := newUC(t)

	for _, badSlug := range []string{"UPPERCASE", "has space", "-leading-dash", "trailing-dash-", "double--dash", "hello_world"} {
		_, err := uc.Create(context.Background(), "title", badSlug, "content", false, 0)
		require.Error(t, err, "expected error for slug %q", badSlug)
	}
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	id := uuid.New()
	existing := makePage("old-slug", false)
	existing.ID = id

	repo.EXPECT().GetByID(mock.Anything, id).Return(existing, nil)
	repo.EXPECT().GetBySlug(mock.Anything, "new-slug").Return(nil, nil)
	repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(p *domain.Page) bool {
		return p.Title == "new title" && p.Slug == "new-slug"
	})).Return(nil)

	got, err := uc.Update(context.Background(), id, "new title", "new-slug", "new content", false, 1)

	require.NoError(t, err)
	assert.Equal(t, "new title", got.Title)
	assert.Equal(t, "new-slug", got.Slug)
}

func TestUpdate_SlugConflict_Error(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	id := uuid.New()
	existing := makePage("old-slug", false)
	existing.ID = id

	other := makePage("taken-slug", false)
	other.ID = uuid.New()

	repo.EXPECT().GetByID(mock.Anything, id).Return(existing, nil)
	repo.EXPECT().GetBySlug(mock.Anything, "taken-slug").Return(other, nil)

	_, err := uc.Update(context.Background(), id, "title", "taken-slug", "content", false, 0)

	require.ErrorIs(t, err, apperr.ErrPageSlugConflict)
}

func TestUpdate_SameSlugSamePage_Success(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	id := uuid.New()
	existing := makePage("same-slug", false)
	existing.ID = id

	// GetBySlug returns the same page - not a conflict
	repo.EXPECT().GetByID(mock.Anything, id).Return(existing, nil)
	repo.EXPECT().GetBySlug(mock.Anything, "same-slug").Return(existing, nil)
	repo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)

	_, err := uc.Update(context.Background(), id, "title", "same-slug", "content", false, 0)

	require.NoError(t, err)
}

func TestUpdate_InvalidSlug_Error(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	id := uuid.New()
	existing := makePage("old-slug", false)
	existing.ID = id
	repo.EXPECT().GetByID(mock.Anything, id).Return(existing, nil)

	_, err := uc.Update(context.Background(), id, "title", "INVALID SLUG", "content", false, 0)

	require.Error(t, err)
}

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	id := uuid.New()
	repo.EXPECT().Delete(mock.Anything, id).Return(nil)

	err := uc.Delete(context.Background(), id)

	require.NoError(t, err)
}

func TestDelete_RepoError(t *testing.T) {
	t.Parallel()

	uc, repo := newUC(t)
	id := uuid.New()
	repo.EXPECT().Delete(mock.Anything, id).Return(errors.New("constraint error"))

	err := uc.Delete(context.Background(), id)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "constraint error")
}
