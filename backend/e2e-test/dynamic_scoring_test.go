package e2e_test

import (
	"net/http"
	"testing"
)

func TestDynamicScoring_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_dynamic")

	// 2. Create Dynamic Challenge
	// Initial: 500, Min: 100, Decay: 1 (rapid decay)
	// Formula: Initial + (Min - Initial) / (Decay^2) * (solve_count^2)
	// where solve_count = solves - 1
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

	// 3. Register 2 Users
	_, _, user1 := h.RegisterUserAndLogin("user_dyn_1")
	_, _, user2 := h.RegisterUserAndLogin("user_dyn_2")

	// 4. First Solve (First Blood) - Should get 500 (Initial)
	// CTFd Logic: solves=1, solve_count=0. Score = Initial.
	h.SubmitFlag(user1, challID, "flag{dyn}", http.StatusOK)
	h.AssertTeamScore("user_dyn_1", 500)

	// 5. Verify Challenge Points Updated
	// Decay = 1.
	// If User 2 solves: solves = 2. solve_count = 1.
	// Formula: 500 + (100-500)/(1^2) * 1^2 = 500 - 400 = 100.

	h.SubmitFlag(user2, challID, "flag{dyn}", http.StatusOK)

	// 5. Verify User 2 Score
	scoreboard := h.GetScoreboard().Status(http.StatusOK).JSON().Array()

	var user2Points float64
	for _, val := range scoreboard.Iter() {
		obj := val.Object()
		if obj.Value("team_name").String().Raw() == "user_dyn_2" {
			user2Points = obj.Value("points").Number().Raw()
		}
	}

	if user2Points != 100 {
		t.Fatalf("Dynamic scoring failed, user2 got %v points (expected 100)", user2Points)
	}
}
