package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APITokenRepo struct {
	pool *pgxpool.Pool
}

var _ repo.APITokenRepository = (*APITokenRepo)(nil)

func NewAPITokenRepo(pool *pgxpool.Pool) *APITokenRepo {
	return &APITokenRepo{pool: pool}
}

func (r *APITokenRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
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
	if err := r.q(ctx).CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
		ID:          token.ID,
		UserID:      token.UserID,
		TokenHash:   token.TokenHash,
		Description: desc,
		ExpiresAt:   expiresAt,
		CreatedAt:   createdAt,
	}); err != nil {
		return fmt.Errorf("APITokenRepo - Create: %w", err)
	}
	return nil
}

func (r *APITokenRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.APIToken, error) {
	rows, err := r.q(ctx).GetAPITokensByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("APITokenRepo - GetByUserID: %w", err)
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
	row, err := r.q(ctx).GetAPITokenByHash(ctx, tokenHash)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrAPITokenNotFound
		}
		return nil, fmt.Errorf("APITokenRepo - GetByTokenHash: %w", err)
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

func (r *APITokenRepo) Delete(ctx context.Context, ID, userID uuid.UUID) error {
	if err := r.q(ctx).DeleteAPIToken(ctx, sqlc.DeleteAPITokenParams{ID: ID, UserID: userID}); err != nil {
		return fmt.Errorf("APITokenRepo - Delete: %w", err)
	}
	return nil
}

func (r *APITokenRepo) UpdateLastUsedAt(ctx context.Context, ID uuid.UUID, at time.Time) error {
	if err := r.q(ctx).UpdateAPITokenLastUsed(ctx, sqlc.UpdateAPITokenLastUsedParams{
		ID:         ID,
		LastUsedAt: &at,
	}); err != nil {
		return fmt.Errorf("APITokenRepo - UpdateLastUsedAt: %w", err)
	}
	return nil
}
