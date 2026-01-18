package persistent

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
)

type VerificationTokenRepo struct {
	db *sql.DB
}

func NewVerificationTokenRepo(db *sql.DB) *VerificationTokenRepo {
	return &VerificationTokenRepo{db: db}
}

func (r *VerificationTokenRepo) Create(ctx context.Context, token *entity.VerificationToken) error {
	if token.Id == "" {
		token.Id = uuid.New().String()
	}

	query, args, err := sq.Insert("verification_tokens").
		Columns("id", "user_id", "token", "type", "expires_at").
		Values(token.Id, token.UserId, token.Token, token.Type, token.ExpiresAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - Create - ToSql: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - Create - ExecContext: %w", err)
	}

	return nil
}

func (r *VerificationTokenRepo) GetByToken(ctx context.Context, token string) (*entity.VerificationToken, error) {
	query, args, err := sq.Select("id", "user_id", "token", "type", "expires_at", "used_at", "created_at").
		From("verification_tokens").
		Where(sq.Eq{"token": token}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("VerificationTokenRepo - GetByToken - ToSql: %w", err)
	}

	var t entity.VerificationToken
	var tokenType string
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&t.Id, &t.UserId, &t.Token, &tokenType, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entityError.ErrTokenNotFound
		}
		return nil, fmt.Errorf("VerificationTokenRepo - GetByToken - QueryRowContext: %w", err)
	}

	t.Type = entity.TokenType(tokenType)
	return &t, nil
}

func (r *VerificationTokenRepo) MarkUsed(ctx context.Context, id string) error {
	query, args, err := sq.Update("verification_tokens").
		Set("used_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - MarkUsed - ToSql: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - MarkUsed - ExecContext: %w", err)
	}

	return nil
}

func (r *VerificationTokenRepo) DeleteExpired(ctx context.Context) error {
	query, args, err := sq.Delete("verification_tokens").
		Where(sq.Lt{"expires_at": time.Now()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteExpired - ToSql: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteExpired - ExecContext: %w", err)
	}

	return nil
}

func (r *VerificationTokenRepo) DeleteByUserAndType(ctx context.Context, userId string, tokenType entity.TokenType) error {
	query, args, err := sq.Delete("verification_tokens").
		Where(sq.Eq{"user_id": userId, "type": tokenType}).
		ToSql()
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteByUserAndType - ToSql: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("VerificationTokenRepo - DeleteByUserAndType - ExecContext: %w", err)
	}

	return nil
}
