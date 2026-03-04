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

type OAuthRepo struct {
	pool *pgxpool.Pool
}

var _ repo.OAuthAccountRepository = (*OAuthRepo)(nil)

func NewOAuthRepo(pool *pgxpool.Pool) *OAuthRepo {
	return &OAuthRepo{pool: pool}
}

func (r *OAuthRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func toEntityOAuthAccount(o sqlc.OAuthAccount) *entity.OAuthAccount {
	acc := &entity.OAuthAccount{
		ID:             o.ID,
		UserID:         o.UserID,
		Provider:       o.Provider,
		ProviderUserID: o.ProviderUserID,
		CreatedAt:      ptrTimeToTime(o.CreatedAt),
	}
	if o.AccessToken != nil {
		acc.AccessToken = *o.AccessToken
	}
	acc.RefreshToken = o.RefreshToken
	acc.ExpiresAt = o.ExpiresAt
	return acc
}

func (r *OAuthRepo) Create(ctx context.Context, acc *entity.OAuthAccount) error {
	if acc.ID == uuid.Nil {
		acc.ID = uuid.New()
	}
	var accessToken *string
	if acc.AccessToken != "" {
		accessToken = &acc.AccessToken
	}
	err := r.q(ctx).CreateOAuthAccount(ctx, sqlc.CreateOAuthAccountParams{
		ID:             acc.ID,
		UserID:         acc.UserID,
		Provider:       acc.Provider,
		ProviderUserID: acc.ProviderUserID,
		AccessToken:    accessToken,
		RefreshToken:   acc.RefreshToken,
		ExpiresAt:      acc.ExpiresAt,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return httperr.ErrOAuthAccountAlreadyLinked
		}
		return fmt.Errorf("OAuthRepo - Create: %w", err)
	}
	return nil
}

func (r *OAuthRepo) Upsert(ctx context.Context, acc *entity.OAuthAccount) error {
	if acc.ID == uuid.Nil {
		acc.ID = uuid.New()
	}
	var accessToken *string
	if acc.AccessToken != "" {
		accessToken = &acc.AccessToken
	}
	err := r.q(ctx).UpsertOAuthAccount(ctx, sqlc.UpsertOAuthAccountParams{
		ID:             acc.ID,
		UserID:         acc.UserID,
		Provider:       acc.Provider,
		ProviderUserID: acc.ProviderUserID,
		AccessToken:    accessToken,
		RefreshToken:   acc.RefreshToken,
		ExpiresAt:      acc.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("OAuthRepo - Upsert: %w", err)
	}
	return nil
}

func (r *OAuthRepo) GetByProvider(ctx context.Context, provider, providerUserID string) (*entity.OAuthAccount, error) {
	o, err := r.q(ctx).GetOAuthAccount(ctx, sqlc.GetOAuthAccountParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrOAuthAccountNotFound
		}
		return nil, fmt.Errorf("OAuthRepo - GetByProvider: %w", err)
	}
	return toEntityOAuthAccount(o), nil
}

func (r *OAuthRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.OAuthAccount, error) {
	rows, err := r.q(ctx).GetOAuthAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("OAuthRepo - GetByUserID: %w", err)
	}
	out := make([]*entity.OAuthAccount, len(rows))
	for i := range rows {
		out[i] = toEntityOAuthAccount(rows[i])
	}
	return out, nil
}
