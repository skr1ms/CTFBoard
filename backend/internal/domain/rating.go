package domain

import (
	"time"

	"github.com/google/uuid"
)

// Rating is a numerical score with an optional review that a user submits for a challenge after solving it.
type Rating struct {
	ID          uuid.UUID `json:"id"`
	ChallengeID uuid.UUID `json:"challenge_id"`
	UserID      uuid.UUID `json:"user_id"`
	TeamID      uuid.UUID `json:"team_id"`
	Value       int       `json:"value"`
	Review      string    `json:"review"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
