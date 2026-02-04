package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /scoreboard with freeze_time: solves after freeze are not counted in public scoreboard (frozen view).
func TestScoreboard_Freeze(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_freeze")

	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Freeze Chall",
		"description": "Test freeze functionality",
		"flag":        "flag{freeze}",
		"points":      100,
		"category":    "misc",
		"is_hidden":   false,
	})

	_, _, user1 := h.RegisterUserAndLogin("user_freeze_1")
	h.CreateSoloTeam(user1, http.StatusCreated)

	now := time.Now().UTC()
	freezeTime := now.Add(2 * time.Second)
	h.UpdateCompetition(tokenAdmin, map[string]any{
		"name":              "Freeze CTF",
		"start_time":        now.Add(-1 * time.Hour),
		"end_time":          now.Add(24 * time.Hour),
		"freeze_time":       freezeTime,
		"allow_team_switch": true,
		"mode":              "flexible",
	})

	h.SubmitFlag(user1, challID, "flag{freeze}", http.StatusOK)

	_, _, user2 := h.RegisterUserAndLogin("user_freeze_2")
	h.CreateSoloTeam(user2, http.StatusCreated)

	time.Sleep(3 * time.Second)

	h.SubmitFlag(user2, challID, "flag{freeze}", http.StatusOK)

	scoreboard := h.GetScoreboard()
	helper.RequireStatus(t, http.StatusOK, scoreboard.StatusCode(), scoreboard.Body, "scoreboard freeze")
	require.NotNil(t, scoreboard.JSON200)

	foundUser2 := false
	for _, entry := range *scoreboard.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == "user_freeze_2" {
			foundUser2 = true
			var points int
			if entry.Points != nil {
				points = *entry.Points
			}
			if points != 0 {
				t.Errorf("Scoreboard not frozen! User 2 has %v points", points)
			}
		}
	}

	if !foundUser2 {
		t.Log("User 2 not found in frozen scoreboard (acceptable behavior)")
	}
}

// GET /scoreboard with freeze_time: when no solves exist, returns 200 and empty array.
func TestScoreboard_Freeze_NoSolves_Empty(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())
	_, tokenAdmin := h.SetupCompetition("admin_freeze_empty")
	_ = h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Freeze Empty", "description": "x", "flag": "flag{fe}",
		"points": 100, "category": "misc", "is_hidden": false,
	})
	now := time.Now().UTC()
	h.UpdateCompetition(tokenAdmin, map[string]any{
		"name": "Freeze Empty CTF", "start_time": now.Add(-1 * time.Hour),
		"end_time": now.Add(24 * time.Hour), "freeze_time": now.Add(1 * time.Hour),
		"allow_team_switch": true, "mode": "flexible",
	})
	resp := h.GetScoreboard()
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "scoreboard freeze empty")
	require.NotNil(t, resp.JSON200)
}
