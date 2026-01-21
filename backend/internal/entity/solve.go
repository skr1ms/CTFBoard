package entity

import (
	"time"

	"github.com/google/uuid"
)

type Solve struct {
	Id          uuid.UUID `json:"id"`
	UserId      uuid.UUID `json:"user_id"`
	TeamId      uuid.UUID `json:"team_id"`
	ChallengeId uuid.UUID `json:"challenge_id"`
	SolvedAt    time.Time `json:"solved_at"`
}
