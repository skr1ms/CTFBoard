package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-wskit"
)

// SolveBroadcaster is the interface for broadcasting solve and notification events
// to all connected WebSocket/SSE clients.
type SolveBroadcaster interface {
	NotifySolve(teamID uuid.UUID, challengeTitle string, points int, isFirstBlood bool)
	NotifyNotification(message, level string)
}

// Broadcaster wraps a wskit.Hub and dispatches real-time events asynchronously.
// All dispatch goroutines are tracked via an internal WaitGroup; call Wait before shutdown.
type Broadcaster struct {
	ctx context.Context
	hub *wskit.Hub
	wg  sync.WaitGroup
}

const broadcastTimeout = 5 * time.Second

// NewBroadcaster creates a Broadcaster backed by the given wskit.Hub.
func NewBroadcaster(ctx context.Context, hub *wskit.Hub) *Broadcaster {
	if ctx == nil {
		panic("websocket.NewBroadcaster: nil context")
	}

	return &Broadcaster{ctx: ctx, hub: hub}
}

// Wait blocks until all in-flight broadcast goroutines complete.
// Safe to call on a nil Broadcaster.
func (b *Broadcaster) Wait() {
	if b == nil {
		return
	}

	b.wg.Wait()
}

// NotifySolve dispatches a "scoreboard_update" event for the solved challenge.
// When isFirstBlood is true, a second event with type EventTypeFirstBlood is sent
// in the same goroutine after the solve event. Both share a 5-second context timeout.
// No-op when the Broadcaster or its hub is nil.
func (b *Broadcaster) NotifySolve(teamID uuid.UUID, challengeTitle string, points int, isFirstBlood bool) {
	if b == nil || b.hub == nil {
		return
	}

	now := time.Now()
	solveEv := wskit.Event{
		Type: EventTypeScoreboardUpdate,
		Payload: ScoreboardUpdate{
			Type:      EventTypeSolve,
			TeamID:    teamID.String(),
			Challenge: challengeTitle,
			Points:    points,
			Timestamp: now,
		},
		Timestamp: now,
	}
	firstBloodEv := wskit.Event{
		Type: EventTypeScoreboardUpdate,
		Payload: ScoreboardUpdate{
			Type:      EventTypeFirstBlood,
			TeamID:    teamID.String(),
			Challenge: challengeTitle,
			Points:    points,
			Timestamp: now,
		},
		Timestamp: now,
	}

	b.wg.Go(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(b.ctx), broadcastTimeout)
		defer cancel()

		_ = b.hub.BroadcastEvent(ctx, solveEv)

		if isFirstBlood {
			_ = b.hub.BroadcastEvent(ctx, firstBloodEv)
		}
	})
}

// NotifyNotification dispatches a "notification" event with the given message and level
// asynchronously with a 5-second context timeout. No-op when the Broadcaster or hub is nil.
func (b *Broadcaster) NotifyNotification(message, level string) {
	if b == nil || b.hub == nil {
		return
	}

	now := time.Now()
	ev := wskit.Event{
		Type: "notification",
		Payload: Notification{
			Type:      EventTypeNotification,
			Message:   message,
			Level:     level,
			Timestamp: now,
		},
		Timestamp: now,
	}

	b.wg.Go(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(b.ctx), broadcastTimeout)
		defer cancel()

		_ = b.hub.BroadcastEvent(ctx, ev)
	})
}

var _ SolveBroadcaster = (*Broadcaster)(nil)
