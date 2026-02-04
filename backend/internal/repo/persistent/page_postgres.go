package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type PageRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewPageRepo(pool *pgxpool.Pool) *PageRepo {
	return &PageRepo{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *PageRepo) Create(ctx context.Context, page *entity.Page) error {
	if page.ID == uuid.Nil {
		page.ID = uuid.New()
	}
	if page.CreatedAt.IsZero() {
		page.CreatedAt = time.Now()
	}
	if page.UpdatedAt.IsZero() {
		page.UpdatedAt = page.CreatedAt
	}
	isDraft := &page.IsDraft
	orderIndex, err := intToInt32Safe(page.OrderIndex)
	if err != nil {
		return fmt.Errorf("PageRepo - Create OrderIndex: %w", err)
	}
	createdAt := &page.CreatedAt
	updatedAt := &page.UpdatedAt
	_, err = r.q.CreatePage(ctx, sqlc.CreatePageParams{
		ID:         page.ID,
		Title:      page.Title,
		Slug:       page.Slug,
		Content:    page.Content,
		IsDraft:    isDraft,
		OrderIndex: &orderIndex,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return entityError.ErrPageSlugConflict
		}
		return fmt.Errorf("PageRepo - Create: %w", err)
	}
	return nil
}

func (r *PageRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Page, error) {
	row, err := r.q.GetPageByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrPageNotFound
		}
		return nil, fmt.Errorf("PageRepo - GetByID: %w", err)
	}
	return pageRowToEntity(row), nil
}

func (r *PageRepo) GetBySlug(ctx context.Context, slug string) (*entity.Page, error) {
	row, err := r.q.GetPageBySlug(ctx, slug)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrPageNotFound
		}
		return nil, fmt.Errorf("PageRepo - GetBySlug: %w", err)
	}
	return pageRowToEntity(row), nil
}

func (r *PageRepo) GetPublishedList(ctx context.Context) ([]*entity.PageListItem, error) {
	rows, err := r.q.GetPublishedPagesList(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageRepo - GetPublishedList: %w", err)
	}
	out := make([]*entity.PageListItem, len(rows))
	for i, row := range rows {
		orderIndex := 0
		if row.OrderIndex != nil {
			orderIndex = int(*row.OrderIndex)
		}
		out[i] = &entity.PageListItem{
			ID:         row.ID,
			Title:      row.Title,
			Slug:       row.Slug,
			OrderIndex: orderIndex,
		}
	}
	return out, nil
}

func (r *PageRepo) GetAllList(ctx context.Context) ([]*entity.Page, error) {
	rows, err := r.q.GetAllPagesList(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageRepo - GetAllList: %w", err)
	}
	out := make([]*entity.Page, len(rows))
	for i, row := range rows {
		out[i] = pageRowToEntity(row)
	}
	return out, nil
}

func (r *PageRepo) Update(ctx context.Context, page *entity.Page) error {
	page.UpdatedAt = time.Now()
	isDraft := &page.IsDraft
	orderIndex, err := intToInt32Safe(page.OrderIndex)
	if err != nil {
		return fmt.Errorf("PageRepo - Update OrderIndex: %w", err)
	}
	err = r.q.UpdatePage(ctx, sqlc.UpdatePageParams{
		ID:         page.ID,
		Title:      page.Title,
		Slug:       page.Slug,
		Content:    page.Content,
		IsDraft:    isDraft,
		OrderIndex: &orderIndex,
		UpdatedAt:  &page.UpdatedAt,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return entityError.ErrPageSlugConflict
		}
		return fmt.Errorf("PageRepo - Update: %w", err)
	}
	return nil
}

func (r *PageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeletePage(ctx, id)
}

func pageRowToEntity(row sqlc.Page) *entity.Page {
	orderIndex := 0
	if row.OrderIndex != nil {
		orderIndex = int(*row.OrderIndex)
	}
	isDraft := false
	if row.IsDraft != nil {
		isDraft = *row.IsDraft
	}
	return &entity.Page{
		ID:         row.ID,
		Title:      row.Title,
		Slug:       row.Slug,
		Content:    row.Content,
		IsDraft:    isDraft,
		OrderIndex: orderIndex,
		CreatedAt:  ptrTimeToTime(row.CreatedAt),
		UpdatedAt:  ptrTimeToTime(row.UpdatedAt),
	}
}
