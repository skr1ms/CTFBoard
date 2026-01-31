package entity

import (
	"time"

	"github.com/google/uuid"
)

type Hint struct {
	ID          uuid.UUID `json:"id"`
	ChallengeID uuid.UUID `json:"challenge_id"`
	Content     string    `json:"content"`
	Cost        int       `json:"cost"`
	OrderIndex  int       `json:"order_index"`
}

type HintUnlock struct {
	ID         uuid.UUID `json:"id"`
	HintID     uuid.UUID `json:"hint_id"`
	TeamID     uuid.UUID `json:"team_id"`
	UnlockedAt time.Time `json:"unlocked_at"`
}

type Award struct {
	ID          uuid.UUID  `json:"id"`
	TeamID      uuid.UUID  `json:"team_id"`
	Value       int        `json:"value"`
	Description string     `json:"description"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
