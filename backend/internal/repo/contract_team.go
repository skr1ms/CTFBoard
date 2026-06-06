package repo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// Team
// =============================================================================

type (
	// TeamRepository provides persistence operations for teams and their audit logs.
	TeamRepository interface {
		Create(ctx context.Context, team *domain.Team) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error)
		GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Team, error)
		GetByInviteToken(ctx context.Context, inviteToken uuid.UUID) (*domain.Team, error)
		GetByName(ctx context.Context, name string) (*domain.Team, error)
		GetSoloTeamByUserID(ctx context.Context, userID uuid.UUID) (*domain.Team, error)
		GetAll(ctx context.Context) ([]*domain.Team, error)
		Search(ctx context.Context, search *string, limit, offset int) ([]*domain.Team, error)
		CountSearch(ctx context.Context, search *string) (int64, error)
		SearchAdmin(ctx context.Context, search *string, limit, offset int) ([]*domain.Team, error)
		CountSearchAdmin(ctx context.Context, search *string) (int64, error)
		CountTeamMembers(ctx context.Context, teamID uuid.UUID) (int, error)
		CountActiveTeams(ctx context.Context) (int, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		HardDeleteTeams(ctx context.Context, cutoffDate time.Time) error
		Ban(ctx context.Context, teamID uuid.UUID, reason string) error
		Unban(ctx context.Context, teamID uuid.UUID) error
		SetHidden(ctx context.Context, teamID uuid.UUID, hidden bool) error
		SetBracket(ctx context.Context, teamID uuid.UUID, bracketID *uuid.UUID) error
		UpdateAdmin(ctx context.Context, teamID uuid.UUID, name *string, captainID, bracketID *uuid.UUID, isHidden *bool) error
		UpdateName(ctx context.Context, teamID uuid.UUID, name string) error
		UpdateCaptain(ctx context.Context, teamID, newCaptainID uuid.UUID) error
		UpdateInviteToken(ctx context.Context, teamID, inviteToken uuid.UUID, expiresAt *time.Time) error
		Lock(ctx context.Context, teamID uuid.UUID) error
		// AcquireAdvisoryLock acquires a session-level advisory lock for the duration
		// of the current transaction. Used to serialize team-count checks
		AcquireAdvisoryLock(ctx context.Context, lockKey int64) error
		CreateAuditLog(ctx context.Context, log *domain.TeamAuditLog) error
		GetLatestAuditLogByTeamIDAndAction(ctx context.Context, teamID uuid.UUID, action string) (*domain.TeamAuditLog, error)
		UpdateAvatarURL(ctx context.Context, teamID uuid.UUID, avatarURL string) error
		ClearAvatarURL(ctx context.Context, teamID uuid.UUID) error
		ListAllTeamAvatarURLs(ctx context.Context) ([]*string, error)
	}
)

// =============================================================================
// Award
// =============================================================================

type (
	// AwardRepository provides persistence operations for team bonus point awards.
	AwardRepository interface {
		Create(ctx context.Context, award *domain.Award) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Award, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.Award, error)
		GetAll(ctx context.Context) ([]*domain.Award, error)
		GetAllForBackup(ctx context.Context) ([]*domain.Award, error)
		GetTeamTotalAwards(ctx context.Context, teamID uuid.UUID) (int, error)
		Delete(ctx context.Context, ID uuid.UUID) error
		DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error
		SoftBanByTeamID(ctx context.Context, teamID uuid.UUID) error
		RestoreByBannedTeamID(ctx context.Context, teamID uuid.UUID) error
	}
)
