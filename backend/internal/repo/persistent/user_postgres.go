package persistent

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

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
