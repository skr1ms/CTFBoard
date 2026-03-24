package domain

import (
	"time"

	"github.com/google/uuid"
)

type TrackingEntry struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	TrackedAt time.Time `json:"tracked_at"`
}

type ChallengeOpen struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	ChallengeID uuid.UUID `json:"challenge_id"`
	IP          string    `json:"ip"`
	OpenedAt    time.Time `json:"opened_at"`
}
