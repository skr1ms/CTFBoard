package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// GET /scoreboard: ranks and points reflect solves; team with more solves has higher rank and correct total points.
func TestScoreboard_Display(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_scoreboard")

	challengeID1 := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Challenge 1",
		"description":   "Test challenge 1",
		"points":        100,
		"flag":          "FLAG{chall1}",
		"category":      "web",
		"initial_value": 100,
		"min_value":     100,
		"decay":         1,
	})

	challengeID2 := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Challenge 2",
		"description":   "Test challenge 2",
		"points":        200,
		"flag":          "FLAG{chall2}",
		"category":      "crypto",
		"initial_value": 200,
		"min_value":     200,
		"decay":         1,
	})

	suffix := uuid.New().String()[:8]
	nameUser1 := "user4_" + suffix
	_, _, tokenUser1 := h.RegisterUserAndLogin(nameUser1)
	h.CreateSoloTeam(tokenUser1, http.StatusCreated)

	nameUser2 := "user5_" + suffix
	_, _, tokenUser2 := h.RegisterUserAndLogin(nameUser2)
	h.CreateSoloTeam(tokenUser2, http.StatusCreated)

	h.SubmitFlag(tokenUser1, challengeID1, "FLAG{chall1}", http.StatusOK)
	time.Sleep(1 * time.Second)
	h.SubmitFlag(tokenUser1, challengeID2, "FLAG{chall2}", http.StatusOK)

	time.Sleep(1 * time.Second)
	h.SubmitFlag(tokenUser2, challengeID1, "FLAG{chall1}", http.StatusOK)

	h.AssertTeamScore(nameUser1, 300)
	h.AssertTeamScore(nameUser2, 100)
}

// GET /scoreboard: returns 200 and array even when no teams/solves.
func TestScoreboard_Empty(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	resp := h.GetScoreboard()
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)
}
