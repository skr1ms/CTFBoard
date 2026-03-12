package page

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type PageUseCase struct {
	deps PageDeps
}

type PageDeps struct {
	PageRepo repo.PageRepository
}

var _ usecase.PageUseCase = (*PageUseCase)(nil)

func NewPageUseCase(deps PageDeps) *PageUseCase {
	return &PageUseCase{deps: deps}
}

func (uc *PageUseCase) GetPublishedList(ctx context.Context) ([]*entity.PageListItem, error) {
	list, err := uc.deps.PageRepo.GetPublishedList(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetPublishedList - PageRepo.GetPublishedList: %w", err)
	}
	return list, nil
}

func (uc *PageUseCase) GetBySlug(ctx context.Context, slug string) (*entity.Page, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, httperr.ErrPageSlugRequired
	}
	page, err := uc.deps.PageRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetBySlug - PageRepo.GetBySlug: %w", err)
	}
	if page == nil {
		return nil, httperr.ErrPageNotFound
	}
	if page.IsDraft {
		return nil, httperr.ErrPageNotFound
	}
	return page, nil
}

func (uc *PageUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Page, error) {
	page, err := uc.deps.PageRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetByID - PageRepo.GetByID: %w", err)
	}
	return page, nil
}

func (uc *PageUseCase) GetAllList(ctx context.Context) ([]*entity.Page, error) {
	list, err := uc.deps.PageRepo.GetAllList(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetAllList - PageRepo.GetAllList: %w", err)
	}
	return list, nil
}

func (uc *PageUseCase) Create(ctx context.Context, title, slug, content string, isDraft bool, orderIndex int) (*entity.Page, error) {
	title = strings.TrimSpace(title)
	slug = strings.TrimSpace(slug)
	if title == "" {
		return nil, httperr.ErrPageTitleRequired
	}
	if slug == "" {
		return nil, httperr.ErrPageSlugRequired
	}
	if !slugRe.MatchString(slug) {
		return nil, httperr.NewValidationErrorf("slug must match ^[a-z0-9]+(?:-[a-z0-9]+)*$")
	}
	page := &entity.Page{
		ID:         uuid.New(),
		Title:      title,
		Slug:       slug,
		Content:    content,
		IsDraft:    isDraft,
		OrderIndex: orderIndex,
	}
	if err := uc.deps.PageRepo.Create(ctx, page); err != nil {
		return nil, fmt.Errorf("PageUseCase - Create - PageRepo.Create: %w", err)
	}
	return page, nil
}

func (uc *PageUseCase) Update(ctx context.Context, ID uuid.UUID, title, slug, content string, isDraft bool, orderIndex int) (*entity.Page, error) {
	page, err := uc.deps.PageRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - Update - PageRepo.GetByID: %w", err)
	}
	title = strings.TrimSpace(title)
	slug = strings.TrimSpace(slug)
	if title == "" {
		return nil, httperr.ErrPageTitleRequired
	}
	if slug == "" {
		return nil, httperr.ErrPageSlugRequired
	}
	if !slugRe.MatchString(slug) {
		return nil, httperr.NewValidationErrorf("slug must match ^[a-z0-9]+(?:-[a-z0-9]+)*$")
	}
	existing, err := uc.deps.PageRepo.GetBySlug(ctx, slug)
	if err != nil && !errors.Is(err, httperr.ErrPageNotFound) {
		return nil, fmt.Errorf("PageUseCase - Update - PageRepo.GetBySlug: %w", err)
	}
	if existing != nil && existing.ID != ID {
		return nil, httperr.ErrPageSlugConflict
	}
	page.Title = title
	page.Slug = slug
	page.Content = content
	page.IsDraft = isDraft
	page.OrderIndex = orderIndex
	if err := uc.deps.PageRepo.Update(ctx, page); err != nil {
		return nil, fmt.Errorf("PageUseCase - Update - PageRepo.Update: %w", err)
	}
	return page, nil
}

func (uc *PageUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := uc.deps.PageRepo.Delete(ctx, ID); err != nil {
		return fmt.Errorf("PageUseCase - Delete - PageRepo.Delete: %w", err)
	}
	return nil
}
