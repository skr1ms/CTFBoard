package domain

import (
	"time"

	"github.com/google/uuid"
)

// TokenType is a string-backed enum indicating the purpose of a verification token.
type TokenType string

const (
	// TokenTypeEmailVerification identifies a token issued to verify a user's email address.
	TokenTypeEmailVerification TokenType = "email_verification" // #nosec G101
	// TokenTypePasswordReset identifies a token issued for a password reset flow.
	TokenTypePasswordReset TokenType = "password_reset"
)

// VerificationToken is a short-lived, single-use token sent to a user for email
// verification or password reset.
type VerificationToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	Type      TokenType
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// IsExpired returns true if the token's expiry time is in the past.
func (t *VerificationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has already been consumed.
func (t *VerificationToken) IsUsed() bool {
	return t.UsedAt != nil
}
