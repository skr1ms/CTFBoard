package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wslib "github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-wskit"
)

func startTestHub(t *testing.T) *wskit.Hub {
	t.Helper()

	hub := wskit.NewHub(wskit.WithOnConnect(func(sub wskit.Subscriber) {
		c, ok := sub.(*wskit.Client)
		if !ok {
			return
		}

		data, err := json.Marshal(wskit.NewEvent(EventTypeConnected, nil))
		if err == nil {
			c.Send(data)
		}
	}))

	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)

	t.Cleanup(cancel)

	return hub
}

func dialHubClient(t *testing.T, hub *wskit.Hub) *wslib.Conn {
	t.Helper()

	srvCtx, srvCancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := wskit.Accept(srvCtx, w, r, hub, &wslib.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}

		go c.WritePump()
		go c.ReadPump()
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(srvCancel)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, resp, err := wslib.Dial(context.Background(), wsURL, nil)
	require.NoError(t, err)

	if resp != nil && resp.Body != nil {
		t.Cleanup(func() { _ = resp.Body.Close() })
	}

	t.Cleanup(func() { _ = conn.Close(wslib.StatusNormalClosure, "") })

	return conn
}

func readEvent(t *testing.T, conn *wslib.Conn) wskit.Event {
	t.Helper()

	readCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, data, err := conn.Read(readCtx)
	require.NoError(t, err)

	var ev wskit.Event
	require.NoError(t, json.Unmarshal(data, &ev))

	return ev
}

func TestNewBroadcaster(t *testing.T) {
	t.Parallel()

	hub := wskit.NewHub()
	b := NewBroadcaster(hub)
	require.NotNil(t, b)
}

func TestBroadcaster_NotifySolve_NilBroadcaster(t *testing.T) {
	t.Parallel()

	var b *Broadcaster
	b.NotifySolve(uuid.New(), "ch", 100, false)
	b.NotifySolve(uuid.New(), "ch", 100, true)
}

func TestBroadcaster_NotifySolve_NilHub(t *testing.T) {
	t.Parallel()

	b := NewBroadcaster(nil)
	b.NotifySolve(uuid.New(), "ch", 100, false)
	b.NotifySolve(uuid.New(), "ch", 100, true)
}

func TestBroadcaster_NotifySolve_WithHub_NoFirstBlood(t *testing.T) {
	t.Parallel()
	hub := startTestHub(t)
	conn := dialHubClient(t, hub)
	ev0 := readEvent(t, conn)
	assert.Equal(t, EventTypeConnected, ev0.Type)

	teamID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	b := NewBroadcaster(hub)
	b.NotifySolve(teamID, "Challenge A", 150, false)

	assert.Eventually(t, func() bool {
		readCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_, data, err := conn.Read(readCtx)
		if err != nil {
			return false
		}

		var ev wskit.Event
		if json.Unmarshal(data, &ev) != nil {
			return false
		}

		if ev.Type != "scoreboard_update" {
			return false
		}

		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			return false
		}

		return payload["type"] == EventTypeSolve &&
			payload["team_id"] == teamID.String() &&
			payload["challenge"] == "Challenge A" &&
			payload["points"] == float64(150)
	}, 2*time.Second, 10*time.Millisecond)
}

func TestBroadcaster_NotifySolve_WithHub_FirstBlood(t *testing.T) {
	t.Parallel()
	hub := startTestHub(t)
	conn := dialHubClient(t, hub)
	_ = readEvent(t, conn)

	teamID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	b := NewBroadcaster(hub)
	b.NotifySolve(teamID, "Challenge B", 200, true)

	var solveEv, fbEv wskit.Event

	assert.Eventually(t, func() bool {
		readCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_, data, err := conn.Read(readCtx)
		if err != nil {
			return false
		}

		if json.Unmarshal(data, &solveEv) != nil {
			return false
		}

		return solveEv.Type == "scoreboard_update"
	}, 2*time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool {
		readCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_, data, err := conn.Read(readCtx)
		if err != nil {
			return false
		}

		if json.Unmarshal(data, &fbEv) != nil {
			return false
		}

		if fbEv.Type != "scoreboard_update" {
			return false
		}

		payload, ok := fbEv.Payload.(map[string]any)

		return ok && payload["type"] == EventTypeFirstBlood
	}, 2*time.Second, 10*time.Millisecond)
}

func TestBroadcaster_NotifyNotification_NilBroadcaster(t *testing.T) {
	t.Parallel()

	var b *Broadcaster
	b.NotifyNotification("msg", "info")
}

func TestBroadcaster_NotifyNotification_NilHub(t *testing.T) {
	t.Parallel()

	b := NewBroadcaster(nil)
	b.NotifyNotification("msg", "warning")
}

func TestBroadcaster_NotifyNotification_WithHub(t *testing.T) {
	t.Parallel()
	hub := startTestHub(t)
	conn := dialHubClient(t, hub)
	_ = readEvent(t, conn)

	b := NewBroadcaster(hub)
	b.NotifyNotification("Hello", "success")

	assert.Eventually(t, func() bool {
		readCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_, data, err := conn.Read(readCtx)
		if err != nil {
			return false
		}

		var ev wskit.Event
		if json.Unmarshal(data, &ev) != nil {
			return false
		}

		if ev.Type != "notification" {
			return false
		}

		payload, ok := ev.Payload.(map[string]any)

		return ok &&
			payload["type"] == EventTypeNotification &&
			payload["message"] == "Hello" &&
			payload["level"] == "success"
	}, 2*time.Second, 10*time.Millisecond)
}
