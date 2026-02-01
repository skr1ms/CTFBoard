package entity

import (
	"time"

	"github.com/google/uuid"
)

type ScorePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     int       `json:"score"`
}

type TeamTimeline struct {
	TeamID   uuid.UUID    `json:"team_id"`
	TeamName string       `json:"team_name"`
	Timeline []ScorePoint `json:"timeline"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type ScoreboardGraph struct {
	Range TimeRange      `json:"range"`
	Teams []TeamTimeline `json:"teams"`
}
