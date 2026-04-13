package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type UserRepo struct {
	BaseRepo
}

var _ repo.UserRepository = (*UserRepo)(nil)

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{BaseRepo: BaseRepo{pool: pool}}
}

type userRow struct {
	ID              uuid.UUID
	TeamID          *uuid.UUID
	Username        string
	Email           string
	PasswordHash    string
	Role            string
	IsVerified      bool
	VerifiedAt      pgtype.Timestamptz
	IsBanned        bool
	BannedAt        pgtype.Timestamptz
	BannedReason    *string
	WasInBannedTeam bool
	AvatarUrl       *string
	CreatedAt       pgtype.Timestamptz
}

func userRowFrom(u sqlc.User) *userRow {
	return &userRow{
		ID:              u.ID,
		TeamID:          u.TeamID,
		Username:        u.Username,
		Email:           u.Email,
		PasswordHash:    u.PasswordHash,
		Role:            u.Role,
		IsVerified:      u.IsVerified,
		VerifiedAt:      u.VerifiedAt,
		IsBanned:        u.IsBanned,
		BannedAt:        u.BannedAt,
		BannedReason:    u.BannedReason,
		WasInBannedTeam: u.WasInBannedTeam,
		AvatarUrl:       u.AvatarUrl,
		CreatedAt:       u.CreatedAt,
	}
}

func toDomainUser(r *userRow) *domain.User {
	return &domain.User{
		ID:              r.ID,
		Username:        r.Username,
		Email:           r.Email,
		PasswordHash:    r.PasswordHash,
		Role:            domain.Role(r.Role),
		IsVerified:      r.IsVerified,
		TeamID:          r.TeamID,
		VerifiedAt:      pgutil.TimestamptzToTime(r.VerifiedAt),
		CreatedAt:       pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(r.CreatedAt)),
		IsBanned:        r.IsBanned,
		BannedAt:        pgutil.TimestamptzToTime(r.BannedAt),
		BannedReason:    r.BannedReason,
		WasInBannedTeam: r.WasInBannedTeam,
		AvatarURL:       r.AvatarUrl,
	}
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	if u.Role == "" {
		u.Role = domain.RoleUser
	}

	EnsureID(&u.ID)
	u.CreatedAt = time.Now()

	err := r.Q(ctx).CreateUser(ctx, sqlc.CreateUserParams{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
		IsVerified:   u.IsVerified,
		CreatedAt:    pgutil.TimeToTimestamptz(&u.CreatedAt),
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return apperr.ErrUserAlreadyExists
		}

		return fmt.Errorf("UserRepo - Create: %w", err)
	}

	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.User, error) {
	u, err := GetOrNotFound(func() (sqlc.User, error) { return r.Q(ctx).GetUserByID(ctx, ID) }, apperr.ErrUserNotFound, "UserRepo - GetByID")
	if err != nil {
		return nil, err
	}

	return toDomainUser(userRowFrom(u)), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := GetOrNotFound(func() (sqlc.User, error) { return r.Q(ctx).GetUserByEmail(ctx, email) }, apperr.ErrUserNotFound, "UserRepo - GetByEmail")
	if err != nil {
		return nil, err
	}

	return toDomainUser(userRowFrom(u)), nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	u, err := GetOrNotFound(func() (sqlc.User, error) { return r.Q(ctx).GetUserByUsername(ctx, username) }, apperr.ErrUserNotFound, "UserRepo - GetByUsername")
	if err != nil {
		return nil, err
	}

	return toDomainUser(userRowFrom(u)), nil
}

func (r *UserRepo) GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error) {
	rows, err := r.Q(ctx).ListUsersByTeamID(ctx, &teamID)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamID: %w", err)
	}

	out := make([]*domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(userRowFrom(u)))
	}

	return out, nil
}

