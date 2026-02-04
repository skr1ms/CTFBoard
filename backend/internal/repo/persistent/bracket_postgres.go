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

type BracketRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewBracketRepo(pool *pgxpool.Pool) *BracketRepo {
	return &BracketRepo{
		pool: pool,
		q:    sqlc.New(pool),
	}
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
	_, err := r.q.CreateBracket(ctx, sqlc.CreateBracketParams{
		ID:          bracket.ID,
		Name:        bracket.Name,
		Description: desc,
		IsDefault:   isDefault,
		CreatedAt:   createdAt,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return entityError.ErrBracketNameConflict
		}
		return fmt.Errorf("BracketRepo - Create: %w", err)
	}
	return nil
}

func (r *BracketRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Bracket, error) {
	row, err := r.q.GetBracketByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrBracketNotFound
		}
		return nil, fmt.Errorf("BracketRepo - GetByID: %w", err)
	}
	return bracketRowToEntity(row), nil
}

func (r *BracketRepo) GetByName(ctx context.Context, name string) (*entity.Bracket, error) {
	row, err := r.q.GetBracketByName(ctx, name)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrBracketNotFound
		}
		return nil, fmt.Errorf("BracketRepo - GetByName: %w", err)
	}
	return bracketRowToEntity(row), nil
}

func (r *BracketRepo) GetAll(ctx context.Context) ([]*entity.Bracket, error) {
	rows, err := r.q.GetAllBrackets(ctx)
	if err != nil {
		return nil, fmt.Errorf("BracketRepo - GetAll: %w", err)
	}
	out := make([]*entity.Bracket, len(rows))
	for i := range rows {
		out[i] = bracketRowToEntity(rows[i])
	}
	return out, nil
}

func (r *BracketRepo) Update(ctx context.Context, bracket *entity.Bracket) error {
	desc := &bracket.Description
	if bracket.Description == "" {
		desc = nil
	}
	isDefault := &bracket.IsDefault
	err := r.q.UpdateBracket(ctx, sqlc.UpdateBracketParams{
		ID:          bracket.ID,
		Name:        bracket.Name,
		Description: desc,
		IsDefault:   isDefault,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return entityError.ErrBracketNameConflict
		}
		return fmt.Errorf("BracketRepo - Update: %w", err)
	}
	return nil
}

func (r *BracketRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteBracket(ctx, id)
}

func bracketRowToEntity(row sqlc.Bracket) *entity.Bracket {
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
		CreatedAt:   ptrTimeToTime(row.CreatedAt),
	}
}
