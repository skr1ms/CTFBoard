package e2e_test

import (
	"net/http"
	"testing"
	"time"
)

// GET /scoreboard with freeze_time: solves after freeze are not counted in public scoreboard (frozen view).
func TestScoreboard_Freeze(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

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

	scoreboard := h.GetScoreboard().Status(http.StatusOK).JSON().Array()

	foundUser2 := false
	for _, val := range scoreboard.Iter() {
		obj := val.Object()
		if obj.Value("team_name").String().Raw() == "user_freeze_2" {
			foundUser2 = true
			points := obj.Value("points").Number().Raw()
			if points != 0 {
				t.Errorf("Scoreboard not frozen! User 2 has %v points", points)
			}
		}
	}

	if !foundUser2 {
		t.Log("User 2 not found in frozen scoreboard (acceptable behavior)")
	}
}
