package persistent

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

type TeamRepo struct {
	BaseRepo
}

var _ repo.TeamRepository = (*TeamRepo)(nil)

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{BaseRepo: BaseRepo{pool: pool}}
}

type teamRow struct {
	ID                   uuid.UUID
	Name                 string
	InviteToken          uuid.UUID
	InviteTokenExpiresAt *time.Time
	CaptainID            uuid.UUID
	BracketID            *uuid.UUID
	IsSolo               bool
	IsAutoCreated        bool
	IsBanned             bool
	IsHidden             bool
	BannedAt             *time.Time
	BannedReason         *string
	AvatarURL            *string
	CreatedAt            *time.Time
}

func toDomainTeam(r teamRow) *domain.Team {
	return &domain.Team{
		ID:                   r.ID,
		Name:                 r.Name,
		InviteToken:          r.InviteToken,
		InviteTokenExpiresAt: r.InviteTokenExpiresAt,
		CaptainID:            r.CaptainID,
		BracketID:            r.BracketID,
		IsSolo:               r.IsSolo,
		IsAutoCreated:        r.IsAutoCreated,
		IsBanned:             r.IsBanned,
		BannedAt:             r.BannedAt,
		BannedReason:         r.BannedReason,
		IsHidden:             r.IsHidden,
		AvatarURL:            r.AvatarURL,
		CreatedAt:            pgutil.PtrTimeToTime(r.CreatedAt),
	}
}

// teamRowFromSQLC converts the common set of sqlc-generated team fields to teamRow,
// eliminating the repeated pgutil.TimestamptzToTime conversions across all repo methods.
func teamRowFromSQLC(
	id uuid.UUID, name string, inviteToken uuid.UUID,
	inviteTokenExpiresAt, bannedAt, createdAt pgtype.Timestamptz,
	captainID uuid.UUID, bracketID *uuid.UUID,
	isSolo, isAutoCreated, isBanned, isHidden bool,
	bannedReason, avatarURL *string,
) teamRow {
	return teamRow{
		ID: id, Name: name, InviteToken: inviteToken,
		InviteTokenExpiresAt: pgutil.TimestamptzToTime(inviteTokenExpiresAt),
		CaptainID:            captainID, BracketID: bracketID,
		IsSolo: isSolo, IsAutoCreated: isAutoCreated, IsBanned: isBanned, IsHidden: isHidden,
		BannedAt: pgutil.TimestamptzToTime(bannedAt), BannedReason: bannedReason,
		AvatarURL: avatarURL,
		CreatedAt: pgutil.TimestamptzToTime(createdAt),
	}
}
