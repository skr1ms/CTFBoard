package persistent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *entity.User) error {
	if u.Role == "" {
		u.Role = "user"
	}
	u.Id = uuid.New().String()
	u.CreatedAt = time.Now()

	query := squirrel.Insert("users").
		Columns("id", "username", "email", "password_hash", "role", "created_at").
		Values(u.Id, u.Username, u.Email, u.PasswordHash, u.Role, u.CreatedAt)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - Create - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("UserRepo - Create - ExecQuery: %w", err)
	}

	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*entity.User, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByID - ParseID: %w", err)
	}

	query := squirrel.Select("id", "team_id", "username", "email", "password_hash", "role", "created_at").
		From("users").
		Where(squirrel.Eq{"id": uuidID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByID - BuildQuery: %w", err)
	}

	var user entity.User
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&user.Id,
		&user.TeamId,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByID - Scan: %w", err)
	}

	return &user, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := squirrel.Select("id", "team_id", "username", "email", "password_hash", "role", "created_at").
		From("users").
		Where(squirrel.Eq{"email": email})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByEmail - BuildQuery: %w", err)
	}

	var user entity.User
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&user.Id,
		&user.TeamId,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByEmail - Scan: %w", err)
	}

	return &user, nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	query := squirrel.Select("id", "team_id", "username", "email", "password_hash", "role", "created_at").
		From("users").
		Where(squirrel.Eq{"username": username})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByUsername - BuildQuery: %w", err)
	}

	var user entity.User
	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&user.Id,
		&user.TeamId,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entityError.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByUsername - Scan: %w", err)
	}

	return &user, nil
}

func (r *UserRepo) GetByTeamId(ctx context.Context, teamId string) ([]*entity.User, error) {
	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamId - ParseID: %w", err)
	}

	query := squirrel.Select("id", "team_id", "username", "email", "password_hash", "role", "created_at").
		From("users").
		Where(squirrel.Eq{"team_id": teamUUID})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamId - BuildQuery: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamId - Query: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var users []*entity.User
	for rows.Next() {
		var user entity.User
		if err := rows.Scan(
			&user.Id,
			&user.TeamId,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("UserRepo - GetByTeamId - Scan: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamId - Rows: %w", err)
	}

	return users, nil
}

func (r *UserRepo) UpdateTeamId(ctx context.Context, userId string, teamId *string) error {
	userUUID, err := uuid.Parse(userId)
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateTeamId - Parse UserID: %w", err)
	}

	query := squirrel.Update("users").
		Where(squirrel.Eq{"id": userUUID})

	if teamId != nil {
		teamUUID, err := uuid.Parse(*teamId)
		if err != nil {
			return fmt.Errorf("UserRepo - UpdateTeamId - Parse TeamID: %w", err)
		}
		query = query.Set("team_id", teamUUID)
	} else {
		query = query.Set("team_id", nil)
	}

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateTeamId - BuildQuery: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateTeamId - ExecQuery: %w", err)
	}

	return nil
}
