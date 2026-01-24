package websocket

import "time"

const (
	EventTypeSolve        = "solve"
	EventTypeFirstBlood   = "first_blood"
	EventTypeNotification = "notification"
)

type ScoreboardUpdate struct {
	Type      string    `json:"type"`
	TeamID    string    `json:"team_id,omitempty"`
	Challenge string    `json:"challenge,omitempty"`
	Points    int       `json:"points,omitempty"`
	Payload   any       `json:"payload,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type Notification struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Level     string    `json:"level"` // info, warning, error, success
	Timestamp time.Time `json:"timestamp"`
}

type Event struct {
	Type      string    `json:"type"`
	Payload   any       `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}
