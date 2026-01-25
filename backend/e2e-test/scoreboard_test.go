package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestScoreboard_Display(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_scoreboard")

	// 2. Create Multiple Challenges
	challengeID1 := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Challenge 1",
		"description": "Test challenge 1",
		"points":      100,
		"flag":        "FLAG{chall1}",
		"category":    "web",
	})

	challengeID2 := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Challenge 2",
		"description": "Test challenge 2",
		"points":      200,
		"flag":        "FLAG{chall2}",
		"category":    "crypto",
	})

	// 3. Register Two Solver Teams
	suffix := uuid.New().String()[:8]
	nameUser1 := "user4_" + suffix
	_, _, tokenUser1 := h.RegisterUserAndLogin(nameUser1)

	nameUser2 := "user5_" + suffix
	_, _, tokenUser2 := h.RegisterUserAndLogin(nameUser2)

	// 4. Team 1 Solves Both Challenges
	h.SubmitFlag(tokenUser1, challengeID1, "FLAG{chall1}", http.StatusOK)
	time.Sleep(1 * time.Second)
	h.SubmitFlag(tokenUser1, challengeID2, "FLAG{chall2}", http.StatusOK)

	// 5. Team 2 Solves Only Challenge 1
	time.Sleep(1 * time.Second)
	h.SubmitFlag(tokenUser2, challengeID1, "FLAG{chall1}", http.StatusOK)

	// 6. Verify Scoreboard Ranks and Points
	h.AssertTeamScore(nameUser1, 300)
	h.AssertTeamScore(nameUser2, 100)
}

func TestScoreboard_Empty(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Verify Scoreboard Returns Successfully Even When Empty
	h.GetScoreboard().
		Status(http.StatusOK).
		JSON().
		Array()
}
