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

func scanUser(row rowScanner) (*entity.User, error) {
	var user entity.User
	err := row.Scan(
		&user.ID,
		&user.TeamID,
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
		u.Role = entity.RoleUser
	}
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	u.IsVerified = false

	query := squirrel.Insert("users").
		Columns("id", "username", "email", "password_hash", "role", "is_verified", "created_at").
		Values(u.ID, u.Username, u.Email, u.PasswordHash, u.Role, u.IsVerified, u.CreatedAt).
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

func (r *UserRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.User, error) {
	query := squirrel.Select(userColumns...).
		From("users").
		Where(squirrel.Eq{"id": ID}).
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

func (r *UserRepo) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error) {
	query := squirrel.Select(userColumns...).
		From("users").
		Where(squirrel.Eq{"team_id": teamID}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamID - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamID - Query: %w", err)
	}
	defer rows.Close()

	users, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[entity.User])
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamID - CollectRows: %w", err)
	}

	return users, nil
}

func (r *UserRepo) GetAll(ctx context.Context) ([]*entity.User, error) {
	query := squirrel.Select(userColumns...).
		From("users").
		OrderBy("created_at ASC").
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetAll - BuildQuery: %w", err)
	}

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetAll - Query: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("UserRepo - GetAll - Scan: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("UserRepo - GetAll - Rows: %w", err)
	}
	return users, nil
}

func (r *UserRepo) UpdateTeamID(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID) error {
	query := squirrel.Update("users").
		Where(squirrel.Eq{"id": userID}).
		PlaceholderFormat(squirrel.Dollar)

	if teamID != nil {
		query = query.Set("team_id", *teamID)
	} else {
		query = query.Set("team_id", nil)
	}

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateTeamID - BuildQuery: %w", err)
	}

	if _, err := r.pool.Exec(ctx, sqlQuery, args...); err != nil {
		return fmt.Errorf("UserRepo - UpdateTeamID - Exec: %w", err)
	}

	return nil
}

func (r *UserRepo) SetVerified(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	query := squirrel.Update("users").
		Set("is_verified", true).
		Set("verified_at", now).
		Where(squirrel.Eq{"id": userID}).
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

func (r *UserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := squirrel.Update("users").
		Set("password_hash", passwordHash).
		Where(squirrel.Eq{"id": userID}).
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
