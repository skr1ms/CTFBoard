package entity

import "time"

type TokenType string

const (
	TokenTypeEmailVerification TokenType = "email_verification" // #nosec G101
	TokenTypePasswordReset     TokenType = "password_reset"
)

type VerificationToken struct {
	Id        string
	UserId    string
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
