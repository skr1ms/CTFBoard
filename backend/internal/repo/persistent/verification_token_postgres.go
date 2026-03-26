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

type VerificationTokenRepo struct {
	BaseRepo
}

var _ repo.VerificationTokenRepository = (*VerificationTokenRepo)(nil)

func NewVerificationTokenRepo(pool *pgxpool.Pool) *VerificationTokenRepo {
	return &VerificationTokenRepo{BaseRepo: BaseRepo{pool: pool}}
}

func toDomainVerificationToken(t sqlc.VerificationToken) *domain.VerificationToken {
	return &domain.VerificationToken{
		ID:        t.ID,
		UserID:    t.UserID,
		Token:     t.Token,
		Type:      domain.TokenType(t.Type),
		ExpiresAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(t.ExpiresAt)),
		UsedAt:    pgutil.TimestamptzToTime(t.UsedAt),
		CreatedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(t.CreatedAt)),
	}
}

func (r *VerificationTokenRepo) Create(ctx context.Context, token *domain.VerificationToken) error {
	EnsureID(&token.ID)

	err := r.Q(ctx).CreateVerificationToken(ctx, sqlc.CreateVerificationTokenParams{
		ID:        token.ID,
		UserID:    token.UserID,
		Token:     token.Token,
		Type:      string(token.Type),
		ExpiresAt: pgutil.TimeToTimestamptz(&token.ExpiresAt),
	})
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - Create: %w", err)
	}

	return nil
}

func (r *VerificationTokenRepo) GetByToken(ctx context.Context, token string) (*domain.VerificationToken, error) {
	t, err := r.Q(ctx).GetVerificationTokenByToken(ctx, token)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrTokenNotFound
		}

		return nil, fmt.Errorf("VerificationTokenRepo - GetByToken: %w", err)
	}

	return toDomainVerificationToken(t), nil
}

func (r *VerificationTokenRepo) MarkUsed(ctx context.Context, ID uuid.UUID) error {
	now := time.Now()

	err := r.Q(ctx).MarkVerificationTokenUsed(ctx, sqlc.MarkVerificationTokenUsedParams{
		ID:     ID,
		UsedAt: pgutil.TimeToTimestamptz(&now),
	})
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - MarkUsed: %w", err)
	}

	return nil
}

func (r *VerificationTokenRepo) DeleteExpired(ctx context.Context) error {
	expiresAt := time.Now()

	err := r.Q(ctx).DeleteExpiredVerificationTokens(ctx, pgutil.TimeToTimestamptz(&expiresAt))
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteExpired: %w", err)
	}

	return nil
}

func (r *VerificationTokenRepo) DeleteByUserAndType(ctx context.Context, userID uuid.UUID, tokenType domain.TokenType) error {
	err := r.Q(ctx).DeleteVerificationTokensByUserAndType(ctx, sqlc.DeleteVerificationTokensByUserAndTypeParams{
		UserID: userID,
		Type:   string(tokenType),
	})
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteByUserAndType: %w", err)
	}

	return nil
}
