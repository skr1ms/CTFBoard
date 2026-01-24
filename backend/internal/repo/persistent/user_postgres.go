package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

var userColumns = []string{"id", "team_id", "username", "email", "password_hash", "role", "is_verified", "verified_at", "created_at"}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*entity.User, error) {
	var user entity.User
	err := row.Scan(
		&user.Id,
		&user.TeamId,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsVerified,
		&user.VerifiedAt,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) Create(ctx context.Context, u *entity.User) error {
	if u.Role == "" {
		u.Role = "user"
	}
	u.Id = uuid.New()
	u.CreatedAt = time.Now()
	u.IsVerified = false

	query := squirrel.Insert("users").
		Columns("id", "username", "email", "password_hash", "role", "is_verified", "created_at").
		Values(u.Id, u.Username, u.Email, u.PasswordHash, u.Role, u.IsVerified, u.CreatedAt).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("UserRepo - Create - ExecQuery: %w", err)
	}

	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := squirrel.Select(userColumns...).
		From("users").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByID - BuildQuery: %w", err)
	}

	user, err := scanUser(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByID - Scan: %w", err)
	}

	return user, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := squirrel.Select(userColumns...).
		From("users").
		Where(squirrel.Eq{"email": email}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByEmail - BuildQuery: %w", err)
	}

	user, err := scanUser(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByEmail - Scan: %w", err)
	}

	return user, nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	query := squirrel.Select(userColumns...).
		From("users").
		Where(squirrel.Eq{"username": username}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByUsername - BuildQuery: %w", err)
	}

	user, err := scanUser(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByUsername - Scan: %w", err)
	}

	return user, nil
}

func (r *UserRepo) GetByTeamId(ctx context.Context, teamId uuid.UUID) ([]*entity.User, error) {
	query := squirrel.Select(userColumns...).
		From("users").
		Where(squirrel.Eq{"team_id": teamId}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamId - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamId - Query: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("UserRepo - GetByTeamId - Scan: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamId - Rows: %w", err)
	}

	return users, nil
}

func (r *UserRepo) UpdateTeamId(ctx context.Context, userId uuid.UUID, teamId *uuid.UUID) error {
	query := squirrel.Update("users").
		Where(squirrel.Eq{"id": userId}).
		PlaceholderFormat(squirrel.Dollar)

	if teamId != nil {
		query = query.Set("team_id", *teamId)
	} else {
		query = query.Set("team_id", nil)
	}

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateTeamId - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateTeamId - ExecQuery: %w", err)
	}

	return nil
}

func (r *UserRepo) SetVerified(ctx context.Context, userId uuid.UUID) error {
	now := time.Now()
	query := squirrel.Update("users").
		Set("is_verified", true).
		Set("verified_at", now).
		Where(squirrel.Eq{"id": userId}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - SetVerified - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("UserRepo - SetVerified - ExecQuery: %w", err)
	}

	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userId uuid.UUID, passwordHash string) error {
	query := squirrel.Update("users").
		Set("password_hash", passwordHash).
		Where(squirrel.Eq{"id": userId}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - UpdatePassword - BuildQuery: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("UserRepo - UpdatePassword - ExecQuery: %w", err)
	}

	return nil
}
