package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"

	// OAuthOnlyPasswordSentinel is stored in PasswordHash for users who signed up via OAuth.
	// They cannot log in with email/password; Login rejects before bcrypt.
	OAuthOnlyPasswordSentinel = "oauth-only"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	TeamID       *uuid.UUID `json:"team_id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	IsVerified   bool       `json:"is_verified"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	IsBanned     bool       `json:"is_banned"`
	BannedAt     *time.Time `json:"banned_at,omitempty"`
	BannedReason *string    `json:"banned_reason,omitempty"`
}
