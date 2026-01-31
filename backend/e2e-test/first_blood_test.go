package e2e_test

import (
	"net/http"
	"testing"
	"time"
)

func TestFirstBlood_Display(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("adminfb")

	// 2. Create Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "First Blood Test",
		"description": "Test first blood functionality",
		"flag":        "FLAG{firstblood}",
		"points":      100,
		"category":    "web",
		"is_hidden":   false,
	})

	// 3. Register Two Potential Solvers
	_, _, tokenUser1 := h.RegisterUserAndLogin("fbuser1")
	h.CreateSoloTeam(tokenUser1, http.StatusCreated)
	_, _, tokenUser2 := h.RegisterUserAndLogin("fbuser2")
	h.CreateSoloTeam(tokenUser2, http.StatusCreated)

	// 4. User 1 Submits Flag First
	h.SubmitFlag(tokenUser1, challengeID, "FLAG{firstblood}", http.StatusOK)

	time.Sleep(1 * time.Second)

	// 5. User 2 Submits Flag Second
	h.SubmitFlag(tokenUser2, challengeID, "FLAG{firstblood}", http.StatusOK)

	// 6. Verify User 1 is Credited with First Blood
	h.AssertFirstBlood(challengeID, "fbuser1")
}

func TestFirstBlood_NotFound(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("adminfb2")

	// 2. Create Unsolved Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "No Solves Test",
		"description": "Test no solves scenario",
		"flag":        "FLAG{nosolves}",
		"category":    "misc",
		"points":      100,
	})

	// 3. Verify 404 on First Blood Query
	h.GetFirstBlood(challengeID, http.StatusNotFound).
		Value("error").String().IsEqual("no solves yet")
}
