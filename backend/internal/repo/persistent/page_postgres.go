package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	pageuc "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page"
)

type PageRepo struct {
	BaseRepo
}

var _ pageuc.PageRepository = (*PageRepo)(nil)
var _ challenge.PageReader = (*PageRepo)(nil)
var _ backup.PageRepository = (*PageRepo)(nil)

func NewPageRepo(pool *pgxpool.Pool) *PageRepo {
	return &PageRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *PageRepo) Create(ctx context.Context, page *domain.Page) error {
	EnsureID(&page.ID)

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
		IsDraft:    page.IsDraft,
		OrderIndex: orderIndex,
		CreatedAt:  pgutil.TimeToTimestamptz(&now),
		UpdatedAt:  pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return apperr.ErrPageSlugConflict
		}

		return fmt.Errorf("PageRepo - Create: %w", err)
	}

	page.CreatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt))
	page.UpdatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt))

	return nil
}

func (r *PageRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Page, error) {
	row, err := GetOrNotFound(func() (sqlc.Page, error) { return r.Q(ctx).GetPageByID(ctx, ID) },
		apperr.ErrPageNotFound, "PageRepo - GetByID")
	if err != nil {
		return nil, err
	}

	return toDomainPage(row), nil
}

func (r *PageRepo) GetBySlug(ctx context.Context, slug string) (*domain.Page, error) {
	row, err := GetOrNotFound(func() (sqlc.Page, error) { return r.Q(ctx).GetPageBySlug(ctx, slug) },
		apperr.ErrPageNotFound, "PageRepo - GetBySlug")
	if err != nil {
		return nil, err
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
		out[i] = &domain.PageListItem{
			ID:         row.ID,
			Title:      row.Title,
			Slug:       row.Slug,
			OrderIndex: int(row.OrderIndex),
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
		IsDraft:    page.IsDraft,
		OrderIndex: orderIndex,
		UpdatedAt:  pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return apperr.ErrPageSlugConflict
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
	return &domain.Page{
		ID:         row.ID,
		Title:      row.Title,
		Slug:       row.Slug,
		Content:    row.Content,
		IsDraft:    row.IsDraft,
		OrderIndex: int(row.OrderIndex),
		CreatedAt:  pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
		UpdatedAt:  pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.UpdatedAt)),
	}
}
