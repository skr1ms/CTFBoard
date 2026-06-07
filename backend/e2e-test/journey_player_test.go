package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestE2E_PlayerJourney_SubmitScoreboardAndWebSocket(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("journey_admin")
	player := s.registerUser("journey_player")
	s.createSoloTeam(&player)

	challengeID := s.createChallenge(admin, "Journey submit "+e2eUID("chall"), "flag{journey_submit}", 100)

	wsURL := "ws://localhost:" + testPort + "/api/v1/ws"

	conn, resp, err := websocket.Dial(context.Background(), wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": []string{"Bearer " + player.Token}},
	})
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	require.NoError(t, err)

	defer conn.Close(websocket.StatusNormalClosure, "")

	waitWSMessage(t, conn, "connected", 5*time.Second)

	s.submitFlag(player, challengeID, "flag{wrong}", false, http.StatusOK)
	s.submitFlag(player, challengeID, "flag{journey_submit}", true, http.StatusOK)
	s.requireTeamScore(admin.Token, player.TeamName, 100)

	msg := waitWSMessage(t, conn, "scoreboard_update", 15*time.Second)

	payload, _ := msg["payload"].(map[string]any)
	if payload != nil {
		got, _ := payload["type"].(string)
		require.Contains(t, []string{"solve", "first_blood"}, got)
	}
}
