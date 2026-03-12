package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type BracketRepo struct {
	pool *pgxpool.Pool
}

var _ repo.BracketRepository = (*BracketRepo)(nil)

func NewBracketRepo(pool *pgxpool.Pool) *BracketRepo {
	return &BracketRepo{pool: pool}
}

func (r *BracketRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func (r *BracketRepo) Create(ctx context.Context, bracket *entity.Bracket) error {
	if bracket.ID == uuid.Nil {
		bracket.ID = uuid.New()
	}
	if bracket.CreatedAt.IsZero() {
		bracket.CreatedAt = time.Now()
	}
	desc := &bracket.Description
	if bracket.Description == "" {
		desc = nil
	}
	isDefault := &bracket.IsDefault
	createdAt := &bracket.CreatedAt
	_, err := r.q(ctx).CreateBracket(ctx, sqlc.CreateBracketParams{
		ID:          bracket.ID,
		Name:        bracket.Name,
		Description: desc,
		IsDefault:   isDefault,
		CreatedAt:   timeToTimestamptz(createdAt),
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return httperr.ErrBracketNameConflict
		}
		return fmt.Errorf("BracketRepo - Create: %w", err)
	}
	return nil
}

func (r *BracketRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.Bracket, error) {
	row, err := r.q(ctx).GetBracketByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrBracketNotFound
		}
		return nil, fmt.Errorf("BracketRepo - GetByID: %w", err)
	}
	return toEntityBracket(row), nil
}

func (r *BracketRepo) GetByName(ctx context.Context, name string) (*entity.Bracket, error) {
	row, err := r.q(ctx).GetBracketByName(ctx, name)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrBracketNotFound
		}
		return nil, fmt.Errorf("BracketRepo - GetByName: %w", err)
	}
	return toEntityBracket(row), nil
}

func (r *BracketRepo) GetAll(ctx context.Context) ([]*entity.Bracket, error) {
	rows, err := r.q(ctx).GetAllBrackets(ctx)
	if err != nil {
		return nil, fmt.Errorf("BracketRepo - GetAll: %w", err)
	}
	out := make([]*entity.Bracket, len(rows))
	for i := range rows {
		out[i] = toEntityBracket(rows[i])
	}
	return out, nil
}

func (r *BracketRepo) Update(ctx context.Context, bracket *entity.Bracket) error {
	desc := &bracket.Description
	if bracket.Description == "" {
		desc = nil
	}
	isDefault := &bracket.IsDefault
	err := r.q(ctx).UpdateBracket(ctx, sqlc.UpdateBracketParams{
		ID:          bracket.ID,
		Name:        bracket.Name,
		Description: desc,
		IsDefault:   isDefault,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return httperr.ErrBracketNameConflict
		}
		return fmt.Errorf("BracketRepo - Update: %w", err)
	}
	return nil
}

func (r *BracketRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := r.q(ctx).DeleteBracket(ctx, ID); err != nil {
		return fmt.Errorf("BracketRepo - Delete: %w", err)
	}
	return nil
}

func (r *BracketRepo) ClearAllDefaults(ctx context.Context) error {
	if err := r.q(ctx).ClearAllDefaultBrackets(ctx); err != nil {
		return fmt.Errorf("BracketRepo - ClearAllDefaults: %w", err)
	}
	return nil
}

func toEntityBracket(row sqlc.Bracket) *entity.Bracket {
	desc := ""
	if row.Description != nil {
		desc = *row.Description
	}
	isDefault := false
	if row.IsDefault != nil {
		isDefault = *row.IsDefault
	}
	return &entity.Bracket{
		ID:          row.ID,
		Name:        row.Name,
		Description: desc,
		IsDefault:   isDefault,
		CreatedAt:   ptrTimeToTime(timestamptzToTime(row.CreatedAt)),
	}
}
