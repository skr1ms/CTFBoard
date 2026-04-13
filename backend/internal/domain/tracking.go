package domain

import (
	"time"

	"github.com/google/uuid"
)

// TrackingEntry records a user's IP and user-agent at a specific point in time for session analytics.
type TrackingEntry struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	TrackedAt time.Time `json:"tracked_at"`
}

// ChallengeOpen records that a user opened a challenge detail page, used for engagement analytics.
type ChallengeOpen struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	ChallengeID uuid.UUID `json:"challenge_id"`
	IP          string    `json:"ip"`
	OpenedAt    time.Time `json:"opened_at"`
}
