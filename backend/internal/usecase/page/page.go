package page

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type PageUseCase struct {
	pageRepo repo.PageRepository
}

func NewPageUseCase(
	pageRepo repo.PageRepository,
) *PageUseCase {
	return &PageUseCase{pageRepo: pageRepo}
}

func (uc *PageUseCase) GetPublishedList(ctx context.Context) ([]*entity.PageListItem, error) {
	list, err := uc.pageRepo.GetPublishedList(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetPublishedList: %w", err)
	}
	return list, nil
}

func (uc *PageUseCase) GetBySlug(ctx context.Context, slug string) (*entity.Page, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, fmt.Errorf("PageUseCase - GetBySlug: slug is required")
	}
	page, err := uc.pageRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetBySlug: %w", err)
	}
	if page.IsDraft {
		return nil, entityError.ErrPageNotFound
	}
	return page, nil
}

func (uc *PageUseCase) GetByID(ctx context.Context, id uuid.UUID) (*entity.Page, error) {
	page, err := uc.pageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetByID: %w", err)
	}
	return page, nil
}

func (uc *PageUseCase) GetAllList(ctx context.Context) ([]*entity.Page, error) {
	list, err := uc.pageRepo.GetAllList(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - GetAllList: %w", err)
	}
	return list, nil
}

func (uc *PageUseCase) Create(ctx context.Context, title, slug, content string, isDraft bool, orderIndex int) (*entity.Page, error) {
	title = strings.TrimSpace(title)
	slug = strings.TrimSpace(slug)
	if title == "" {
		return nil, fmt.Errorf("PageUseCase - Create: title is required")
	}
	if slug == "" {
		return nil, fmt.Errorf("PageUseCase - Create: slug is required")
	}
	page := &entity.Page{
		ID:         uuid.New(),
		Title:      title,
		Slug:       slug,
		Content:    content,
		IsDraft:    isDraft,
		OrderIndex: orderIndex,
	}
	if err := uc.pageRepo.Create(ctx, page); err != nil {
		return nil, fmt.Errorf("PageUseCase - Create: %w", err)
	}
	return page, nil
}

func (uc *PageUseCase) Update(ctx context.Context, id uuid.UUID, title, slug, content string, isDraft bool, orderIndex int) (*entity.Page, error) {
	page, err := uc.pageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("PageUseCase - Update - GetByID: %w", err)
	}
	title = strings.TrimSpace(title)
	slug = strings.TrimSpace(slug)
	if title == "" {
		return nil, fmt.Errorf("PageUseCase - Update: title is required")
	}
	if slug == "" {
		return nil, fmt.Errorf("PageUseCase - Update: slug is required")
	}
	page.Title = title
	page.Slug = slug
	page.Content = content
	page.IsDraft = isDraft
	page.OrderIndex = orderIndex
	if err := uc.pageRepo.Update(ctx, page); err != nil {
		return nil, fmt.Errorf("PageUseCase - Update: %w", err)
	}
	return page, nil
}

func (uc *PageUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := uc.pageRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("PageUseCase - Delete: %w", err)
	}
	return nil
}
