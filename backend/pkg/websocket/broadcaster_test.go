package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBroadcaster(t *testing.T) {
	hub := NewHub(nil, "")
	b := NewBroadcaster(hub)
	require.NotNil(t, b)
}

func TestBroadcaster_NotifySolve_NilBroadcaster(t *testing.T) {
	var b *Broadcaster
	b.NotifySolve(uuid.New(), "ch", 100, false)
	b.NotifySolve(uuid.New(), "ch", 100, true)
}

func TestBroadcaster_NotifySolve_NilHub(t *testing.T) {
	b := NewBroadcaster(nil)
	b.NotifySolve(uuid.New(), "ch", 100, false)
	b.NotifySolve(uuid.New(), "ch", 100, true)
}

func TestBroadcaster_NotifySolve_WithHub_NoFirstBlood(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run(context.Background())

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 4),
	}
	hub.Register(client)

	select {
	case <-client.send:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for connected")
	}

	teamID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	b := NewBroadcaster(hub)
	b.NotifySolve(teamID, "Challenge A", 150, false)

	select {
	case data := <-client.send:
		var ev Event
		require.NoError(t, json.Unmarshal(data, &ev))
		assert.Equal(t, "scoreboard_update", ev.Type)
		payload, ok := ev.Payload.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, EventTypeSolve, payload["type"])
		assert.Equal(t, teamID.String(), payload["team_id"])
		assert.Equal(t, "Challenge A", payload["challenge"])
		assert.EqualValues(t, 150, payload["points"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestBroadcaster_NotifySolve_WithHub_FirstBlood(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run(context.Background())

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 4),
	}
	hub.Register(client)

	select {
	case <-client.send:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for connected")
	}

	teamID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	b := NewBroadcaster(hub)
	b.NotifySolve(teamID, "Challenge B", 200, true)

	var solveEv, fbEv Event
	select {
	case data := <-client.send:
		require.NoError(t, json.Unmarshal(data, &solveEv))
		assert.Equal(t, "scoreboard_update", solveEv.Type)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for solve event")
	}
	select {
	case data := <-client.send:
		require.NoError(t, json.Unmarshal(data, &fbEv))
		assert.Equal(t, "scoreboard_update", fbEv.Type)
		payload, ok := fbEv.Payload.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, EventTypeFirstBlood, payload["type"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for first blood event")
	}
}

func TestBroadcaster_NotifyNotification_NilBroadcaster(t *testing.T) {
	var b *Broadcaster
	b.NotifyNotification("msg", "info")
}

func TestBroadcaster_NotifyNotification_NilHub(t *testing.T) {
	b := NewBroadcaster(nil)
	b.NotifyNotification("msg", "warning")
}

func TestBroadcaster_NotifyNotification_WithHub(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run(context.Background())

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 4),
	}
	hub.Register(client)

	select {
	case <-client.send:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for connected")
	}

	b := NewBroadcaster(hub)
	b.NotifyNotification("Hello", "success")

	select {
	case data := <-client.send:
		var ev Event
		require.NoError(t, json.Unmarshal(data, &ev))
		assert.Equal(t, "notification", ev.Type)
		payload, ok := ev.Payload.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, EventTypeNotification, payload["type"])
		assert.Equal(t, "Hello", payload["message"])
		assert.Equal(t, "success", payload["level"])
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for notification event")
	}
}
