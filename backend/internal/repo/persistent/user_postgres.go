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

type UserRepo struct {
	pool *pgxpool.Pool
}

var _ repo.UserRepository = (*UserRepo)(nil)

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) q(ctx context.Context) *sqlc.Queries {
	return sqlc.New(ExtractDB(ctx, r.pool))
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
		IsBanned:     u.IsBanned,
		BannedAt:     u.BannedAt,
		BannedReason: u.BannedReason,
	}
}

func (r *UserRepo) Create(ctx context.Context, u *entity.User) error {
	if u.Role == "" {
		u.Role = entity.RoleUser
	}
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	u.CreatedAt = time.Now()
	u.IsVerified = false
	isVerified := false
	err := r.q(ctx).CreateUser(ctx, sqlc.CreateUserParams{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         &u.Role,
		IsVerified:   &isVerified,
		CreatedAt:    &u.CreatedAt,
	})
	if err != nil {
		if isPgUniqueViolation(err) {
			return httperr.ErrUserAlreadyExists
		}
		return fmt.Errorf("UserRepo - Create: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, ID uuid.UUID) (*entity.User, error) {
	u, err := r.q(ctx).GetUserByID(ctx, ID)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByID: %w", err)
	}
	return toEntityUser(u), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	u, err := r.q(ctx).GetUserByEmail(ctx, email)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByEmail: %w", err)
	}
	return toEntityUser(u), nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	u, err := r.q(ctx).GetUserByUsername(ctx, username)
	if err != nil {
		if isNoRows(err) {
			return nil, httperr.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepo - GetByUsername: %w", err)
	}
	return toEntityUser(u), nil
}

func (r *UserRepo) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*entity.User, error) {
	rows, err := r.q(ctx).ListUsersByTeamID(ctx, &teamID)
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
	rows, err := r.q(ctx).GetAllUsers(ctx)
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
	_, err := r.q(ctx).UpdateUserTeamID(ctx, sqlc.UpdateUserTeamIDParams{
		ID:     userID,
		TeamID: teamID,
	})
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrUserNotFound
		}
		return fmt.Errorf("UserRepo - UpdateTeamID: %w", err)
	}
	return nil
}

func (r *UserRepo) SetVerified(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	ok := true
	if _, err := r.q(ctx).UpdateUserVerified(ctx, sqlc.UpdateUserVerifiedParams{
		ID:         userID,
		IsVerified: &ok,
		VerifiedAt: &now,
	}); err != nil {
		if isNoRows(err) {
			return httperr.ErrUserNotFound
		}
		return fmt.Errorf("UserRepo - SetVerified: %w", err)
	}
	return nil
}

func (r *UserRepo) SetUnverified(ctx context.Context, userID uuid.UUID) error {
	ok := false
	if _, err := r.q(ctx).UpdateUserVerified(ctx, sqlc.UpdateUserVerifiedParams{
		ID:         userID,
		IsVerified: &ok,
		VerifiedAt: nil,
	}); err != nil {
		if isNoRows(err) {
			return httperr.ErrUserNotFound
		}
		return fmt.Errorf("UserRepo - SetUnverified: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	if err := r.q(ctx).UpdatePassword(ctx, sqlc.UpdatePasswordParams{
		ID:           userID,
		PasswordHash: passwordHash,
	}); err != nil {
		if isNoRows(err) {
			return httperr.ErrUserNotFound
		}
		return fmt.Errorf("UserRepo - UpdatePassword: %w", err)
	}
	return nil
}

func (r *UserRepo) Search(ctx context.Context, search *string, limit, offset int) ([]*entity.User, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - Search - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - Search - offset: %w", err)
	}
	rows, err := r.q(ctx).SearchUsers(ctx, sqlc.SearchUsersParams{
		Limit:  limit32,
		Offset: offset32,
		Search: search,
	})
	if err != nil {
		return nil, fmt.Errorf("UserRepo - Search: %w", err)
	}
	out := make([]*entity.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toEntityUser(u))
	}
	return out, nil
}

func (r *UserRepo) CountSearch(ctx context.Context, search *string) (int64, error) {
	count, err := r.q(ctx).CountSearchUsers(ctx, search)
	if err != nil {
		return 0, fmt.Errorf("UserRepo - CountSearch: %w", err)
	}
	return count, nil
}

func (r *UserRepo) UpdateAdmin(ctx context.Context, userID uuid.UUID, username, email, role, passwordHash *string, isVerified *bool) error {
	if err := r.q(ctx).UpdateUserAdmin(ctx, sqlc.UpdateUserAdminParams{
		ID:           userID,
		Username:     username,
		Email:        email,
		Role:         role,
		IsVerified:   isVerified,
		PasswordHash: passwordHash,
	}); err != nil {
		if isNoRows(err) {
			return httperr.ErrUserNotFound
		}
		return fmt.Errorf("UserRepo - UpdateAdmin: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdateProfile(ctx context.Context, userID uuid.UUID, username, email, passwordHash *string) error {
	if err := r.q(ctx).UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		ID:           userID,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
	}); err != nil {
		if isNoRows(err) {
			return httperr.ErrUserNotFound
		}
		if isPgUniqueViolation(err) {
			return httperr.ErrUserAlreadyExists
		}
		return fmt.Errorf("UserRepo - UpdateProfile: %w", err)
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, userID uuid.UUID) error {
	if err := r.q(ctx).DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("UserRepo - Delete: %w", err)
	}
	return nil
}

func (r *UserRepo) SearchByIP(ctx context.Context, ip string, limit, offset int) ([]*entity.User, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchByIP - limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchByIP - offset: %w", err)
	}
	rows, err := r.q(ctx).SearchUsersByIP(ctx, sqlc.SearchUsersByIPParams{
		Limit:  limit32,
		Offset: offset32,
		IP:     ip,
	})
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchByIP: %w", err)
	}
	out := make([]*entity.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toEntityUser(u))
	}
	return out, nil
}

func (r *UserRepo) CountSearchByIP(ctx context.Context, ip string) (int64, error) {
	count, err := r.q(ctx).CountSearchUsersByIP(ctx, ip)
	if err != nil {
		return 0, fmt.Errorf("UserRepo - CountSearchByIP: %w", err)
	}
	return count, nil
}

func (r *UserRepo) Lock(ctx context.Context, userID uuid.UUID) error {
	_, err := r.q(ctx).LockUser(ctx, userID)
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrUserNotFound
		}
		return fmt.Errorf("UserRepo - Lock: %w", err)
	}
	return nil
}

func (r *UserRepo) Ban(ctx context.Context, userID uuid.UUID, reason string) error {
	bannedAt := time.Now()
	_, err := r.q(ctx).BanUser(ctx, sqlc.BanUserParams{
		ID:           userID,
		BannedAt:     &bannedAt,
		BannedReason: &reason,
	})
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrUserNotFound
		}
		return fmt.Errorf("UserRepo - Ban: %w", err)
	}
	return nil
}

func (r *UserRepo) Unban(ctx context.Context, userID uuid.UUID) error {
	_, err := r.q(ctx).UnbanUser(ctx, userID)
	if err != nil {
		if isNoRows(err) {
			return httperr.ErrUserNotFound
		}
		return fmt.Errorf("UserRepo - Unban: %w", err)
	}
	return nil
}
