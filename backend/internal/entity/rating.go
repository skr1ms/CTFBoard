package entity

import (
	"time"

	"github.com/google/uuid"
)

type CTFEvent struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Weight    float64   `json:"weight"`
	CreatedAt time.Time `json:"created_at"`
}

type TeamRating struct {
	ID           uuid.UUID `json:"id"`
	TeamID       uuid.UUID `json:"team_id"`
	CTFEventID   uuid.UUID `json:"ctf_event_id"`
	Rank         int       `json:"rank"`
	Score        int       `json:"score"`
	RatingPoints float64   `json:"rating_points"`
	CreatedAt    time.Time `json:"created_at"`
}

type GlobalRating struct {
	TeamID      uuid.UUID `json:"team_id"`
	TeamName    string    `json:"team_name"`
	TotalPoints float64   `json:"total_points"`
	EventsCount int       `json:"events_count"`
	BestRank    *int      `json:"best_rank,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}
