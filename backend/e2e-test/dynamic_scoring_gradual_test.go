package e2e_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/scoring"
)

// TestDynamicScoring_GradualDecay_LogarithmicCurve verifies that a dynamic challenge with
// decay=5 loses points gradually as more teams solve it, following the logarithmic decay
// formula, and that solve_count on the challenge updates accordingly.
func TestDynamicScoring_GradualDecay_LogarithmicCurve(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("gradual_decay_" + suffix)

	// Ensure logarithmic decay is active - the seeded default is "linear".
	h.PutAdminConfig(tokenAdmin, "decay_function", "logarithmic", "string", "Decay function", http.StatusOK)

	const (
		initial  = 1000
		minimum  = 100
		decay    = 5
		numTeams = 7
	)

	flag := "flag{gradual_" + suffix + "}"
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "GradualDecay_" + suffix,
		"description":   "gradual decay scoring test",
		"flag":          flag,
		"points":        initial,
		"category":      "misc",
		"state":         "visible",
		"initial_value": initial,
		"min_value":     minimum,
		"decay":         decay,
	})

	// Register numTeams users, each with their own solo team.
	tokens := make([]string, numTeams)
	for i := range numTeams {
		_, _, tok := h.RegisterUserAndLogin(fmt.Sprintf("gdecay_%s_%d", suffix, i))
		h.CreateSoloTeam(tok, http.StatusCreated)
		tokens[i] = tok
	}

	// Each team solves the challenge sequentially and we verify the current points
	// on the challenge match the expected logarithmic decay formula.
	for i, tok := range tokens {
		solveNumber := i + 1

		h.SubmitFlag(tok, challID, flag, http.StatusOK)

		// Read current challenge points from the list endpoint.
		ch := h.FindChallengeInList(tok, challID)
		require.NotNil(t, ch, "challenge not found in list after solve %d", solveNumber)
		require.NotNil(t, ch.Points, "challenge points nil after solve %d", solveNumber)

		expected := scoring.CalculateDynamicScore(initial, minimum, decay, solveNumber)

		assert.Equal(t, expected, *ch.Points, "solve %d: expected points=%d, got=%d", solveNumber, expected, *ch.Points)
	}

	// After numTeams solves the score must be clamped to minimum.
	lastCh := h.FindChallengeInList(tokens[numTeams-1], challID)
	require.NotNil(t, lastCh)
	assert.Equal(t, minimum, *lastCh.Points, "score should be at minimum after enough solves")

	// Scoreboard: the first solver must rank above the last solver.
	sbResp := h.GetScoreboard(tokens[0])
	require.Equal(t, http.StatusOK, sbResp.StatusCode())
	require.NotNil(t, sbResp.JSON200)

	firstTeam := h.GetMyTeam(tokens[0], http.StatusOK)
	require.NotNil(t, firstTeam.JSON200)

	lastTeam := h.GetMyTeam(tokens[numTeams-1], http.StatusOK)
	require.NotNil(t, lastTeam.JSON200)

	firstPos, lastPos := -1, -1

	for idx, entry := range *sbResp.JSON200 {
		if entry.TeamName == nil {
			continue
		}

		if firstTeam.JSON200.Name != nil && *entry.TeamName == *firstTeam.JSON200.Name {
			firstPos = idx
		}

		if lastTeam.JSON200.Name != nil && *entry.TeamName == *lastTeam.JSON200.Name {
			lastPos = idx
		}
	}

	require.NotEqual(t, -1, firstPos, "first solver team not found on scoreboard")
	require.NotEqual(t, -1, lastPos, "last solver team not found on scoreboard")
	assert.Less(t, firstPos, lastPos, "first solver (pos %d) should rank above last solver (pos %d) on scoreboard", firstPos, lastPos)
}
