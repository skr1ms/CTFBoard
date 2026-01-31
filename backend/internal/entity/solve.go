package entity

import (
	"time"

	"github.com/google/uuid"
)

type Solve struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	TeamID      uuid.UUID `json:"team_id"`
	ChallengeID uuid.UUID `json:"challenge_id"`
	SolvedAt    time.Time `json:"solved_at"`
}
