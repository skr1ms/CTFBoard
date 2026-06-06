package page

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	slugpkg "github.com/TakuyaYagam1/AstroCTFb/pkg/slug"
)

type PageUseCase struct {
	deps PageDeps
}

type PageRepository interface {
	Create(ctx context.Context, page *domain.Page) error
	GetByID(ctx context.Context, ID uuid.UUID) (*domain.Page, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Page, error)
	GetPublishedList(ctx context.Context) ([]*domain.PageListItem, error)
	GetAllList(ctx context.Context) ([]*domain.Page, error)
	Update(ctx context.Context, page *domain.Page) error
	Delete(ctx context.Context, ID uuid.UUID) error
}

type PageDeps struct {
	PageRepo PageRepository
	Logger   logkit.Logger
}

var _ usecase.PageUseCase = (*PageUseCase)(nil)

func NewPageUseCase(deps PageDeps) *PageUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &PageUseCase{deps: deps}
}

func (uc *PageUseCase) GetPublishedList(ctx context.Context) ([]*domain.PageListItem, error) {
	list, err := uc.deps.PageRepo.GetPublishedList(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetPublishedList - PageRepo.GetPublishedList: %w", err)
	}

	return list, nil
}

func (uc *PageUseCase) GetBySlug(ctx context.Context, slug string) (*domain.Page, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, apperr.ErrPageSlugRequired
	}

	page, err := uc.deps.PageRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetBySlug - PageRepo.GetBySlug: %w", err)
	}

	if page == nil {
		return nil, apperr.ErrPageNotFound
	}

	if page.IsDraft {
		return nil, apperr.ErrPageNotFound
	}

	return page, nil
}

func (uc *PageUseCase) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Page, error) {
	page, err := uc.deps.PageRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetByID - PageRepo.GetByID: %w", err)
	}

	return page, nil
}

func (uc *PageUseCase) GetAllList(ctx context.Context) ([]*domain.Page, error) {
	list, err := uc.deps.PageRepo.GetAllList(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetAllList - PageRepo.GetAllList: %w", err)
	}

	return list, nil
}

func (uc *PageUseCase) Create(ctx context.Context, title, slug, content string, isDraft bool, orderIndex int) (*domain.Page, error) {
	title = strings.TrimSpace(title)
	slug = strings.TrimSpace(slug)

	if title == "" {
		return nil, apperr.ErrPageTitleRequired
	}

	if slug == "" {
		return nil, apperr.ErrPageSlugRequired
	}

	if !slugpkg.MatchPageSlug(slug) {
		return nil, apperr.NewValidationErrorf("slug must match %q", slugpkg.PageSlugPatternString)
	}

	page := &domain.Page{
		ID:         uuid.New(),
		Title:      title,
		Slug:       slug,
		Content:    content,
		IsDraft:    isDraft,
		OrderIndex: orderIndex,
	}

	err := uc.deps.PageRepo.Create(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - Create - PageRepo.Create: %w", err)
	}

	return page, nil
}

func (uc *PageUseCase) Update(ctx context.Context, ID uuid.UUID, title, slug, content string, isDraft bool, orderIndex int) (*domain.Page, error) {
	page, err := uc.deps.PageRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - Update - PageRepo.GetByID: %w", err)
	}

	title = strings.TrimSpace(title)
	slug = strings.TrimSpace(slug)

	if title == "" {
		return nil, apperr.ErrPageTitleRequired
	}

	if slug == "" {
		return nil, apperr.ErrPageSlugRequired
	}

	if !slugpkg.MatchPageSlug(slug) {
		return nil, apperr.NewValidationErrorf("slug must match %q", slugpkg.PageSlugPatternString)
	}

	existing, err := uc.deps.PageRepo.GetBySlug(ctx, slug)
	if err != nil && !errors.Is(err, apperr.ErrPageNotFound) {
		return nil, fmt.Errorf("PageUseCase - Update - PageRepo.GetBySlug: %w", err)
	}

	if existing != nil && existing.ID != ID {
		return nil, apperr.ErrPageSlugConflict
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
	err := uc.deps.PageRepo.Delete(ctx, ID)
	if err != nil {
		return fmt.Errorf("PageUseCase - Delete - PageRepo.Delete: %w", err)
	}

	return nil
}
