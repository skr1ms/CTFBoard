package entity

import "time"

type Solve struct {
	Id          string    `json:"id"`
	UserId      string    `json:"user_id"`
	TeamId      string    `json:"team_id"`
	ChallengeId string    `json:"challenge_id"`
	SolvedAt    time.Time `json:"solved_at"`
}
