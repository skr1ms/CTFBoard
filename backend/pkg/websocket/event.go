// Package websocket provides real-time event broadcasting via WebSocket connections.
//
// # Connection Protocol
//
// Clients connect to GET /ws. The server upgrades the HTTP connection to WebSocket
// using the coder/websocket library. No authentication is required for the WS
// connection itself (events are public scoreboard data and notifications).
//
// On successful connection, the server sends a "connected" event:
//
//	{"type": "connected", "payload": null, "timestamp": "2025-01-01T00:00:00Z"}
//
// # Keep-Alive
//
// The server sends WebSocket ping frames every 54 seconds (pingPeriod).
// If no pong is received within 60 seconds (pongWait), the connection is closed.
// Clients should respond to ping frames automatically (most WebSocket libraries do this).
//
// # Event Types
//
// All events use the Event envelope:
//
//	{"type": "<event_type>", "payload": {...}, "timestamp": "RFC3339"}
//
// Supported event types:
//
//   - "connected"          - sent once upon successful connection (payload: null)
//   - "scoreboard_update"  - sent when a team solves a challenge or achieves first blood
//   - "notification"       - sent when an admin broadcasts a notification
//
// # Reconnection
//
// The server does not maintain session state between connections.
// Clients should implement reconnection with exponential backoff.
// On reconnect, the client receives a fresh "connected" event.
package websocket

import "time"

const (
	EventTypeConnected    = "connected"
	EventTypeSolve        = "solve"
	EventTypeFirstBlood   = "first_blood"
	EventTypeNotification = "notification"
)

// ScoreboardUpdate is the payload for "scoreboard_update" events.
//
// Example (solve):
//
//	{
//	  "type": "scoreboard_update",
//	  "payload": {
//	    "type": "solve",
//	    "team_id": "550e8400-e29b-41d4-a716-446655440000",
//	    "challenge": "SQL Injection 101",
//	    "points": 500,
//	    "timestamp": "2025-01-15T14:30:00Z"
//	  },
//	  "timestamp": "2025-01-15T14:30:00Z"
//	}
//
// Example (first_blood):
//
//	{
//	  "type": "scoreboard_update",
//	  "payload": {
//	    "type": "first_blood",
//	    "team_id": "550e8400-e29b-41d4-a716-446655440000",
//	    "challenge": "Binary Exploitation",
//	    "points": 1000,
//	    "timestamp": "2025-01-15T14:30:00Z"
//	  },
//	  "timestamp": "2025-01-15T14:30:00Z"
//	}
type ScoreboardUpdate struct {
	Type      string    `json:"type"`
	TeamID    string    `json:"team_id,omitempty"`
	Challenge string    `json:"challenge,omitempty"`
	Points    int       `json:"points,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Notification is the payload for "notification" events.
// Level is one of: "info", "warning", "error", "success".
//
// Example:
//
//	{
//	  "type": "notification",
//	  "payload": {
//	    "type": "notification",
//	    "message": "New hint released for challenge 'Crypto 101'",
//	    "level": "info",
//	    "timestamp": "2025-01-15T15:00:00Z"
//	  },
//	  "timestamp": "2025-01-15T15:00:00Z"
//	}
type Notification struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
}

// Event is the top-level envelope for all WebSocket messages.
// Type determines the structure of Payload.
type Event struct {
	Type      string    `json:"type"`
	Payload   any       `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}
