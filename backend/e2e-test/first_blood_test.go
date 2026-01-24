package e2e_test

import (
	"net/http"
	"testing"
	"time"
)

func TestFirstBlood_Display(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, _, tokenAdmin := h.RegisterAdmin("adminfb")
	h.StartCompetition(tokenAdmin)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "First Blood Test",
		"description": "Test first blood functionality",
		"flag":        "FLAG{firstblood}",
		"points":      100,
		"category":    "web",
		"is_hidden":   false,
	})

	_, _, tokenUser1 := h.RegisterUserAndLogin("fbuser1")
	_, _, tokenUser2 := h.RegisterUserAndLogin("fbuser2")

	h.SubmitFlag(tokenUser1, challengeID, "FLAG{firstblood}", http.StatusOK)

	time.Sleep(1 * time.Second)

	h.SubmitFlag(tokenUser2, challengeID, "FLAG{firstblood}", http.StatusOK)

	h.AssertFirstBlood(challengeID, "fbuser1")
}

func TestFirstBlood_NotFound(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, _, tokenAdmin := h.RegisterAdmin("adminfb2")
	h.StartCompetition(tokenAdmin)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "No Solves Test",
		"description": "Test no solves scenario",
		"flag":        "FLAG{nosolves}",
		"category":    "misc",
		"points":      100,
	})

	h.GetFirstBlood(challengeID, http.StatusNotFound).
		Value("error").String().IsEqual("no solves yet")
}