func (r *UserRepo) GetByTeamIDs(ctx context.Context, teamIDs []uuid.UUID) (map[uuid.UUID][]*domain.User, error) {
	if len(teamIDs) == 0 {
		return map[uuid.UUID][]*domain.User{}, nil
	}

	rows, err := r.Q(ctx).ListUsersByTeamIDs(ctx, teamIDs)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetByTeamIDs: %w", err)
	}

	out := make(map[uuid.UUID][]*domain.User)

	for _, u := range rows {
		if u.TeamID != nil {
			out[*u.TeamID] = append(out[*u.TeamID], toDomainUser(userRowFrom(u)))
		}
	}

	return out, nil
}

func (r *UserRepo) UpdateTeamIDBatch(ctx context.Context, userIDs []uuid.UUID, teamID *uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	err := r.Q(ctx).UpdateUserTeamIDBatch(ctx, sqlc.UpdateUserTeamIDBatchParams{
		TeamID:  teamID,
		Column2: userIDs,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateTeamIDBatch: %w", err)
	}

	return nil
}

func (r *UserRepo) FilterIDsByTeamIDNull(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	ids, err := r.Q(ctx).ListUserIDsWithTeamIDNull(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - FilterIDsByTeamIDNull: %w", err)
	}

	return ids, nil
}

func (r *UserRepo) FilterIDsByTeamIDNullAndNotBanned(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	ids, err := r.Q(ctx).ListUserIDsWithTeamIDNullAndNotBanned(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - FilterIDsByTeamIDNullAndNotBanned: %w", err)
	}

	return ids, nil
}

func (r *UserRepo) GetAll(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.Q(ctx).GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - GetAll: %w", err)
	}

	out := make([]*domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(userRowFrom(u)))
	}

	return out, nil
}

func (r *UserRepo) UpdateTeamID(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).UpdateUserTeamID(ctx, sqlc.UpdateUserTeamIDParams{ID: userID, TeamID: teamID})
	}, apperr.ErrUserNotFound, "UserRepo - UpdateTeamID")

	return err
}

func (r *UserRepo) SetVerified(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).UpdateUserVerified(ctx, sqlc.UpdateUserVerifiedParams{ID: userID, IsVerified: true, VerifiedAt: pgutil.TimeToTimestamptz(&now)})
	}, apperr.ErrUserNotFound, "UserRepo - SetVerified")

	return err
}

func (r *UserRepo) SetUnverified(ctx context.Context, userID uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).UpdateUserVerified(ctx, sqlc.UpdateUserVerifiedParams{ID: userID, IsVerified: false, VerifiedAt: pgutil.TimeToTimestamptz(nil)})
	}, apperr.ErrUserNotFound, "UserRepo - SetUnverified")

	return err
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	rows, err := r.Q(ctx).UpdatePassword(ctx, sqlc.UpdatePasswordParams{
		ID:           userID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - UpdatePassword: %w", err)
	}

	if rows == 0 {
		return apperr.ErrUserNotFound
	}

	return nil
}

func (r *UserRepo) Search(ctx context.Context, search *string, limit, offset int) ([]*domain.User, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - Search: %w", err)
	}

	escapedSearch := EscapeSearchPtr(search)

	rows, err := r.Q(ctx).SearchUsers(ctx, sqlc.SearchUsersParams{
		Limit:  limit32,
		Offset: offset32,
		Search: escapedSearch,
	})
	if err != nil {
		return nil, fmt.Errorf("UserRepo - Search - scan: %w", err)
	}

	out := make([]*domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(userRowFrom(u)))
	}

	return out, nil
}

func (r *UserRepo) CountSearch(ctx context.Context, search *string) (int64, error) {
	escapedSearch := EscapeSearchPtr(search)

	count, err := r.Q(ctx).CountSearchUsers(ctx, escapedSearch)
	if err != nil {
		return 0, fmt.Errorf("UserRepo - CountSearch: %w", err)
	}

	return count, nil
}

