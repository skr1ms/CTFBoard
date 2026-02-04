package persistent

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type APITokenRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewAPITokenRepo(pool *pgxpool.Pool) *APITokenRepo {
	return &APITokenRepo{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *APITokenRepo) Create(ctx context.Context, token *entity.APIToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}
	desc := strPtrOrNil(token.Description)
	var expiresAt *time.Time
	if token.ExpiresAt != nil && !token.ExpiresAt.IsZero() {
		expiresAt = token.ExpiresAt
	}
	createdAt := &token.CreatedAt
	return r.q.CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
		ID:          token.ID,
		UserID:      token.UserID,
		TokenHash:   token.TokenHash,
		Description: desc,
		ExpiresAt:   expiresAt,
		CreatedAt:   createdAt,
	})
}

func (r *APITokenRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.APIToken, error) {
	rows, err := r.q.GetAPITokensByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.APIToken, len(rows))
	for i, row := range rows {
		out[i] = &entity.APIToken{
			ID:          row.ID,
			UserID:      row.UserID,
			TokenHash:   row.TokenHash,
			Description: ptrStrToStr(row.Description),
			ExpiresAt:   row.ExpiresAt,
			LastUsedAt:  row.LastUsedAt,
			CreatedAt:   ptrTimeToTime(row.CreatedAt),
		}
	}
	return out, nil
}

func (r *APITokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*entity.APIToken, error) {
	row, err := r.q.GetAPITokenByHash(ctx, tokenHash)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &entity.APIToken{
		ID:          row.ID,
		UserID:      row.UserID,
		TokenHash:   row.TokenHash,
		Description: ptrStrToStr(row.Description),
		ExpiresAt:   row.ExpiresAt,
		LastUsedAt:  row.LastUsedAt,
		CreatedAt:   ptrTimeToTime(row.CreatedAt),
	}, nil
}

func (r *APITokenRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteAPIToken(ctx, sqlc.DeleteAPITokenParams{ID: id, UserID: userID})
}

func (r *APITokenRepo) UpdateLastUsedAt(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.q.UpdateAPITokenLastUsed(ctx, sqlc.UpdateAPITokenLastUsedParams{
		ID:         id,
		LastUsedAt: &at,
	})
}
