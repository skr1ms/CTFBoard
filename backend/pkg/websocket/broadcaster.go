package websocket

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SolveBroadcaster interface {
	NotifySolve(teamID uuid.UUID, challengeTitle string, points int, isFirstBlood bool)
	NotifyNotification(message, level string)
}

type Broadcaster struct {
	hub *Hub
}

func NewBroadcaster(hub *Hub) *Broadcaster {
	return &Broadcaster{hub: hub}
}

func (b *Broadcaster) NotifySolve(teamID uuid.UUID, challengeTitle string, points int, isFirstBlood bool) {
	if b == nil || b.hub == nil {
		return
	}
	now := time.Now()
	solveEv := Event{
		Type: "scoreboard_update",
		Payload: ScoreboardUpdate{
			Type:      EventTypeSolve,
			TeamID:    teamID.String(),
			Challenge: challengeTitle,
			Points:    points,
			Timestamp: now,
		},
		Timestamp: now,
	}
	firstBloodEv := Event{
		Type: "scoreboard_update",
		Payload: ScoreboardUpdate{
			Type:      EventTypeFirstBlood,
			TeamID:    teamID.String(),
			Challenge: challengeTitle,
			Points:    points,
			Timestamp: now,
		},
		Timestamp: now,
	}
	go func() {
		// Best-effort: bounded by timeout, detached from caller.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		b.hub.BroadcastEvent(ctx, solveEv)
		if isFirstBlood {
			b.hub.BroadcastEvent(ctx, firstBloodEv)
		}
	}()
}

func (b *Broadcaster) NotifyNotification(message, level string) {
	if b == nil || b.hub == nil {
		return
	}
	now := time.Now()
	ev := Event{
		Type: "notification",
		Payload: Notification{
			Type:      EventTypeNotification,
			Message:   message,
			Level:     level,
			Timestamp: now,
		},
		Timestamp: now,
	}
	go func() {
		// Best-effort: bounded by timeout, detached from caller.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		b.hub.BroadcastEvent(ctx, ev)
	}()
}

var _ SolveBroadcaster = (*Broadcaster)(nil)
