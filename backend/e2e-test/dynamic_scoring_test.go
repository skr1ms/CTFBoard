package e2e_test

import (
	"net/http"
	"testing"

	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// Dynamic scoring: first solver gets initial_value; second solver gets decayed score (min_value) and scoreboard reflects it.
func TestDynamicScoring_Flow(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

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
		"is_hidden":     false,
	})

	_, _, user1 := h.RegisterUserAndLogin("user_dyn_1")
	h.CreateSoloTeam(user1, http.StatusCreated)
	_, _, user2 := h.RegisterUserAndLogin("user_dyn_2")
	h.CreateSoloTeam(user2, http.StatusCreated)

	h.SubmitFlag(user1, challID, "flag{dyn}", http.StatusOK)
	h.AssertTeamScore("user_dyn_1", 500)

	h.SubmitFlag(user2, challID, "flag{dyn}", http.StatusOK)

	scoreboard := h.GetScoreboard()
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

// POST /challenges/{ID}/submit: wrong flag returns 400 Bad Request.
func TestDynamicScoring_InvalidFlag_Returns400(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())
	_, tokenAdmin := h.SetupCompetition("admin_dynamic_err")
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Dyn Err", "description": "x", "flag": "flag{dyn_err}",
		"points": 500, "initial_value": 500, "min_value": 100, "decay": 1,
		"category": "misc", "is_hidden": false,
	})
	_, _, tokenUser := h.RegisterUserAndLogin("user_dyn_err")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	resp := h.SubmitFlag(tokenUser, challID, "wrong_flag", http.StatusBadRequest)
	require.NotNil(t, resp.JSON400)
	require.NotNil(t, resp.JSON400.Error)
	require.Equal(t, "invalid flag", *resp.JSON400.Error)
}
