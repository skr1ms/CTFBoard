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

	suffix := uuid.New().String()[:8]

	_, _, tokenAdmin := h.RegisterAdmin("admin6_" + suffix)

	h.StartCompetition(tokenAdmin)

	challengeID1 := h.CreateChallenge(tokenAdmin, map[string]interface{}{
		"title":       "Challenge 1",
		"description": "Test challenge 1",
		"points":      100,
		"flag":        "FLAG{chall1}",
		"category":    "web",
	})

	challengeID2 := h.CreateChallenge(tokenAdmin, map[string]interface{}{
		"title":       "Challenge 2",
		"description": "Test challenge 2",
		"points":      200,
		"flag":        "FLAG{chall2}",
		"category":    "crypto",
	})

	nameUser1 := "user4_" + suffix
	_, _, tokenUser1 := h.RegisterUserAndLogin(nameUser1)

	nameUser2 := "user5_" + suffix
	_, _, tokenUser2 := h.RegisterUserAndLogin(nameUser2)

	h.SubmitFlag(tokenUser1, challengeID1, "FLAG{chall1}", http.StatusOK)
	time.Sleep(1 * time.Second)
	h.SubmitFlag(tokenUser1, challengeID2, "FLAG{chall2}", http.StatusOK)

	time.Sleep(1 * time.Second)
	h.SubmitFlag(tokenUser2, challengeID1, "FLAG{chall1}", http.StatusOK)

	h.AssertTeamScore(nameUser1, 300)
	h.AssertTeamScore(nameUser2, 100)
}

func TestScoreboard_Empty(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	h.GetScoreboard().
		Status(http.StatusOK).
		JSON().
		Array()
}
