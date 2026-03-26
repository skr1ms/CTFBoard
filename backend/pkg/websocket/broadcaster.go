package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-wskit"
)

type SolveBroadcaster interface {
	NotifySolve(teamID uuid.UUID, challengeTitle string, points int, isFirstBlood bool)
	NotifyNotification(message, level string)
}

type Broadcaster struct {
	hub *wskit.Hub
	wg  sync.WaitGroup
}

func NewBroadcaster(hub *wskit.Hub) *Broadcaster {
	return &Broadcaster{hub: hub}
}

func (b *Broadcaster) Wait() {
	if b == nil {
		return
	}

	b.wg.Wait()
}

func (b *Broadcaster) NotifySolve(teamID uuid.UUID, challengeTitle string, points int, isFirstBlood bool) {
	if b == nil || b.hub == nil {
		return
	}

	now := time.Now()
	solveEv := wskit.Event{
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
	firstBloodEv := wskit.Event{
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

	b.wg.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = b.hub.BroadcastEvent(ctx, solveEv)

		if isFirstBlood {
			_ = b.hub.BroadcastEvent(ctx, firstBloodEv)
		}
	})
}

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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = b.hub.BroadcastEvent(ctx, ev)
	})
}

var _ SolveBroadcaster = (*Broadcaster)(nil)
