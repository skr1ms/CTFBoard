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

type VerificationTokenRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewVerificationTokenRepo(db *pgxpool.Pool) *VerificationTokenRepo {
	return &VerificationTokenRepo{db: db, q: sqlc.New(db)}
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
	err := r.q.CreateVerificationToken(ctx, sqlc.CreateVerificationTokenParams{
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
	t, err := r.q.GetVerificationTokenByToken(ctx, token)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrTokenNotFound
		}
		return nil, fmt.Errorf("VerificationTokenRepo - GetByToken: %w", err)
	}
	return toEntityVerificationToken(t), nil
}

func (r *VerificationTokenRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	if err := r.q.MarkVerificationTokenUsed(ctx, id); err != nil {
		return fmt.Errorf("VerificationTokenRepo - MarkUsed: %w", err)
	}
	return nil
}

func (r *VerificationTokenRepo) DeleteExpired(ctx context.Context) error {
	if err := r.q.DeleteExpiredVerificationTokens(ctx, time.Now()); err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteExpired: %w", err)
	}
	return nil
}

func (r *VerificationTokenRepo) DeleteByUserAndType(ctx context.Context, userID uuid.UUID, tokenType entity.TokenType) error {
	if err := r.q.DeleteVerificationTokensByUserAndType(ctx, sqlc.DeleteVerificationTokensByUserAndTypeParams{
		UserID: userID,
		Type:   string(tokenType),
	}); err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteByUserAndType: %w", err)
	}
	return nil
}
