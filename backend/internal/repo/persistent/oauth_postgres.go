package persistent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type OAuthRepo struct {
	BaseRepo
}

var _ repo.OAuthAccountRepository = (*OAuthRepo)(nil)

func NewOAuthRepo(pool *pgxpool.Pool) *OAuthRepo {
	return &OAuthRepo{BaseRepo: BaseRepo{pool: pool}}
}

func toDomainOAuthAccount(o sqlc.OAuthAccount) *domain.OAuthAccount {
	acc := &domain.OAuthAccount{
		ID:             o.ID,
		UserID:         o.UserID,
		Provider:       o.Provider,
		ProviderUserID: o.ProviderUserID,
		CreatedAt:      pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(o.CreatedAt)),
	}
	if o.AccessToken != nil {
		acc.AccessToken = *o.AccessToken
	}
	acc.RefreshToken = o.RefreshToken
	acc.ExpiresAt = pgutil.TimestamptzToTime(o.ExpiresAt)
	return acc
}

func (r *OAuthRepo) Create(ctx context.Context, acc *domain.OAuthAccount) error {
	EnsureID(&acc.ID)
	var accessToken *string
	if acc.AccessToken != "" {
		accessToken = &acc.AccessToken
	}
	err := r.Q(ctx).CreateOAuthAccount(ctx, sqlc.CreateOAuthAccountParams{
		ID:             acc.ID,
		UserID:         acc.UserID,
		Provider:       acc.Provider,
		ProviderUserID: acc.ProviderUserID,
		AccessToken:    accessToken,
		RefreshToken:   acc.RefreshToken,
		ExpiresAt:      pgutil.TimeToTimestamptz(acc.ExpiresAt),
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return httperr.ErrOAuthAccountAlreadyLinked
		}
		return fmt.Errorf("OAuthRepo - Create: %w", err)
	}
	return nil
}

func (r *OAuthRepo) Upsert(ctx context.Context, acc *domain.OAuthAccount) error {
	EnsureID(&acc.ID)
	var accessToken *string
	if acc.AccessToken != "" {
		accessToken = &acc.AccessToken
	}
	err := r.Q(ctx).UpsertOAuthAccount(ctx, sqlc.UpsertOAuthAccountParams{
		ID:             acc.ID,
		UserID:         acc.UserID,
		Provider:       acc.Provider,
		ProviderUserID: acc.ProviderUserID,
		AccessToken:    accessToken,
		RefreshToken:   acc.RefreshToken,
		ExpiresAt:      pgutil.TimeToTimestamptz(acc.ExpiresAt),
	})
	if err != nil {
		return fmt.Errorf("OAuthRepo - Upsert: %w", err)
	}
	return nil
}

func (r *OAuthRepo) GetByProvider(ctx context.Context, provider, providerUserID string) (*domain.OAuthAccount, error) {
	o, err := r.Q(ctx).GetOAuthAccount(ctx, sqlc.GetOAuthAccountParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrOAuthAccountNotFound
		}
		return nil, fmt.Errorf("OAuthRepo - GetByProvider: %w", err)
	}
	return toDomainOAuthAccount(o), nil
}

func (r *OAuthRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.OAuthAccount, error) {
	rows, err := r.Q(ctx).GetOAuthAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("OAuthRepo - GetByUserID: %w", err)
	}
	out := make([]*domain.OAuthAccount, len(rows))
	for i := range rows {
		out[i] = toDomainOAuthAccount(rows[i])
	}
	return out, nil
}
