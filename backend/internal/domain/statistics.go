package domain

import (
	"time"

	"github.com/google/uuid"
)

// GeneralStats holds top-level competition counters shown on the admin dashboard.
type GeneralStats struct {
	UserCount      int `json:"user_count"`
	TeamCount      int `json:"team_count"`
	ChallengeCount int `json:"challenge_count"`
	SolveCount     int `json:"solve_count"`
}

// ChallengeStats is a summary row for a challenge used in admin statistics listings.
type ChallengeStats struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	Points     int       `json:"points"`
	SolveCount int       `json:"solve_count"`
}

// ChallengeSolveEntry identifies a team that solved a challenge and when they did so.
type ChallengeSolveEntry struct {
	TeamID   uuid.UUID `json:"team_id"`
	TeamName string    `json:"team_name"`
	SolvedAt time.Time `json:"solved_at"`
}

// ChallengeDetailStats aggregates solve rate and solver list for a single challenge detail view.
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

// ScoreboardHistoryEntry represents a team's point total at a specific point in time for history charts.
type ScoreboardHistoryEntry struct {
	TeamID    uuid.UUID `json:"team_id"`
	TeamName  string    `json:"team_name"`
	Points    int       `json:"points"`
	Timestamp time.Time `json:"timestamp"`
}

// ChallengeSolvePercentage pairs a challenge with its solve rate relative to the total team count.
type ChallengeSolvePercentage struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	SolveCount int       `json:"solve_count"`
	TotalTeams int       `json:"total_teams"`
	Percentage float64   `json:"percentage"`
}

// ScoreDistributionBucket counts how many teams fall within a named score range bucket.
type ScoreDistributionBucket struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// SubmissionTimeSeries holds correct and incorrect submission counts for a single date bucket.
type SubmissionTimeSeries struct {
	Date      string `json:"date"`
	Correct   int    `json:"correct"`
	Incorrect int    `json:"incorrect"`
}

// SubmissionTimeSeriesStats wraps the per-day submission series with overall totals.
type SubmissionTimeSeriesStats struct {
	Items          []*SubmissionTimeSeries `json:"items"`
	TotalCorrect   int                     `json:"total_correct"`
	TotalIncorrect int                     `json:"total_incorrect"`
}

// RegistrationTimePoint records how many users registered on a given date, used for growth charts.
type RegistrationTimePoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// SolveMatrixRow is a single cell in the admin solve matrix, indicating whether a team solved a challenge.
type SolveMatrixRow struct {
	TeamID            uuid.UUID  `json:"team_id"`
	TeamName          string     `json:"team_name"`
	ChallengeID       uuid.UUID  `json:"challenge_id"`
	ChallengeTitle    string     `json:"challenge_title"`
	ChallengeCategory string     `json:"challenge_category"`
	Solved            bool       `json:"solved"`
	SolvedAt          *time.Time `json:"solved_at,omitempty"`
}

// AdminStatisticsFunnel summarizes challenge engagement from open to attempt to solve.
type AdminStatisticsFunnel struct {
	Challenges []*FunnelChallengeRow `json:"challenges"`
	Teams      []*FunnelTeamRow      `json:"teams"`
	TeamCells  []*FunnelTeamCell     `json:"team_cells"`
	Users      []*FunnelUserRow      `json:"users"`
	UserCells  []*FunnelUserCell     `json:"user_cells"`
}

type FunnelChallengeRow struct {
	ChallengeID       uuid.UUID `json:"challenge_id"`
	ChallengeTitle    string    `json:"challenge_title"`
	ChallengeCategory string    `json:"challenge_category"`
	OpenedCount       int       `json:"opened_count"`
	AttemptedCount    int       `json:"attempted_count"`
	SolvedCount       int       `json:"solved_count"`
}

type FunnelTeamRow struct {
	TeamID         uuid.UUID `json:"team_id"`
	TeamName       string    `json:"team_name"`
	OpenedCount    int       `json:"opened_count"`
	AttemptedCount int       `json:"attempted_count"`
	SolvedCount    int       `json:"solved_count"`
}

type FunnelTeamCell struct {
	TeamID           uuid.UUID  `json:"team_id"`
	ChallengeID      uuid.UUID  `json:"challenge_id"`
	Opened           bool       `json:"opened"`
	Attempted        bool       `json:"attempted"`
	Solved           bool       `json:"solved"`
	FirstOpenedAt    *time.Time `json:"first_opened_at,omitempty"`
	FirstAttemptedAt *time.Time `json:"first_attempted_at,omitempty"`
	SolvedAt         *time.Time `json:"solved_at,omitempty"`
}

type FunnelUserRow struct {
	UserID         uuid.UUID `json:"user_id"`
	Username       string    `json:"username"`
	OpenedCount    int       `json:"opened_count"`
	AttemptedCount int       `json:"attempted_count"`
	SolvedCount    int       `json:"solved_count"`
}

type FunnelUserCell struct {
	UserID           uuid.UUID  `json:"user_id"`
	ChallengeID      uuid.UUID  `json:"challenge_id"`
	Opened           bool       `json:"opened"`
	Attempted        bool       `json:"attempted"`
	Solved           bool       `json:"solved"`
	FirstOpenedAt    *time.Time `json:"first_opened_at,omitempty"`
	FirstAttemptedAt *time.Time `json:"first_attempted_at,omitempty"`
	SolvedAt         *time.Time `json:"solved_at,omitempty"`
}
