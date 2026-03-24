package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"

	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type APITokenRepo struct {
	BaseRepo
}

var _ repo.APITokenRepository = (*APITokenRepo)(nil)

func NewAPITokenRepo(pool *pgxpool.Pool) *APITokenRepo {
	return &APITokenRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *APITokenRepo) Create(ctx context.Context, token *domain.APIToken) error {
	EnsureID(&token.ID)
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}
	desc := lo.EmptyableToPtr(token.Description)
	var expiresAt *time.Time
	if token.ExpiresAt != nil && !token.ExpiresAt.IsZero() {
		expiresAt = token.ExpiresAt
	}
	createdAt := &token.CreatedAt
	if err := r.Q(ctx).CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
		ID:          token.ID,
		UserID:      token.UserID,
		TokenHash:   token.TokenHash,
		Description: desc,
		ExpiresAt:   pgutil.TimeToTimestamptz(expiresAt),
		CreatedAt:   pgutil.TimeToTimestamptz(createdAt),
	}); err != nil {
		return fmt.Errorf("APITokenRepo - Create: %w", err)
	}
	return nil
}

func (r *APITokenRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.APIToken, error) {
	rows, err := r.Q(ctx).GetAPITokensByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("APITokenRepo - GetByUserID: %w", err)
	}
	out := make([]*domain.APIToken, len(rows))
	for i, row := range rows {
		out[i] = &domain.APIToken{
			ID:          row.ID,
			UserID:      row.UserID,
			TokenHash:   row.TokenHash,
			Description: lo.FromPtr(row.Description),
			ExpiresAt:   pgutil.TimestamptzToTime(row.ExpiresAt),
			LastUsedAt:  pgutil.TimestamptzToTime(row.LastUsedAt),
			CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
		}
	}
	return out, nil
}

func (r *APITokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.APIToken, error) {
	row, err := r.Q(ctx).GetAPITokenByHash(ctx, tokenHash)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrAPITokenNotFound
		}
		return nil, fmt.Errorf("APITokenRepo - GetByTokenHash: %w", err)
	}
	return &domain.APIToken{
		ID:          row.ID,
		UserID:      row.UserID,
		TokenHash:   row.TokenHash,
		Description: lo.FromPtr(row.Description),
		ExpiresAt:   pgutil.TimestamptzToTime(row.ExpiresAt),
		LastUsedAt:  pgutil.TimestamptzToTime(row.LastUsedAt),
		CreatedAt:   pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
	}, nil
}

func (r *APITokenRepo) Delete(ctx context.Context, ID, userID uuid.UUID) error {
	if err := r.Q(ctx).DeleteAPIToken(ctx, sqlc.DeleteAPITokenParams{ID: ID, UserID: userID}); err != nil {
		return fmt.Errorf("APITokenRepo - Delete: %w", err)
	}
	return nil
}

func (r *APITokenRepo) UpdateLastUsedAt(ctx context.Context, ID uuid.UUID, at time.Time) error {
	if err := r.Q(ctx).UpdateAPITokenLastUsed(ctx, sqlc.UpdateAPITokenLastUsedParams{
		ID:         ID,
		LastUsedAt: pgutil.TimeToTimestamptz(&at),
	}); err != nil {
		return fmt.Errorf("APITokenRepo - UpdateLastUsedAt: %w", err)
	}
	return nil
}
