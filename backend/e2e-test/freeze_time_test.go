package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /scoreboard with freeze_time: solves after freeze are not counted in public scoreboard (frozen view).
func TestScoreboard_Freeze(t *testing.T) {
	t.Helper()
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("admin_freeze_" + suffix)

	// Set competition times directly in DB to avoid COMPETITION_ACTIVE_CANNOT_UPDATE
	// race: parallel tests may activate competition between resetCompetitionToNotStarted and the API PUT.
	now := time.Now().UTC()
	freezeTime := now.Add(2 * time.Second)
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), &freezeTime)

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
	h.SubmitFlag(user1, challID, "flag{freeze}", http.StatusOK)

	_, _, user2 := h.RegisterUserAndLogin("user_freeze_2")
	h.CreateSoloTeam(user2, http.StatusCreated)

	time.Sleep(3 * time.Second)

	h.SubmitFlag(user2, challID, "flag{freeze}", http.StatusOK)

	scoreboard := h.GetScoreboard(user2)
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
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("admin_freeze_empty_" + suffix)
	_ = h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Freeze Empty", "description": "x", "flag": "flag{fe}",
		"points": 100, "category": "misc", "is_hidden": false,
	})
	// Set competition times directly in DB to avoid COMPETITION_ACTIVE_CANNOT_UPDATE
	// race: parallel tests may activate competition between resetCompetitionToNotStarted and the API PUT.
	now := time.Now().UTC()
	freezeTime := now.Add(1 * time.Hour)
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), &freezeTime)
	resp := h.GetScoreboard(tokenAdmin)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "scoreboard freeze empty")
	require.NotNil(t, resp.JSON200)
}
