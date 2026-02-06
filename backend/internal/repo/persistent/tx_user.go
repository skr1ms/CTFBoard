package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type TxUserRepo struct {
	base *TxBase
}

func (r *TxUserRepo) CreateUserTx(ctx context.Context, tx pgx.Tx, user *entity.User) error {
	user.CreatedAt = time.Now()
	isVerified := false
	id, err := r.base.q.WithTx(tx).CreateUserReturningID(ctx, sqlc.CreateUserReturningIDParams{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         &user.Role,
		IsVerified:   &isVerified,
		CreatedAt:    &user.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("TxUserRepo - CreateUserTx: %w", err)
	}
	user.ID = id
	return nil
}

func (r *TxUserRepo) UpdateUserTeamIDTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, teamID *uuid.UUID) error {
	_, err := r.base.q.WithTx(tx).UpdateUserTeamID(ctx, sqlc.UpdateUserTeamIDParams{
		ID:     userID,
		TeamID: teamID,
	})
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrUserNotFound
		}
		return fmt.Errorf("TxUserRepo - UpdateUserTeamIDTx: %w", err)
	}
	return nil
}

func (r *TxUserRepo) LockUserTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	query := squirrel.Select("id").
		From("users").
		Where(squirrel.Eq{"id": userID}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(squirrel.Dollar)
	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("TxUserRepo - LockUserTx - BuildQuery: %w", err)
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, sqlQuery, args...).Scan(&id)
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrUserNotFound
		}
		return fmt.Errorf("TxUserRepo - LockUserTx - Scan: %w", err)
	}
	return nil
}
