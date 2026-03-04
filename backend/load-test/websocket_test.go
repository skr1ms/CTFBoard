package load_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

const (
	wsEnduranceConns         = 200
	wsEnduranceDuration      = 5 * time.Minute
	wsEnduranceDurationShort = 60 * time.Second
	wsConnectTimeout         = 10 * time.Second
	wsAliveThreshold         = 0.90
	wsEventInterval          = 3 * time.Second
)

type wsConnState struct {
	connected  bool
	startedAt  time.Time
	messagesRx int
	reconnects int
	lastErr    error
}

func TestWebSocket_Endurance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping WebSocket endurance test in short mode")
	}
	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)
	require.NotEmpty(t, Fixture.ChallengeIDs)

	duration := wsEnduranceDuration
	runWSEndurance(t, duration)
}

func TestWebSocket_Endurance_Short(t *testing.T) {
	require.NotNil(t, Fixture)
	require.NotEmpty(t, Fixture.Users)
	require.NotEmpty(t, Fixture.ChallengeIDs)

	runWSEndurance(t, wsEnduranceDurationShort)
}

func runWSEndurance(t *testing.T, duration time.Duration) {
	t.Helper()

	wsURL := strings.Replace(Fixture.BaseURL, "http://", "ws://", 1) + "/api/v1/ws"
	numConns := wsEnduranceConns
	if numConns > len(Fixture.Users) {
		numConns = len(Fixture.Users)
	}

	var (
		mu           sync.Mutex
		states       = make([]*wsConnState, numConns)
		totalMsgsRx  atomic.Int64
		disconnected atomic.Int64
	)

	ctx, cancel := context.WithTimeout(context.Background(), duration+30*time.Second)
	defer cancel()

	var connectWg sync.WaitGroup
	for i := range numConns {
		connectWg.Add(1)
		go func(idx int) {
			defer connectWg.Done()
			state := &wsConnState{startedAt: time.Now()}
			mu.Lock()
			states[idx] = state
			mu.Unlock()

			token := Fixture.Users[idx].Token
			dialCtx, dialCancel := context.WithTimeout(ctx, wsConnectTimeout)
			defer dialCancel()

			conn, resp, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
				HTTPHeader: http.Header{"Authorization": []string{token}},
			})
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			if err != nil {
				state.lastErr = err
				disconnected.Add(1)
				return
			}

			state.connected = true
			go runWSClientLoop(ctx, conn, state, &totalMsgsRx, &disconnected, duration)
		}(i)
	}
	connectWg.Wait()

	aliveAfterConnect := numConns - int(disconnected.Load())
	fmt.Printf("\n[ws-endurance] connected %d/%d clients\n", aliveAfterConnect, numConns)
	require.Greater(t, aliveAfterConnect, 0, "no WebSocket connections established")

	eventCtx, eventCancel := context.WithCancel(ctx)
	defer eventCancel()
	go generateWSEvents(eventCtx, Fixture, wsEventInterval)

	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}
	eventCancel()

	aliveAtEnd := aliveAfterConnect - int(disconnected.Load())
	fmt.Printf("[ws-endurance] duration=%s  alive=%d/%d  msgs_rx=%d  reconnects=%d\n",
		duration,
		aliveAtEnd,
		aliveAfterConnect,
		totalMsgsRx.Load(),
		countReconnects(states),
	)

	ratio := float64(aliveAtEnd) / float64(aliveAfterConnect)
	require.GreaterOrEqual(t, ratio, wsAliveThreshold,
		"WebSocket endurance: alive ratio %.2f%% < threshold %.0f%%",
		ratio*100, wsAliveThreshold*100)
}

func runWSClientLoop(
	ctx context.Context,
	conn *websocket.Conn,
	state *wsConnState,
	totalMsgsRx *atomic.Int64,
	disconnected *atomic.Int64,
	duration time.Duration,
) {
	endAt := time.Now().Add(duration)
	defer conn.Close(websocket.StatusNormalClosure, "")

	for {
		if time.Now().After(endAt) {
			return
		}

		readCtx, readCancel := context.WithDeadline(ctx, endAt)
		_, data, err := conn.Read(readCtx)
		readCancel()
		if err != nil {
			if !isWSEnduranceDone(ctx, endAt) {
				state.lastErr = err
				state.connected = false
				disconnected.Add(1)
			}
			return
		}

		state.messagesRx++
		totalMsgsRx.Add(1)

		var msg map[string]any
		if json.Unmarshal(data, &msg) == nil {
			if t, _ := msg["type"].(string); t == "ping" { //nolint:errcheck // type assertion bool intentionally discarded
				_ = t
			}
		}
	}
}

func isWSEnduranceDone(ctx context.Context, endAt time.Time) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return time.Now().After(endAt)
}

func generateWSEvents(ctx context.Context, fixture *TestFixture, interval time.Duration) {
	if len(fixture.ChallengeIDs) == 0 || len(fixture.Users) == 0 {
		return
	}
	chalID := fixture.ChallengeIDs[0]
	client := &http.Client{Timeout: 5 * time.Second}

	tick := time.NewTicker(interval)
	defer tick.Stop()

	var i int
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			token := fixture.Users[i%len(fixture.Users)].Token
			sendWrongFlag(ctx, client, fixture.BaseURL, token, chalID)
			i++
		}
	}
}

func sendWrongFlag(ctx context.Context, client *http.Client, baseURL, token, challengeID string) {
	url := fmt.Sprintf("%s/api/v1/challenges/%s/submit", baseURL, challengeID)
	body := `{"flag":"ws_endurance_wrong_flag_` + fmt.Sprintf("%d", time.Now().UnixNano()) + `"}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req) //nolint:gosec // test-only: URL is constructed from test config
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

func countReconnects(states []*wsConnState) int {
	total := 0
	for _, s := range states {
		if s != nil {
			total += s.reconnects
		}
	}
	return total
}
