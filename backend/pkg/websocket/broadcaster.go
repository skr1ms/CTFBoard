package websocket

import (
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
		b.hub.BroadcastEvent(solveEv)
		if isFirstBlood {
			b.hub.BroadcastEvent(firstBloodEv)
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
	go func() { b.hub.BroadcastEvent(ev) }()
}

var _ SolveBroadcaster = (*Broadcaster)(nil)
