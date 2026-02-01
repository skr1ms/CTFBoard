package e2e_test

import (
	"net/http"
	"testing"
)

// Dynamic scoring: first solver gets initial_value; second solver gets decayed score (min_value) and scoreboard reflects it.
func TestDynamicScoring_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

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
