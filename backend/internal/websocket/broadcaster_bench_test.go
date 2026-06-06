package websocket_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-wskit"

	ws "github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
)

func BenchmarkNotifySolve(b *testing.B) {
	hub := wskit.NewHub()

	bc := ws.NewBroadcaster(context.Background(), hub)
	defer bc.Wait()

	teamID := uuid.New()

	b.ReportAllocs()

	for b.Loop() {
		bc.NotifySolve(teamID, "Web Challenge", 500, false)
	}
}

func BenchmarkNotifySolve_FirstBlood(b *testing.B) {
	hub := wskit.NewHub()

	bc := ws.NewBroadcaster(context.Background(), hub)
	defer bc.Wait()

	teamID := uuid.New()

	b.ReportAllocs()

	for b.Loop() {
		bc.NotifySolve(teamID, "Crypto Challenge", 1000, true)
	}
}

func BenchmarkNotifyNotification(b *testing.B) {
	hub := wskit.NewHub()

	bc := ws.NewBroadcaster(context.Background(), hub)
	defer bc.Wait()

	b.ReportAllocs()

	for b.Loop() {
		bc.NotifyNotification("competition starts in 5 minutes", "info")
	}
}
