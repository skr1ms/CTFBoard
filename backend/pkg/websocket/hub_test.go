package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHub_Run_RegisterUnregister(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run()

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 10),
	}

	hub.Register(client)

	assert.Eventually(t, func() bool {
		return hub.ClientCount() == 1
	}, time.Second, 10*time.Millisecond)

	hub.Unregister(client)

	assert.Eventually(t, func() bool {
		return hub.ClientCount() == 0
	}, time.Second, 10*time.Millisecond)
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run()

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 10),
	}
	hub.Register(client)

	select {
	case <-client.send:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for connected")
	}

	msg := []byte("hello")
	hub.Broadcast(msg)

	select {
	case received := <-client.send:
		assert.Equal(t, msg, received)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestHub_BroadcastEvent_Redis(t *testing.T) {
	db, redisClient := redismock.NewClientMock()
	hub := NewHub(db, "test-channel")
	go hub.Run()

	event := Event{
		Type:      "test",
		Payload:   "payload",
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	redisClient.ExpectPublish("test-channel", data).SetVal(1)

	hub.BroadcastEvent(event)

	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestHub_BroadcastEvent_LocalFallback(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run()

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 10),
	}
	hub.Register(client)

	select {
	case <-client.send:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for connected")
	}

	event := Event{
		Type:      "test",
		Payload:   map[string]string{"foo": "bar"},
		Timestamp: time.Now(),
	}

	hub.BroadcastEvent(event)

	select {
	case received := <-client.send:
		var receivedEvent Event
		err := json.Unmarshal(received, &receivedEvent)
		assert.NoError(t, err)
		assert.Equal(t, "test", receivedEvent.Type)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}
