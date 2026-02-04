package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /challenges/{ID}/first-blood: first solver is credited as first blood; response contains username/team.
func TestFirstBlood_Display(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("adminfb")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "First Blood Test",
		"description": "Test first blood functionality",
		"flag":        "FLAG{firstblood}",
		"points":      100,
		"category":    "web",
		"is_hidden":   false,
	})

	_, _, tokenUser1 := h.RegisterUserAndLogin("fbuser1")
	h.CreateSoloTeam(tokenUser1, http.StatusCreated)
	_, _, tokenUser2 := h.RegisterUserAndLogin("fbuser2")
	h.CreateSoloTeam(tokenUser2, http.StatusCreated)

	h.SubmitFlag(tokenUser1, challengeID, "FLAG{firstblood}", http.StatusOK)

	time.Sleep(1 * time.Second)

	h.SubmitFlag(tokenUser2, challengeID, "FLAG{firstblood}", http.StatusOK)

	h.AssertFirstBlood(challengeID, "fbuser1", "fbuser1")
}

// GET /challenges/{ID}/first-blood: unsolved challenge returns 404 with "solve not found".
func TestFirstBlood_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("adminfb2")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "No Solves Test",
		"description": "Test no solves scenario",
		"flag":        "FLAG{nosolves}",
		"category":    "misc",
		"points":      100,
	})

	resp := h.GetFirstBlood(challengeID, http.StatusNotFound)
	require.NotNil(t, resp.JSON404)
	require.NotNil(t, resp.JSON404.Error)
	require.Equal(t, "solve not found", *resp.JSON404.Error)
}

// GET /challenges/{ID}/first-blood: invalid challenge ID format returns 400.
func TestFirstBlood_InvalidID(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())
	_, _ = h.SetupCompetition("adminfb3")
	h.GetFirstBlood("not-a-uuid", http.StatusBadRequest)
}
