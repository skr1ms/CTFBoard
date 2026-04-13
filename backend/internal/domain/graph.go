package domain

import (
	"time"

	"github.com/google/uuid"
)

// ScorePoint is a single data point in a team's score timeline, pairing a timestamp with a cumulative score.
type ScorePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     int       `json:"score"`
}

// TeamTimeline holds the ordered sequence of score events for a single team.
type TeamTimeline struct {
	TeamID   uuid.UUID    `json:"team_id"`
	TeamName string       `json:"team_name"`
	Timeline []ScorePoint `json:"timeline"`
}

// TimeRange defines an inclusive start-to-end time window used to bound graph queries.
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ScoreboardGraph is the complete score-over-time dataset for the top teams, bounded by a time range.
type ScoreboardGraph struct {
	Range TimeRange      `json:"range"`
	Teams []TeamTimeline `json:"teams"`
}
