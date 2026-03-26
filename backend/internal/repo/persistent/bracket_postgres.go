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

type BracketRepo struct {
	BaseRepo
}

var _ repo.BracketRepository = (*BracketRepo)(nil)

func NewBracketRepo(pool *pgxpool.Pool) *BracketRepo {
	return &BracketRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *BracketRepo) Create(ctx context.Context, bracket *domain.Bracket) error {
	EnsureID(&bracket.ID)

	if bracket.CreatedAt.IsZero() {
		bracket.CreatedAt = time.Now()
	}

	desc := &bracket.Description
	if bracket.Description == "" {
		desc = nil
	}

	isDefault := &bracket.IsDefault
	createdAt := &bracket.CreatedAt

	_, err := r.Q(ctx).CreateBracket(ctx, sqlc.CreateBracketParams{
		ID:          bracket.ID,
		Name:        bracket.Name,
		Description: desc,
		IsDefault:   isDefault,
		CreatedAt:   pgutil.TimeToTimestamptz(createdAt),
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return httperr.ErrBracketNameConflict
		}

		return fmt.Errorf("BracketRepo - Create: %w", err)
	}

	return nil
}

func (r *BracketRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Bracket, error) {
	row, err := r.Q(ctx).GetBracketByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrBracketNotFound
		}

		return nil, fmt.Errorf("BracketRepo - GetByID: %w", err)
	}

	return toDomainBracket(row), nil
}

func (r *BracketRepo) GetByName(ctx context.Context, name string) (*domain.Bracket, error) {
	row, err := r.Q(ctx).GetBracketByName(ctx, name)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrBracketNotFound
		}

		return nil, fmt.Errorf("BracketRepo - GetByName: %w", err)
	}

	return toDomainBracket(row), nil
}

func (r *BracketRepo) GetAll(ctx context.Context) ([]*domain.Bracket, error) {
	rows, err := r.Q(ctx).GetAllBrackets(ctx)
	if err != nil {
		return nil, fmt.Errorf("BracketRepo - GetAll: %w", err)
	}

	out := make([]*domain.Bracket, len(rows))
	for i := range rows {
		out[i] = toDomainBracket(rows[i])
	}

	return out, nil
}

func (r *BracketRepo) Update(ctx context.Context, bracket *domain.Bracket) error {
	desc := &bracket.Description
	if bracket.Description == "" {
		desc = nil
	}

	isDefault := &bracket.IsDefault

	err := r.Q(ctx).UpdateBracket(ctx, sqlc.UpdateBracketParams{
		ID:          bracket.ID,
		Name:        bracket.Name,
		Description: desc,
		IsDefault:   isDefault,
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return httperr.ErrBracketNameConflict
		}

		return fmt.Errorf("BracketRepo - Update: %w", err)
	}

	return nil
}

func (r *BracketRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	err := r.Q(ctx).DeleteBracket(ctx, ID)
	if err != nil {
		return fmt.Errorf("BracketRepo - Delete: %w", err)
	}

	return nil
}

func (r *BracketRepo) ClearAllDefaults(ctx context.Context) error {
	err := r.Q(ctx).ClearAllDefaultBrackets(ctx)
	if err != nil {
		return fmt.Errorf("BracketRepo - ClearAllDefaults: %w", err)
	}

	return nil
}

func toDomainBracket(row sqlc.Bracket) *domain.Bracket {
	desc := ""

	if row.Description != nil {
		desc = *row.Description
	}

	isDefault := false

	if row.IsDefault != nil {
		isDefault = *row.IsDefault
	}

	return &domain.Bracket{
		ID:          row.ID,
		Name:        row.Name,
		Description: desc,
		IsDefault:   isDefault,
		CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
	}
}
