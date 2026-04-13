// Package websocket holds AstroCTFb-specific WebSocket payloads and event name constants
// The wire format envelope (type, payload, timestamp) is go-wskit.Event (module github.com/wahrwelt-kit/go-wskit)
// hub, client lifecycle, Redis pub/sub and ping keep-alive are provided by go-wskit on top of coder/websocket
//
// # Endpoint and auth
//
// Clients use GET /api/v1/ws (or the route your router mounts). Upgrade uses coder/websocket
// Authenticated users only (JWT or API token); otherwise 401
//
// After registration the server sends a "connected" message (payload null), built with
// wskit.NewEvent and constants from this package where applicable
//
// # Keep-alive
//
// go-wskit sends WebSocket ping frames on a fixed interval (default 30s) with a per-write
// timeout (default 10s). Clients should answer pings as usual
//
// # Top-level message types
//
// Envelope JSON shape
//
//		{"type": "<name>", "payload": {...}, "timestamp": "RFC3339"}
//
//	  - type "connected" - once per connection; payload null
//	  - type "scoreboard_update" - solve or first blood; payload is ScoreboardUpdate JSON
//	  - type "notification" - admin broadcast; payload is Notification JSON
//
// Inner payload field "type" uses EventTypeSolve, EventTypeFirstBlood, EventTypeNotification
// for scoreboard_update and notification payloads respectively
//
// # Reconnection
//
// No session state on the server; clients should reconnect with backoff and expect a new "connected"
package websocket

import "time"

// Event type constants used in wskit.Event.Type (envelope) and in inner payload Type fields.
const (
	EventTypeConnected    = "connected"
	EventTypeSolve        = "solve"
	EventTypeFirstBlood   = "first_blood"
	EventTypeNotification = "notification"
)

// ScoreboardUpdate is the JSON payload inside a "scoreboard_update" envelope (wskit.Event.Payload)
//
// Example (solve)
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
// Example (first_blood)
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

// Notification is the JSON payload inside a "notification" envelope (wskit.Event.Payload)
// Level is one of: "info", "warning", "error", "success"
//
// Example
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
