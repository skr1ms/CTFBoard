package e2e_test

import (
	"net/http"
	"testing"
	"time"
)

func TestScoreboard_Freeze(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_freeze")

	// 2. Create Challenge
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Freeze Chall",
		"description": "Test freeze functionality",
		"flag":        "flag{freeze}",
		"points":      100,
		"category":    "misc",
		"is_hidden":   false,
	})

	// 3. User 1
	_, _, user1 := h.RegisterUserAndLogin("user_freeze_1")
	h.CreateSoloTeam(user1, http.StatusCreated)

	// 4. Set Freeze Time to NOW + 2 sec
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

	// 5. User 1 Solves BEFORE freeze
	h.SubmitFlag(user1, challID, "flag{freeze}", http.StatusOK)

	// 6. User 2
	_, _, user2 := h.RegisterUserAndLogin("user_freeze_2")
	h.CreateSoloTeam(user2, http.StatusCreated)

	// 7. Wait for freeze time to pass
	time.Sleep(3 * time.Second)

	// 8. User 2 Solves AFTER freeze
	h.SubmitFlag(user2, challID, "flag{freeze}", http.StatusOK)

	// 9. Verify Scoreboard - Should NOT show User 2's points (Frozen)
	scoreboard := h.GetScoreboard().Status(http.StatusOK).JSON().Array()

	foundUser2 := false
	for _, val := range scoreboard.Iter() {
		obj := val.Object()
		if obj.Value("team_name").String().Raw() == "user_freeze_2" {
			foundUser2 = true
			// User 2 should have 0 points in frozen scoreboard
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
