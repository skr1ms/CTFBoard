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

type UserRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db, q: sqlc.New(db)}
}

func toEntityUser(u sqlc.User) *entity.User {
	return &entity.User{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         ptrStrToStr(u.Role),
		IsVerified:   boolPtrToBool(u.IsVerified),
		TeamID:       u.TeamID,
		VerifiedAt:   u.VerifiedAt,
		CreatedAt:    ptrTimeToTime(u.CreatedAt),
	}
}

func (r *UserRepo) Create(ctx context.Context, u *entity.User) error {
	if u.Role == "" {
		u.Role = entity.RoleUser
	}
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	u.IsVerified = false
	isVerified := false
	err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         &u.Role,
		IsVerified:   &isVerified,
		CreatedAt:    &u.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - Create: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByID: %w", err)
	}
	return toEntityUser(u), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByEmail: %w", err)
	}
	return toEntityUser(u), nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	u, err := r.q.GetUserByUsername(ctx, username)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByUsername: %w", err)
	}
	return toEntityUser(u), nil
}

func (r *UserRepo) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error) {
	rows, err := r.q.ListUsersByTeamID(ctx, &teamID)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamID: %w", err)
	}
	out := make([]*entity.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toEntityUser(u))
	}
	return out, nil
}

func (r *UserRepo) GetAll(ctx context.Context) ([]*entity.User, error) {
	rows, err := r.q.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetAll: %w", err)
	}
	out := make([]*entity.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toEntityUser(u))
	}
	return out, nil
}

func (r *UserRepo) UpdateTeamID(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID) error {
	_, err := r.q.UpdateUserTeamID(ctx, sqlc.UpdateUserTeamIDParams{
		ID:     userID,
		TeamID: teamID,
	})
	if err != nil {
		if isNoRows(err) {
			return entityError.ErrUserNotFound
		}
		return fmt.Errorf("UserRepo - UpdateTeamID: %w", err)
	}
	return nil
}

func (r *UserRepo) SetVerified(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	ok := true
	if err := r.q.UpdateUserVerified(ctx, sqlc.UpdateUserVerifiedParams{
		ID:         userID,
		IsVerified: &ok,
		VerifiedAt: &now,
	}); err != nil {
		return fmt.Errorf("UserRepo - SetVerified: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	if err := r.q.UpdatePassword(ctx, sqlc.UpdatePasswordParams{
		ID:           userID,
		PasswordHash: passwordHash,
	}); err != nil {
		return fmt.Errorf("UserRepo - UpdatePassword: %w", err)
	}
	return nil
}
