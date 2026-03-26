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
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type OAuthRepo struct {
	BaseRepo

	crypto crypto.Service
}

var _ repo.OAuthAccountRepository = (*OAuthRepo)(nil)

func NewOAuthRepo(pool *pgxpool.Pool, cryptoService crypto.Service) *OAuthRepo {
	return &OAuthRepo{
		BaseRepo: BaseRepo{pool: pool},
		crypto:   cryptoService,
	}
}

func (r *OAuthRepo) toDomainOAuthAccount(o sqlc.OAuthAccount) (*domain.OAuthAccount, error) {
	acc := &domain.OAuthAccount{
		ID:             o.ID,
		UserID:         o.UserID,
		Provider:       o.Provider,
		ProviderUserID: o.ProviderUserID,
		CreatedAt:      pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(o.CreatedAt)),
		ExpiresAt:      pgutil.TimestamptzToTime(o.ExpiresAt),
	}

	if o.AccessToken != nil && *o.AccessToken != "" {
		if r.crypto != nil {
			decrypted, err := r.crypto.Decrypt(*o.AccessToken)
			if err != nil {
				return nil, fmt.Errorf("toDomainOAuthAccount - decrypt access token: %w", err)
			}

			acc.AccessToken = decrypted
		} else {
			acc.AccessToken = *o.AccessToken
		}
	}

	if o.RefreshToken != nil && *o.RefreshToken != "" {
		if r.crypto != nil {
			decrypted, err := r.crypto.Decrypt(*o.RefreshToken)
			if err != nil {
				return nil, fmt.Errorf("toDomainOAuthAccount - decrypt refresh token: %w", err)
			}

			acc.RefreshToken = &decrypted
		} else {
			acc.RefreshToken = o.RefreshToken
		}
	}

	return acc, nil
}

func (r *OAuthRepo) Create(ctx context.Context, acc *domain.OAuthAccount) error {
	EnsureID(&acc.ID)

	var (
		accessToken  *string
		refreshToken *string
	)

	if acc.AccessToken != "" {
		if r.crypto != nil {
			encrypted, err := r.crypto.Encrypt(acc.AccessToken)
			if err != nil {
				return fmt.Errorf("OAuthRepo - Create - encrypt access token: %w", err)
			}

			accessToken = &encrypted
		} else {
			accessToken = &acc.AccessToken
		}
	}

	if acc.RefreshToken != nil && *acc.RefreshToken != "" {
		if r.crypto != nil {
			encrypted, err := r.crypto.Encrypt(*acc.RefreshToken)
			if err != nil {
				return fmt.Errorf("OAuthRepo - Create - encrypt refresh token: %w", err)
			}

			refreshToken = &encrypted
		} else {
			refreshToken = acc.RefreshToken
		}
	}

	err := r.Q(ctx).CreateOAuthAccount(ctx, sqlc.CreateOAuthAccountParams{
		ID:             acc.ID,
		UserID:         acc.UserID,
		Provider:       acc.Provider,
		ProviderUserID: acc.ProviderUserID,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
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

	var (
		accessToken  *string
		refreshToken *string
	)

	if acc.AccessToken != "" {
		if r.crypto != nil {
			encrypted, err := r.crypto.Encrypt(acc.AccessToken)
			if err != nil {
				return fmt.Errorf("OAuthRepo - Upsert - encrypt access token: %w", err)
			}

			accessToken = &encrypted
		} else {
			accessToken = &acc.AccessToken
		}
	}

	if acc.RefreshToken != nil && *acc.RefreshToken != "" {
		if r.crypto != nil {
			encrypted, err := r.crypto.Encrypt(*acc.RefreshToken)
			if err != nil {
				return fmt.Errorf("OAuthRepo - Upsert - encrypt refresh token: %w", err)
			}

			refreshToken = &encrypted
		} else {
			refreshToken = acc.RefreshToken
		}
	}

	err := r.Q(ctx).UpsertOAuthAccount(ctx, sqlc.UpsertOAuthAccountParams{
		ID:             acc.ID,
		UserID:         acc.UserID,
		Provider:       acc.Provider,
		ProviderUserID: acc.ProviderUserID,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
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

	acc, err := r.toDomainOAuthAccount(o)
	if err != nil {
		return nil, fmt.Errorf("OAuthRepo - GetByProvider: %w", err)
	}

	return acc, nil
}

func (r *OAuthRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.OAuthAccount, error) {
	rows, err := r.Q(ctx).GetOAuthAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("OAuthRepo - GetByUserID: %w", err)
	}

	out := make([]*domain.OAuthAccount, 0, len(rows))
	for i := range rows {
		acc, err := r.toDomainOAuthAccount(rows[i])
		if err != nil {
			return nil, fmt.Errorf("OAuthRepo - GetByUserID: %w", err)
		}

		out = append(out, acc)
	}

	return out, nil
}
