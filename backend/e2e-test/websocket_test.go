package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

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

func waitWSConnected(t *testing.T, received <-chan map[string]any, readErr <-chan error, done <-chan struct{}) {
	t.Helper()
	deadline := time.After(wsConnectedTimeout)
	for {
		select {
		case msg := <-received:
			typ, _ := msg["type"].(string)
			t.Logf("ws received message type=%q", typ)
			if typ == "connected" {
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

func waitScoreboardUpdate(t *testing.T, received <-chan map[string]any, readErr <-chan error, done <-chan struct{}) {
	t.Helper()
	deadline := time.After(wsReceiveTimeout)
	for {
		select {
		case msg := <-received:
			typ, _ := msg["type"].(string)
			t.Logf("ws received message type=%q", typ)
			if typ == "scoreboard_update" {
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

func TestWebSocket_ReceiveSolveEvent(t *testing.T) {
	h := NewE2EHelper(t, setupE2E(t), TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_ws")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "WS Chall", "flag{ws_event}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("wsuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	wsURL := "ws://localhost:" + testPort + "/api/v1/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
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
