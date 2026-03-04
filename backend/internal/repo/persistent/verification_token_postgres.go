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

type VerificationTokenRepo struct {
	pool *pgxpool.Pool
}

var _ repo.VerificationTokenRepository = (*VerificationTokenRepo)(nil)

func NewVerificationTokenRepo(pool *pgxpool.Pool) *VerificationTokenRepo {
	return &VerificationTokenRepo{pool: pool}
}

func (r *VerificationTokenRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
}

func toEntityVerificationToken(t sqlc.VerificationToken) *entity.VerificationToken {
	return &entity.VerificationToken{
		ID:        t.ID,
		UserID:    t.UserID,
		Token:     t.Token,
		Type:      entity.TokenType(t.Type),
		ExpiresAt: t.ExpiresAt,
		UsedAt:    t.UsedAt,
		CreatedAt: ptrTimeToTime(t.CreatedAt),
	}
}

func (r *VerificationTokenRepo) Create(ctx context.Context, token *entity.VerificationToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	err := r.q(ctx).CreateVerificationToken(ctx, sqlc.CreateVerificationTokenParams{
		ID:        token.ID,
		UserID:    token.UserID,
		Token:     token.Token,
		Type:      string(token.Type),
		ExpiresAt: token.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - Create: %w", err)
	}
	return nil
}

func (r *VerificationTokenRepo) GetByToken(ctx context.Context, token string) (*entity.VerificationToken, error) {
	t, err := r.q(ctx).GetVerificationTokenByToken(ctx, token)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrTokenNotFound
		}
		return nil, fmt.Errorf("VerificationTokenRepo - GetByToken: %w", err)
	}
	return toEntityVerificationToken(t), nil
}

func (r *VerificationTokenRepo) MarkUsed(ctx context.Context, ID uuid.UUID) error {
	if err := r.q(ctx).MarkVerificationTokenUsed(ctx, ID); err != nil {
		return fmt.Errorf("VerificationTokenRepo - MarkUsed: %w", err)
	}
	return nil
}

func (r *VerificationTokenRepo) DeleteExpired(ctx context.Context) error {
	if err := r.q(ctx).DeleteExpiredVerificationTokens(ctx, time.Now()); err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteExpired: %w", err)
	}
	return nil
}

func (r *VerificationTokenRepo) DeleteByUserAndType(ctx context.Context, userID uuid.UUID, tokenType entity.TokenType) error {
	if err := r.q(ctx).DeleteVerificationTokensByUserAndType(ctx, sqlc.DeleteVerificationTokensByUserAndTypeParams{
		UserID: userID,
		Type:   string(tokenType),
	}); err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteByUserAndType: %w", err)
	}
	return nil
}
