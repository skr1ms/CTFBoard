package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	websocketlib "github.com/coder/websocket"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHub_Run_RegisterUnregister(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run(context.Background())

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
	go hub.Run(context.Background())

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
	go hub.Run(context.Background())

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

func TestHub_Run_ExitsOnContextCancel(t *testing.T) {
	hub := NewHub(nil, "")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after context cancel")
	}
}

func TestHub_SubscribeToRedis_NilClient(t *testing.T) {
	hub := NewHub(nil, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	hub.SubscribeToRedis(ctx)
}

func TestNewClient_WritePump_ReadPump_Integration(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run(context.Background())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocketlib.Accept(w, r, &websocketlib.AcceptOptions{InsecureSkipVerify: true})
		require.NoError(t, err)
		client := NewClient(hub, conn)
		hub.Register(client)
		go client.WritePump()
		go client.ReadPump()
	}))
	defer srv.Close()

	wsURL := "ws" + srv.URL[4:] + "/"
	conn, _, err := websocketlib.Dial(context.Background(), wsURL, nil)
	require.NoError(t, err)
	defer conn.Close(websocketlib.StatusNormalClosure, "")

	_, data, err := conn.Read(context.Background())
	require.NoError(t, err)
	var ev Event
	require.NoError(t, json.Unmarshal(data, &ev))
	assert.Equal(t, "connected", ev.Type)
}

func TestBroadcastEvent_JsonMarshalError(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run(context.Background())

	hub.BroadcastEvent(func() {})
}

func TestHub_BroadcastEvent_LocalFallback(t *testing.T) {
	hub := NewHub(nil, "")
	go hub.Run(context.Background())

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
