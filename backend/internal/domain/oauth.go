package domain

import (
	"time"

	"github.com/google/uuid"
)

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
