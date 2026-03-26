package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// GET /challenges/{ID}/first-blood: first solver is credited as first blood; response contains username/team.
func TestFirstBlood_Display(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resetCompetitionToActive()

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("adminfb_" + suffix)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "First Blood Test",
		"description": "Test first blood functionality",
		"flag":        "FLAG{firstblood}",
		"points":      100,
		"category":    "web",
		"state":       "visible",
	})

	_, _, tokenUser1 := h.RegisterUserAndLogin("fbuser1_" + suffix)
	h.CreateSoloTeam(tokenUser1, http.StatusCreated)
	_, _, tokenUser2 := h.RegisterUserAndLogin("fbuser2_" + suffix)
	h.CreateSoloTeam(tokenUser2, http.StatusCreated)

	h.SubmitFlag(tokenUser1, challengeID, "FLAG{firstblood}", http.StatusOK)

	require.Eventually(t, func() bool { return h.FirstBloodAvailable(tokenUser1, challengeID) }, 2*time.Second, 100*time.Millisecond)

	h.SubmitFlag(tokenUser2, challengeID, "FLAG{firstblood}", http.StatusOK)

	h.AssertFirstBlood(tokenUser1, challengeID, "fbuser1_"+suffix, "fbuser1_"+suffix)
}

// GET /challenges/{ID}/first-blood: unsolved challenge returns 404 with "solve not found".
func TestFirstBlood_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("adminfb2")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "No Solves Test",
		"description": "Test no solves scenario",
		"flag":        "FLAG{nosolves}",
		"category":    "misc",
		"points":      100,
	})

	resp := h.GetFirstBlood(tokenAdmin, challengeID, http.StatusNotFound)
	require.NotNil(t, resp.JSON404)
	require.NotEmpty(t, resp.JSON404.Message)
	require.Equal(t, "solve not found", resp.JSON404.Message)
}

// GET /challenges/{ID}/first-blood: invalid challenge ID format returns 400.
func TestFirstBlood_InvalidID(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	_, token := h.SetupCompetition("adminfb3")
	h.GetFirstBlood(token, "not-a-uuid", http.StatusBadRequest)
}
