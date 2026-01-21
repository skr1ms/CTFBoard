package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
)

type VerificationTokenRepo struct {
	pool *pgxpool.Pool
}

func NewVerificationTokenRepo(pool *pgxpool.Pool) *VerificationTokenRepo {
	return &VerificationTokenRepo{pool: pool}
}

func (r *VerificationTokenRepo) Create(ctx context.Context, token *entity.VerificationToken) error {
	if token.Id == uuid.Nil {
		token.Id = uuid.New()
	}

	query, args, err := sq.Insert("verification_tokens").
		Columns("id", "user_id", "token", "type", "expires_at").
		Values(token.Id, token.UserId, token.Token, token.Type, token.ExpiresAt).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - Create - ToSql: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - Create - Exec: %w", err)
	}

	return nil
}

func (r *VerificationTokenRepo) GetByToken(ctx context.Context, token string) (*entity.VerificationToken, error) {
	query, args, err := sq.Select("id", "user_id", "token", "type", "expires_at", "used_at", "created_at").
		From("verification_tokens").
		Where(sq.Eq{"token": token}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("VerificationTokenRepo - GetByToken - ToSql: %w", err)
	}

	var t entity.VerificationToken
	var tokenType string
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&t.Id, &t.UserId, &t.Token, &tokenType, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrTokenNotFound
		}
		return nil, fmt.Errorf("VerificationTokenRepo - GetByToken - QueryRow: %w", err)
	}

	t.Type = entity.TokenType(tokenType)
	return &t, nil
}

func (r *VerificationTokenRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	query, args, err := sq.Update("verification_tokens").
		Set("used_at", time.Now()).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - MarkUsed - ToSql: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - MarkUsed - Exec: %w", err)
	}

	return nil
}

func (r *VerificationTokenRepo) DeleteExpired(ctx context.Context) error {
	query, args, err := sq.Delete("verification_tokens").
		Where(sq.Lt{"expires_at": time.Now()}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteExpired - ToSql: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteExpired - Exec: %w", err)
	}

	return nil
}

func (r *VerificationTokenRepo) DeleteByUserAndType(ctx context.Context, userId uuid.UUID, tokenType entity.TokenType) error {
	query, args, err := sq.Delete("verification_tokens").
		Where(sq.Eq{"user_id": userId, "type": tokenType}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteByUserAndType - ToSql: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteByUserAndType - Exec: %w", err)
	}

	return nil
}
