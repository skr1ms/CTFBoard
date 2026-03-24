package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// POST /challenges/{ID}/submit (dynamic scoring): first solver gets initial_value; second solver gets decayed score; GET /scoreboard reflects correct points.
func TestDynamicScoring_Flow(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_dynamic")

	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Dynamic Chall",
		"description":   "Points drop fast",
		"flag":          "flag{dyn}",
		"points":        500,
		"initial_value": 500,
		"min_value":     100,
		"decay":         1,
		"category":      "misc",
		"state":         "visible",
	})

	_, _, user1 := h.RegisterUserAndLogin("user_dyn_1")
	h.CreateSoloTeam(user1, http.StatusCreated)
	_, _, user2 := h.RegisterUserAndLogin("user_dyn_2")
	h.CreateSoloTeam(user2, http.StatusCreated)

	h.SubmitFlag(user1, challID, "flag{dyn}", http.StatusOK)
	h.AssertTeamScore(user1, "user_dyn_1", 500)

	h.SubmitFlag(user2, challID, "flag{dyn}", http.StatusOK)

	scoreboard := h.GetScoreboard(user1)
	helper.RequireStatus(t, http.StatusOK, scoreboard.StatusCode(), scoreboard.Body, "scoreboard dynamic")
	require.NotNil(t, scoreboard.JSON200)
	var user2Points int
	for _, entry := range *scoreboard.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == "user_dyn_2" {
			if entry.Points != nil {
				user2Points = *entry.Points
			}
			break
		}
	}
	require.Equal(t, 100, user2Points, "Dynamic scoring: user2 should get 100 points")
}

// POST /challenges/{ID}/submit: wrong flag returns 200 with correct=false.
func TestDynamicScoring_InvalidFlag_Returns200(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	_, tokenAdmin := h.SetupCompetition("admin_dynamic_err")
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Dyn Err", "description": "x", "flag": "flag{dyn_err}",
		"points": 500, "initial_value": 500, "min_value": 100, "decay": 1,
		"category": "misc", "state": "visible",
	})
	_, _, tokenUser := h.RegisterUserAndLogin("user_dyn_err")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	resp := h.SubmitFlag(tokenUser, challID, "wrong_flag", http.StatusOK)
	require.Contains(t, string(resp.Body), "incorrect flag")
}
