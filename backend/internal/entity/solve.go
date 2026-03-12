package entity

import (
	"time"

	"github.com/google/uuid"
)

type Solve struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	TeamID        uuid.UUID  `json:"team_id"`
	ChallengeID   uuid.UUID  `json:"challenge_id"`
	SolvedAt      time.Time  `json:"solved_at"`
	PointsAtSolve int        `json:"points_at_solve"`
	BannedTeamID  *uuid.UUID `json:"banned_team_id,omitempty"`
	BannedUserID  *uuid.UUID `json:"banned_user_id,omitempty"`
}

type SolveWithDetails struct {
	Solve
	Username          string `json:"username,omitempty"`
	TeamName          string `json:"team_name,omitempty"`
	ChallengeTitle    string `json:"challenge_title,omitempty"`
	ChallengeCategory string `json:"challenge_category,omitempty"`
	ChallengePoints   int    `json:"challenge_points,omitempty"`
}

// ScoreboardEntry is a single row in the scoreboard ranking.
type ScoreboardEntry struct {
	TeamID   uuid.UUID
	TeamName string
	Points   int
	SolvedAt time.Time
}

// FirstBloodEntry identifies the first solver of a challenge.
type FirstBloodEntry struct {
	UserID   uuid.UUID
	Username string
	TeamID   uuid.UUID
	TeamName string
	SolvedAt time.Time
}
