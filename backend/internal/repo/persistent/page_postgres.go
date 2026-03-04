package persistent

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PageRepo struct {
	pool *pgxpool.Pool
}

var _ repo.PageRepository = (*PageRepo)(nil)

func NewPageRepo(pool *pgxpool.Pool) *PageRepo {
	return &PageRepo{pool: pool}
}

func (r *PageRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func (r *PageRepo) Create(ctx context.Context, page *entity.Page) error {
	if page.ID == uuid.Nil {
		page.ID = uuid.New()
	}
	isDraft := &page.IsDraft
	orderIndex, err := intToInt32Safe(page.OrderIndex)
	if err != nil {
		return fmt.Errorf("PageRepo - Create - OrderIndex: %w", err)
	}
	row, err := r.q(ctx).CreatePage(ctx, sqlc.CreatePageParams{
		ID:         page.ID,
		Title:      page.Title,
		Slug:       page.Slug,
		Content:    page.Content,
		IsDraft:    isDraft,
		OrderIndex: &orderIndex,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return httperr.ErrPageSlugConflict
		}
		return fmt.Errorf("PageRepo - Create: %w", err)
	}
	page.CreatedAt = ptrTimeToTime(row.CreatedAt)
	page.UpdatedAt = ptrTimeToTime(row.UpdatedAt)
	return nil
}

func (r *PageRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Page, error) {
	row, err := r.q(ctx).GetPageByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrPageNotFound
		}
		return nil, fmt.Errorf("PageRepo - GetByID: %w", err)
	}
	return toEntityPage(row), nil
}

func (r *PageRepo) GetBySlug(ctx context.Context, slug string) (*entity.Page, error) {
	row, err := r.q(ctx).GetPageBySlug(ctx, slug)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrPageNotFound
		}
		return nil, fmt.Errorf("PageRepo - GetBySlug: %w", err)
	}
	return toEntityPage(row), nil
}

func (r *PageRepo) GetPublishedList(ctx context.Context) ([]*entity.PageListItem, error) {
	rows, err := r.q(ctx).GetPublishedPages(ctx)
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
	rows, err := r.q(ctx).GetAllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageRepo - GetAllList: %w", err)
	}
	out := make([]*entity.Page, len(rows))
	for i, row := range rows {
		out[i] = toEntityPage(row)
	}
	return out, nil
}

func (r *PageRepo) Update(ctx context.Context, page *entity.Page) error {
	isDraft := &page.IsDraft
	orderIndex, err := intToInt32Safe(page.OrderIndex)
	if err != nil {
		return fmt.Errorf("PageRepo - Update - OrderIndex: %w", err)
	}
	err = r.q(ctx).UpdatePage(ctx, sqlc.UpdatePageParams{
		ID:         page.ID,
		Title:      page.Title,
		Slug:       page.Slug,
		Content:    page.Content,
		IsDraft:    isDraft,
		OrderIndex: &orderIndex,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return httperr.ErrPageSlugConflict
		}
		return fmt.Errorf("PageRepo - Update: %w", err)
	}
	return nil
}

func (r *PageRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := r.q(ctx).DeletePage(ctx, ID); err != nil {
		return fmt.Errorf("PageRepo - Delete: %w", err)
	}
	return nil
}

func toEntityPage(row sqlc.Page) *entity.Page {
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
