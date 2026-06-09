package repo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// =============================================================================
// Verification token (email)
// =============================================================================

type (
	// VerificationTokenRepository manages one-time email verification and password-reset tokens.
	VerificationTokenRepository interface {
		Create(ctx context.Context, token *domain.VerificationToken) error
		GetByToken(ctx context.Context, token string) (*domain.VerificationToken, error)
		MarkUsed(ctx context.Context, ID uuid.UUID) error
		DeleteExpired(ctx context.Context) error
		DeleteByUserAndType(ctx context.Context, userID uuid.UUID, tokenType domain.TokenType) error
	}
)

// =============================================================================
// Tracking
// =============================================================================

type (
	// TrackingRepository provides persistence operations for user activity and challenge open events.
	TrackingRepository interface {
		Create(ctx context.Context, entry *domain.TrackingEntry) error
		GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.TrackingEntry, error)
		CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
		DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
		CreateChallengeOpen(ctx context.Context, entry *domain.ChallengeOpen) error
		GetChallengeOpensByChallenge(ctx context.Context, challengeID uuid.UUID, limit, offset int) ([]*domain.ChallengeOpen, error)
		DeleteChallengeOpensOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
		CountChallengeOpensByChallenge(ctx context.Context, challengeID uuid.UUID) (int, error)
	}
)

// =============================================================================
// Bracket
// =============================================================================

type (
	// BracketRepository provides persistence operations for competition brackets.
	BracketRepository interface {
		Create(ctx context.Context, bracket *domain.Bracket) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Bracket, error)
		GetByName(ctx context.Context, name string) (*domain.Bracket, error)
		GetAll(ctx context.Context) ([]*domain.Bracket, error)
		Update(ctx context.Context, bracket *domain.Bracket) error
		Delete(ctx context.Context, ID uuid.UUID) error
		ClearAllDefaults(ctx context.Context) error
	}
)

// =============================================================================
// Field
// =============================================================================

type (
	// FieldRepository provides persistence operations for custom entity fields.
	FieldRepository interface {
		Create(ctx context.Context, field *domain.Field) error
		GetByID(ctx context.Context, ID uuid.UUID) (*domain.Field, error)
		GetByEntityType(ctx context.Context, entityType domain.EntityType) ([]*domain.Field, error)
		GetAll(ctx context.Context) ([]*domain.Field, error)
		Update(ctx context.Context, field *domain.Field) error
		Delete(ctx context.Context, ID uuid.UUID) error
	}

	// FieldValueRepository provides persistence operations for dynamic custom field values attached to entities.
	FieldValueRepository interface {
		GetByEntityID(ctx context.Context, entityID uuid.UUID) ([]*domain.FieldValue, error)
		GetAll(ctx context.Context) ([]*domain.FieldValue, error)
		SetValues(ctx context.Context, entityID uuid.UUID, values map[string]string) error
		UpsertValues(ctx context.Context, entityID uuid.UUID, values map[string]string) error
		DeleteByEntityID(ctx context.Context, entityID uuid.UUID) error
	}
)

// =============================================================================
// API Token
// =============================================================================

type (
	// APITokenRepository provides persistence operations for user API tokens.
	APITokenRepository interface {
		Create(ctx context.Context, token *domain.APIToken) error
		GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.APIToken, error)
		GetByTokenHash(ctx context.Context, tokenHash string) (*domain.APIToken, error)
		Delete(ctx context.Context, ID, userID uuid.UUID) error
		DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error
		UpdateLastUsedAt(ctx context.Context, ID uuid.UUID, at time.Time) error
	}
)
