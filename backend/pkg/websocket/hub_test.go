package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

	time.Sleep(50 * time.Millisecond)

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
	mockRedis := mocks.NewMockRedisClient(t)
	hub := NewHub(mockRedis, "test-channel")

	event := Event{
		Type:      "test",
		Payload:   "payload",
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(event)

	cmd := redis.NewIntCmd(context.Background())
	mockRedis.On("Publish", mock.Anything, "test-channel", data).Return(cmd)

	hub.BroadcastEvent(event)

	mockRedis.AssertExpectations(t)
}

func TestHub_BroadcastEvent_LocalFallback(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run()

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 10),
	}
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

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
