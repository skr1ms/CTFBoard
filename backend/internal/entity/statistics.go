package entity

import (
	"time"

	"github.com/google/uuid"
)

type GeneralStats struct {
	UserCount      int `json:"user_count"`
	TeamCount      int `json:"team_count"`
	ChallengeCount int `json:"challenge_count"`
	SolveCount     int `json:"solve_count"`
}

type ChallengeStats struct {
	Id         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	Points     int       `json:"points"`
	SolveCount int       `json:"solve_count"`
}

type ScoreboardHistoryEntry struct {
	TeamId    uuid.UUID `json:"team_id"`
	TeamName  string    `json:"team_name"`
	Points    int       `json:"points"`
	Timestamp time.Time `json:"timestamp"`
}
