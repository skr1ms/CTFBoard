package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// Email
// =============================================================================

type (
	// EmailMessage is the usecase-level email payload. Provider-specific mailer packages
	// adapt this value at the wiring edge.
	EmailMessage struct {
		To      string
		Subject string
		Body    string
		IsHTML  bool
	}

	// EmailSender is the usecase-owned port for transactional email delivery.
	EmailSender interface {
		Send(ctx context.Context, msg EmailMessage) error
	}

	// EmailUseCase handles email delivery for verification, password reset, and related notifications.
	EmailUseCase interface {
		IsEnabled() bool
		SendVerificationEmail(ctx context.Context, user *domain.User) error
		VerifyEmail(ctx context.Context, tokenStr string) error
		SendPasswordResetEmail(ctx context.Context, email string) error
		ResetPasswordRateLimitKey(tokenStr string) string
		ResetPassword(ctx context.Context, tokenStr, newPassword string) error
		ResendVerification(ctx context.Context, userID uuid.UUID) error
		ResendVerificationByEmail(ctx context.Context, email string) error
	}
)

// =============================================================================
// Tracking
// =============================================================================

type (
	// TrackingUseCase records user activity events such as logins and challenge page opens.
	TrackingUseCase interface {
		Track(ctx context.Context, userID uuid.UUID, ip, userAgent string) error
		TrackChallengeOpen(ctx context.Context, userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, ip string) error
		GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int) (*Paginated[*domain.TrackingEntry], error)
	}
)

// =============================================================================
// Cleanup
// =============================================================================

type TrackingCleanupResult struct {
	TrackingDeleted       int64
	ChallengeOpensDeleted int64
}

// Cleaner describes the cleanup use case used by the standalone cleanup command.
type Cleaner interface {
	CleanupDeletedTeams(ctx context.Context, olderThan time.Duration) error
	CleanupOrphanedStorageFiles(ctx context.Context, prefix string) (int, error)
	CleanupOrphanedAvatars(ctx context.Context) (int, error)
	CleanupOldTracking(ctx context.Context, olderThan time.Duration) (*TrackingCleanupResult, error)
}
