package entity

import (
	"time"

	"github.com/google/uuid"
)

type TokenType string

const (
	TokenTypeEmailVerification TokenType = "email_verification" //nolint:gosec // G101: This is an enum value, not a credential
	TokenTypePasswordReset     TokenType = "password_reset"
)

type VerificationToken struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	Token     string
	Type      TokenType
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (t *VerificationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

func (t *VerificationToken) IsUsed() bool {
	return t.UsedAt != nil
}
