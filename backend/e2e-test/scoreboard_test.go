package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// GET /scoreboard: ranks and points reflect solves; team with more solves has higher rank and correct total points.
func TestScoreboard_Display(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_scoreboard")

	challengeID1 := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Challenge 1",
		"description":   "Test challenge 1",
		"points":        100,
		"flag":          "FLAG{chall1}",
		"category":      "web",
		"initial_value": 100,
		"min_value":     100,
		"decay":         1,
	})

	challengeID2 := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Challenge 2",
		"description":   "Test challenge 2",
		"points":        200,
		"flag":          "FLAG{chall2}",
		"category":      "crypto",
		"initial_value": 200,
		"min_value":     200,
		"decay":         1,
	})

	suffix := helper.UID()
	nameUser1 := "user4_" + suffix
	_, _, tokenUser1 := h.RegisterUserAndLogin(nameUser1)
	h.CreateSoloTeam(tokenUser1, http.StatusCreated)

	nameUser2 := "user5_" + suffix
	_, _, tokenUser2 := h.RegisterUserAndLogin(nameUser2)
	h.CreateSoloTeam(tokenUser2, http.StatusCreated)

	h.SubmitFlag(tokenUser1, challengeID1, "FLAG{chall1}", http.StatusOK)
	require.Eventually(t, func() bool { return h.TeamScoreMatches(tokenUser1, nameUser1, 100) }, 2*time.Second, 100*time.Millisecond)
	h.SubmitFlag(tokenUser1, challengeID2, "FLAG{chall2}", http.StatusOK)

	require.Eventually(t, func() bool { return h.TeamScoreMatches(tokenUser1, nameUser1, 300) }, 2*time.Second, 100*time.Millisecond)
	h.SubmitFlag(tokenUser2, challengeID1, "FLAG{chall1}", http.StatusOK)

	require.Eventually(t, func() bool { return h.TeamScoreMatches(tokenUser1, nameUser2, 100) }, 1*time.Second, 50*time.Millisecond)

	_ = TestRedis.Del(context.Background(), "scoreboard", "scoreboard:frozen")

	h.AssertTeamScore(tokenUser1, nameUser1, 300)
	h.AssertTeamScore(tokenUser2, nameUser2, 100)
}

// GET /scoreboard: returns 200 and array even when no teams/solves.
func TestScoreboard_Empty(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, token := h.SetupCompetition("admin_empty_sb")
	resp := h.GetScoreboard(token)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "scoreboard empty")
	require.NotNil(t, resp.JSON200)
}
