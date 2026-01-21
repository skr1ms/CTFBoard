package entity

import (
	"time"

	"github.com/google/uuid"
)

type Hint struct {
	Id          uuid.UUID `json:"id"`
	ChallengeId uuid.UUID `json:"challenge_id"`
	Content     string    `json:"content"`
	Cost        int       `json:"cost"`
	OrderIndex  int       `json:"order_index"`
}

type HintUnlock struct {
	Id         uuid.UUID `json:"id"`
	HintId     uuid.UUID `json:"hint_id"`
	TeamId     uuid.UUID `json:"team_id"`
	UnlockedAt time.Time `json:"unlocked_at"`
}

type Award struct {
	Id          uuid.UUID `json:"id"`
	TeamId      uuid.UUID `json:"team_id"`
	Value       int       `json:"value"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
