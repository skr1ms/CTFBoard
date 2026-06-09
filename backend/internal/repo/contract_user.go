package repo

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// User
// =============================================================================

type (
	UserAdminBanStatus string

	UserAdminSearchFilter struct {
		Search    *string
		BanStatus UserAdminBanStatus
	}

	// UserRepository provides persistence operations for user accounts.
	UserRepository interface {
		Create(ctx context.Context, user *domain.User) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.User, error)
		GetByEmail(ctx context.Context, email string) (*domain.User, error)
		GetByUsername(ctx context.Context, username string) (*domain.User, error)
		GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error)
		GetByTeamIDs(ctx context.Context, teamIDs []uuid.UUID) (map[uuid.UUID][]*domain.User, error)
		GetAll(ctx context.Context) ([]*domain.User, error)
		Search(ctx context.Context, search *string, limit, offset int) ([]*domain.User, error)
		CountSearch(ctx context.Context, search *string) (int64, error)
		SearchAdmin(ctx context.Context, filter UserAdminSearchFilter, limit, offset int) ([]*domain.User, error)
		CountSearchAdmin(ctx context.Context, filter UserAdminSearchFilter) (int64, error)
		SearchByIP(ctx context.Context, ip string, limit, offset int) ([]*domain.User, error)
		CountSearchByIP(ctx context.Context, ip string) (int64, error)
		SearchAdminByIP(ctx context.Context, ip string, banStatus UserAdminBanStatus, limit, offset int) ([]*domain.User, error)
		CountSearchAdminByIP(ctx context.Context, ip string, banStatus UserAdminBanStatus) (int64, error)
		CountActiveUsers(ctx context.Context) (int64, error)
		UpdateTeamID(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID) error
		UpdateTeamIDBatch(ctx context.Context, userIDs []uuid.UUID, teamID *uuid.UUID) error
		FilterIDsByTeamIDNull(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error)
		FilterIDsByTeamIDNullAndNotBanned(ctx context.Context, userIDs []uuid.UUID) ([]uuid.UUID, error)
		SetVerified(ctx context.Context, userID uuid.UUID) error
		SetUnverified(ctx context.Context, userID uuid.UUID) error
		UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
		UpdateAdmin(ctx context.Context, userID uuid.UUID, username, email, role, passwordHash *string, isVerified *bool) error
		UpdateProfile(ctx context.Context, userID uuid.UUID, username, email, passwordHash *string) error
		Delete(ctx context.Context, userID uuid.UUID) error
		Lock(ctx context.Context, userID uuid.UUID) error
		Ban(ctx context.Context, userID uuid.UUID, reason string) error
		Unban(ctx context.Context, userID uuid.UUID) error
		SetWasInBannedTeamByIDs(ctx context.Context, userIDs []uuid.UUID, value bool) error
		AcquireAdvisoryLock(ctx context.Context, lockKey int64) error
		UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string) error
		ClearAvatarURL(ctx context.Context, userID uuid.UUID) error
		ListAllUserAvatarURLs(ctx context.Context) ([]*string, error)
	}
)

const (
	UserAdminBanStatusAll           UserAdminBanStatus = "all"
	UserAdminBanStatusNotBanned     UserAdminBanStatus = "not_banned"
	UserAdminBanStatusDirect        UserAdminBanStatus = "direct"
	UserAdminBanStatusTeamInherited UserAdminBanStatus = "team_inherited"
	UserAdminBanStatusBlocked       UserAdminBanStatus = "blocked"
)

// =============================================================================
// BanAppeal
// =============================================================================

type (
	// BanAppealRepository provides persistence for ban appeal records.
	BanAppealRepository interface {
		Create(ctx context.Context, appeal *domain.BanAppeal) error
		GetByID(ctx context.Context, id uuid.UUID) (*domain.BanAppeal, error)
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.BanAppeal, error)
		GetLatestByUserID(ctx context.Context, userID uuid.UUID) (*domain.BanAppeal, error)
		List(ctx context.Context, decision *domain.AppealDecision, limit, offset int) ([]*domain.BanAppeal, int64, error)
		Update(ctx context.Context, appeal *domain.BanAppeal) error
	}
)

// =============================================================================
// OAuth
// =============================================================================

type (
	// OAuthAccountRepository provides persistence operations for linked OAuth provider accounts.
	OAuthAccountRepository interface {
		Create(ctx context.Context, acc *domain.OAuthAccount) error
		Upsert(ctx context.Context, acc *domain.OAuthAccount) error
		GetByProvider(ctx context.Context, provider, providerUserID string) (*domain.OAuthAccount, error)
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.OAuthAccount, error)
	}
)
