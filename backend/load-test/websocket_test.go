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
	wsEventInterval          = 1 * time.Second
	wsMinMsgDeliveryRatio    = 0.90
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

	duration := wsEnduranceDurationShort

	if testing.Short() {
		duration = 15 * time.Second
	}

	runWSEndurance(t, duration)
}

func runWSEndurance(t *testing.T, duration time.Duration) {
	t.Helper()

	wsURL := strings.Replace(Fixture.BaseURL, "http://", "ws://", 1) + "/api/v1/ws"

	numConns := min(wsEnduranceConns, len(Fixture.Users))

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
	require.Positive(t, aliveAfterConnect, "no WebSocket connections established")

	eventCtx, eventCancel := context.WithCancel(ctx)
	defer eventCancel()

	var notifsSent atomic.Int64

	go broadcastNotifications(eventCtx, Fixture, wsEventInterval, &notifsSent)

	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}

	eventCancel()

	aliveAtEnd := aliveAfterConnect - int(disconnected.Load())
	totalMsgs := totalMsgsRx.Load()
	totalNotifs := notifsSent.Load()

	fmt.Printf("[ws-endurance] duration=%s  alive=%d/%d  msgs_rx=%d  notifs_sent=%d  reconnects=%d\n",
		duration,
		aliveAtEnd,
		aliveAfterConnect,
		totalMsgs,
		totalNotifs,
		countReconnects(states),
	)

	ratio := float64(aliveAtEnd) / float64(aliveAfterConnect)
	require.GreaterOrEqual(t, ratio, wsAliveThreshold,
		"WebSocket endurance: alive ratio %.2f%% < threshold %.0f%%",
		ratio*100, wsAliveThreshold*100)

	if totalNotifs > 0 && aliveAfterConnect > 0 {
		expectedMsgs := int64(aliveAfterConnect) * totalNotifs
		minExpected := int64(float64(expectedMsgs) * wsMinMsgDeliveryRatio)
		require.GreaterOrEqual(t, totalMsgs, minExpected,
			"WebSocket message delivery: got %d msgs, expected ≥%d (%.0f%% of %d conns × %d notifs)",
			totalMsgs, minExpected, wsMinMsgDeliveryRatio*100, aliveAfterConnect, totalNotifs)
	}
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
			if t, _ := msg["type"].(string); t == "ping" {
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

// broadcastNotifications sends global notifications via the admin API, which triggers
// EventTypeNotification broadcasts to all connected WebSocket clients.
func broadcastNotifications(ctx context.Context, fixture *TestFixture, interval time.Duration, sent *atomic.Int64) {
	if fixture.AdminToken == "" {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	url := fixture.BaseURL + "/api/v1/admin/notifications"

	tick := time.NewTicker(interval)
	defer tick.Stop()

	var seq int

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			seq++
			body := fmt.Sprintf(`{"title":"ws-endurance-%d","content":"broadcast event %d","type":"info"}`, seq, seq)

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
			if err != nil {
				continue
			}

			req.Header.Set("Authorization", fixture.AdminToken)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			resp.Body.Close()

			if resp.StatusCode == http.StatusCreated {
				sent.Add(1)
			}
		}
	}
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
