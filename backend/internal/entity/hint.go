package entity

import "time"

type Hint struct {
	Id          string `json:"id"`
	ChallengeId string `json:"challenge_id"`
	Content     string `json:"content"`
	Cost        int    `json:"cost"`
	OrderIndex  int    `json:"order_index"`
}

type HintUnlock struct {
	Id         string    `json:"id"`
	HintId     string    `json:"hint_id"`
	TeamId     string    `json:"team_id"`
	UnlockedAt time.Time `json:"unlocked_at"`
}

type Award struct {
	Id          string    `json:"id"`
	TeamId      string    `json:"team_id"`
	Value       int       `json:"value"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
