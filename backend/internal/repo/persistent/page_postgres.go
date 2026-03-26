package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type PageRepo struct {
	BaseRepo
}

var _ repo.PageRepository = (*PageRepo)(nil)

func NewPageRepo(pool *pgxpool.Pool) *PageRepo {
	return &PageRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *PageRepo) Create(ctx context.Context, page *domain.Page) error {
	EnsureID(&page.ID)
	isDraft := &page.IsDraft

	orderIndex, err := intToInt32Safe(page.OrderIndex)
	if err != nil {
		return fmt.Errorf("PageRepo - Create - OrderIndex: %w", err)
	}

	now := time.Now()

	row, err := r.Q(ctx).CreatePage(ctx, sqlc.CreatePageParams{
		ID:         page.ID,
		Title:      page.Title,
		Slug:       page.Slug,
		Content:    page.Content,
		IsDraft:    isDraft,
		OrderIndex: &orderIndex,
		CreatedAt:  pgutil.TimeToTimestamptz(&now),
		UpdatedAt:  pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return httperr.ErrPageSlugConflict
		}

		return fmt.Errorf("PageRepo - Create: %w", err)
	}

	page.CreatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt))
	page.UpdatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt))

	return nil
}

func (r *PageRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Page, error) {
	row, err := r.Q(ctx).GetPageByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrPageNotFound
		}

		return nil, fmt.Errorf("PageRepo - GetByID: %w", err)
	}

	return toDomainPage(row), nil
}

func (r *PageRepo) GetBySlug(ctx context.Context, slug string) (*domain.Page, error) {
	row, err := r.Q(ctx).GetPageBySlug(ctx, slug)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrPageNotFound
		}

		return nil, fmt.Errorf("PageRepo - GetBySlug: %w", err)
	}

	return toDomainPage(row), nil
}

func (r *PageRepo) GetPublishedList(ctx context.Context) ([]*domain.PageListItem, error) {
	rows, err := r.Q(ctx).GetPublishedPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageRepo - GetPublishedList: %w", err)
	}

	out := make([]*domain.PageListItem, len(rows))
	for i, row := range rows {
		orderIndex := 0

		if row.OrderIndex != nil {
			orderIndex = int(*row.OrderIndex)
		}

		out[i] = &domain.PageListItem{
			ID:         row.ID,
			Title:      row.Title,
			Slug:       row.Slug,
			OrderIndex: orderIndex,
		}
	}

	return out, nil
}

func (r *PageRepo) GetAllList(ctx context.Context) ([]*domain.Page, error) {
	rows, err := r.Q(ctx).GetAllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("PageRepo - GetAllList: %w", err)
	}

	out := make([]*domain.Page, len(rows))
	for i, row := range rows {
		out[i] = toDomainPage(row)
	}

	return out, nil
}

func (r *PageRepo) Update(ctx context.Context, page *domain.Page) error {
	isDraft := &page.IsDraft

	orderIndex, err := intToInt32Safe(page.OrderIndex)
	if err != nil {
		return fmt.Errorf("PageRepo - Update - OrderIndex: %w", err)
	}

	now := time.Now()

	err = r.Q(ctx).UpdatePage(ctx, sqlc.UpdatePageParams{
		ID:         page.ID,
		Title:      page.Title,
		Slug:       page.Slug,
		Content:    page.Content,
		IsDraft:    isDraft,
		OrderIndex: &orderIndex,
		UpdatedAt:  pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return httperr.ErrPageSlugConflict
		}

		return fmt.Errorf("PageRepo - Update: %w", err)
	}

	return nil
}

func (r *PageRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	err := r.Q(ctx).DeletePage(ctx, ID)
	if err != nil {
		return fmt.Errorf("PageRepo - Delete: %w", err)
	}

	return nil
}

func toDomainPage(row sqlc.Page) *domain.Page {
	orderIndex := 0

	if row.OrderIndex != nil {
		orderIndex = int(*row.OrderIndex)
	}

	isDraft := false

	if row.IsDraft != nil {
		isDraft = *row.IsDraft
	}

	return &domain.Page{
		ID:         row.ID,
		Title:      row.Title,
		Slug:       row.Slug,
		Content:    row.Content,
		IsDraft:    isDraft,
		OrderIndex: orderIndex,
		CreatedAt:  pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
		UpdatedAt:  pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt)),
	}
}
