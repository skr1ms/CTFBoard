package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// assertScoreboardSolveMessage checks that msg is a scoreboard_update with payload type solve or first_blood.
func assertScoreboardSolveMessage(t *testing.T, msg map[string]any) {
	t.Helper()
	typ, ok := msg["type"].(string)
	if !ok || typ != "scoreboard_update" {
		t.Errorf("expected type scoreboard_update, got %q (full message: %+v)", typ, msg)
		return
	}
	payload, ok := msg["payload"].(map[string]any)
	if ok && payload != nil {
		payloadType, ok := payload["type"].(string)
		if !ok || (payloadType != "solve" && payloadType != "first_blood") {
			t.Errorf("expected payload.type solve or first_blood, got %q", payloadType)
		}
	}
}

// startWSReader starts a goroutine that reads WebSocket messages into a channel until timeout or close.
func startWSReader(conn *websocket.Conn, readTimeout time.Duration) (received <-chan map[string]any, readErr <-chan error, done <-chan struct{}) {
	rec := make(chan map[string]any, 4)
	errCh := make(chan error, 1)
	d := make(chan struct{})
	go func() {
		defer close(d)
		readCtx, readCancel := context.WithTimeout(context.Background(), readTimeout)
		defer readCancel()
		runWSReadLoop(conn, readCtx, rec, errCh)
	}()
	return rec, errCh, d
}

// runWSReadLoop reads JSON messages from conn and sends them to rec until context is done or read error.
func runWSReadLoop(conn *websocket.Conn, readCtx context.Context, rec chan<- map[string]any, errCh chan<- error) {
	for {
		_, data, err := conn.Read(readCtx)
		if err != nil {
			select {
			case errCh <- err:
			default:
			}
			return
		}
		var msg map[string]any
		if json.Unmarshal(data, &msg) == nil {
			rec <- msg
		}
	}
}

const (
	wsConnectedTimeout = 3 * time.Second
	wsReceiveTimeout   = 15 * time.Second
)

// waitWSConnected blocks until a "connected" message is received or timeout/error.
func waitWSConnected(t *testing.T, received <-chan map[string]any, readErr <-chan error, done <-chan struct{}) {
	t.Helper()
	deadline := time.After(wsConnectedTimeout)
	for {
		select {
		case msg := <-received:
			typ, ok := msg["type"].(string)
			t.Logf("ws received message type=%q", typ)
			if ok && typ == "connected" {
				return
			}
		case err := <-readErr:
			t.Fatalf("ws read failed while waiting for connected: %v", err)
		case <-done:
			t.Fatal("ws reader exited before receiving connected")
		case <-deadline:
			t.Fatal("timeout: no connected message (hub may not be sending to this client)")
		}
	}
}

// waitScoreboardUpdate blocks until a scoreboard_update message is received or timeout/error.
func waitScoreboardUpdate(t *testing.T, received <-chan map[string]any, readErr <-chan error, done <-chan struct{}) {
	t.Helper()
	deadline := time.After(wsReceiveTimeout)
	for {
		select {
		case msg := <-received:
			typ, ok := msg["type"].(string)
			t.Logf("ws received message type=%q", typ)
			if ok && typ == "scoreboard_update" {
				assertScoreboardSolveMessage(t, msg)
				return
			}
		case err := <-readErr:
			t.Fatalf("ws read failed: %v", err)
		case <-done:
			t.Fatal("ws reader exited before receiving scoreboard_update")
		case <-deadline:
			t.Fatalf("timeout: no scoreboard_update message in %v (event may not be published or hub not delivering)", wsReceiveTimeout)
		}
	}
}

// GET /ws: client receives scoreboard_update event when a solve is submitted.
func TestWebSocket_ReceiveSolveEvent(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_ws")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "WS Chall", "flag{ws_event}", 100)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("wsuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	wsURL := "ws://localhost:" + testPort + "/api/v1/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialOpts := &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": []string{tokenUser}},
	}
	conn, resp, err := websocket.Dial(ctx, wsURL, dialOpts)
	if err != nil {
		t.Fatalf("ws dial failed (url=%s): %v", wsURL, err)
	}
	t.Logf("ws dial ok url=%s", wsURL)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	received, readErr, done := startWSReader(conn, wsReceiveTimeout+5*time.Second)
	t.Logf("ws waiting for connected")
	waitWSConnected(t, received, readErr, done)
	t.Logf("ws connected, submitting flag challengeID=%s", challengeID)
	h.SubmitFlag(tokenUser, challengeID, "flag{ws_event}", http.StatusOK)
	t.Logf("ws submit done, waiting for scoreboard_update")
	waitScoreboardUpdate(t, received, readErr, done)
}

// GET /ws on invalid path: connection fails or returns 404.
func TestWebSocket_InvalidPath_NotFound(t *testing.T) {
	t.Parallel()
	_, _ = helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL()), TestPool
	wsURL := "ws://localhost:" + testPort + "/api/v1/ws-invalid"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return
	}
	if conn != nil {
		conn.Close(websocket.StatusNormalClosure, "")
	}
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	if status == http.StatusNotFound {
		return
	}
	t.Fatalf("expected 404 or dial error for invalid path, got status=%d err=%v", status, err)
}

// GET /ws: authenticated user connects and receives a "connected" message.
func TestWebSocket_Connect_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("admin_ws_connect")
	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("ws_connect_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	wsURL := "ws://localhost:" + testPort + "/api/v1/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialOpts := &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": []string{tokenUser}},
	}
	conn, resp, err := websocket.Dial(ctx, wsURL, dialOpts)
	if err != nil {
		t.Fatalf("ws connect failed (url=%s): %v", wsURL, err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	received, readErr, done := startWSReader(conn, wsConnectedTimeout+2*time.Second)
	waitWSConnected(t, received, readErr, done)
}

// GET /ws: no auth -> upgrade refused with 401.
func TestWebSocket_Connect_Unauthorized(t *testing.T) {
	t.Parallel()
	_, _ = helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL()), TestPool

	wsURL := "ws://localhost:" + testPort + "/api/v1/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if conn != nil {
		conn.Close(websocket.StatusNormalClosure, "")
	}
	if err != nil {
		return
	}
	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		return
	}
	t.Fatalf("expected dial error or 401 for unauthenticated ws connect, got status=%v err=%v", func() int {
		if resp != nil {
			return resp.StatusCode
		}
		return 0
	}(), err)
}