func (r *UserRepo) UpdateAdmin(ctx context.Context, userID uuid.UUID, username, email, role, passwordHash *string, isVerified *bool) error {
	rows, err := r.Q(ctx).UpdateUserAdmin(ctx, sqlc.UpdateUserAdminParams{
		ID:           userID,
		Username:     username,
		Email:        email,
		Role:         role,
		IsVerified:   isVerified,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateAdmin: %w", err)
	}

	if rows == 0 {
		return apperr.ErrUserNotFound
	}

	return nil
}

func (r *UserRepo) UpdateProfile(ctx context.Context, userID uuid.UUID, username, email, passwordHash *string) error {
	rows, err := r.Q(ctx).UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		ID:           userID,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		if pgutil.IsPgUniqueViolation(err) {
			return apperr.ErrUserAlreadyExists
		}

		return fmt.Errorf("UserRepo - UpdateProfile: %w", err)
	}

	if rows == 0 {
		return apperr.ErrUserNotFound
	}

	return nil
}

func (r *UserRepo) Delete(ctx context.Context, userID uuid.UUID) error {
	err := r.Q(ctx).DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("UserRepo - Delete: %w", err)
	}

	return nil
}

func (r *UserRepo) SearchByIP(ctx context.Context, ip string, limit, offset int) ([]*domain.User, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchByIP: %w", err)
	}

	rows, err := r.Q(ctx).SearchUsersByIP(ctx, sqlc.SearchUsersByIPParams{
		Limit:  limit32,
		Offset: offset32,
		IP:     ip,
	})
	if err != nil {
		return nil, fmt.Errorf("UserRepo - SearchByIP - scan: %w", err)
	}

	out := make([]*domain.User, 0, len(rows))
	for _, u := range rows {
		out = append(out, toDomainUser(userRowFrom(u)))
	}

	return out, nil
}

func (r *UserRepo) CountSearchByIP(ctx context.Context, ip string) (int64, error) {
	count, err := r.Q(ctx).CountSearchUsersByIP(ctx, ip)
	if err != nil {
		return 0, fmt.Errorf("UserRepo - CountSearchByIP: %w", err)
	}

	return count, nil
}

func (r *UserRepo) Lock(ctx context.Context, userID uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) { return r.Q(ctx).LockUser(ctx, userID) }, apperr.ErrUserNotFound, "UserRepo - Lock")

	return err
}

func (r *UserRepo) Ban(ctx context.Context, userID uuid.UUID, reason string) error {
	bannedAt := time.Now()
	_, err := GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).BanUser(ctx, sqlc.BanUserParams{ID: userID, BannedAt: pgutil.TimeToTimestamptz(&bannedAt), BannedReason: &reason})
	}, apperr.ErrUserNotFound, "UserRepo - Ban")

	return err
}

func (r *UserRepo) Unban(ctx context.Context, userID uuid.UUID) error {
	_, err := GetOrNotFound(func() (uuid.UUID, error) { return r.Q(ctx).UnbanUser(ctx, userID) }, apperr.ErrUserNotFound, "UserRepo - Unban")

	return err
}

func (r *UserRepo) SetWasInBannedTeamByIDs(ctx context.Context, userIDs []uuid.UUID, value bool) error {
	if len(userIDs) == 0 {
		return nil
	}

	err := r.Q(ctx).SetWasInBannedTeamByIDs(ctx, sqlc.SetWasInBannedTeamByIDsParams{
		WasInBannedTeam: value,
		Column2:         userIDs,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - SetWasInBannedTeamByIDs: %w", err)
	}

	return nil
}

func (r *UserRepo) UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string) error {
	err := r.Q(ctx).UpdateUserAvatarURL(ctx, sqlc.UpdateUserAvatarURLParams{
		ID:        userID,
		AvatarUrl: &avatarURL,
	})
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateAvatarURL: %w", err)
	}

	return nil
}

func (r *UserRepo) ClearAvatarURL(ctx context.Context, userID uuid.UUID) error {
	if err := r.Q(ctx).ClearUserAvatarURL(ctx, userID); err != nil {
		return fmt.Errorf("UserRepo - ClearAvatarURL: %w", err)
	}

	return nil
}

func (r *UserRepo) ListAllUserAvatarURLs(ctx context.Context) ([]*string, error) {
	urls, err := r.Q(ctx).ListAllUserAvatarURLs(ctx)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - ListAllUserAvatarURLs: %w", err)
	}

	return urls, nil
}
