package domain

import (
	"time"

	"github.com/google/uuid"
)

// OAuthUserProfile holds the normalized profile data returned by an OAuth provider.
type OAuthUserProfile struct {
	ID       string
	Email    string
	Username string
}

// OAuthAccount links an external OAuth provider identity to a local user account.
type OAuthAccount struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	Provider       string     `json:"provider"`
	ProviderUserID string     `json:"provider_user_id"`
	AccessToken    string     `json:"-"`
	RefreshToken   *string    `json:"-"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
