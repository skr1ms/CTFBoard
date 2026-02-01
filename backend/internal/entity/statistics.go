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
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	Points     int       `json:"points"`
	SolveCount int       `json:"solve_count"`
}

type ChallengeSolveEntry struct {
	TeamID   uuid.UUID `json:"team_id"`
	TeamName string    `json:"team_name"`
	SolvedAt time.Time `json:"solved_at"`
}

type ChallengeDetailStats struct {
	ID               uuid.UUID             `json:"id"`
	Title            string                `json:"title"`
	Category         string                `json:"category"`
	Points           int                   `json:"points"`
	SolveCount       int                   `json:"solve_count"`
	TotalTeams       int                   `json:"total_teams"`
	PercentageSolved float64               `json:"percentage_solved"`
	FirstBlood       *ChallengeSolveEntry  `json:"first_blood,omitempty"`
	Solves           []ChallengeSolveEntry `json:"solves"`
}

type ScoreboardHistoryEntry struct {
	TeamID    uuid.UUID `json:"team_id"`
	TeamName  string    `json:"team_name"`
	Points    int       `json:"points"`
	Timestamp time.Time `json:"timestamp"`
}
