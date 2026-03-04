package page

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page/mocks"
	"github.com/google/uuid"
)

type PageTestHelper struct {
	t    *testing.T
	deps *pageTestDeps
}

type pageTestDeps struct {
	pageRepo *mocks.MockPageRepository
}

func NewPageTestHelper(t *testing.T) *PageTestHelper {
	t.Helper()
	return &PageTestHelper{
		t: t,
		deps: &pageTestDeps{
			pageRepo: mocks.NewMockPageRepository(t),
		},
	}
}

func (h *PageTestHelper) Deps() *pageTestDeps {
	h.t.Helper()
	return h.deps
}

func (h *PageTestHelper) CreateUseCase() *PageUseCase {
	h.t.Helper()
	return NewPageUseCase(PageDeps{PageRepo: h.deps.pageRepo})
}

func (h *PageTestHelper) NewPage(title, slug, content string, isDraft bool, orderIndex int) *entity.Page {
	h.t.Helper()
	return &entity.Page{
		ID:         uuid.New(),
		Title:      title,
		Slug:       slug,
		Content:    content,
		IsDraft:    isDraft,
		OrderIndex: orderIndex,
	}
}

func (h *PageTestHelper) NewPageListItem(id uuid.UUID, title, slug string, orderIndex int) *entity.PageListItem {
	h.t.Helper()
	return &entity.PageListItem{
		ID:         id,
		Title:      title,
		Slug:       slug,
		OrderIndex: orderIndex,
	}
}
