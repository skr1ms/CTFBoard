package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// TestHiddenChallenge_SubmitReturnsNotFound verifies that submitting a flag to a hidden
// challenge returns 404 for a regular user.
func TestHiddenChallenge_SubmitReturnsNotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("hidden_sub_" + suffix)

	flag := "flag{hidden_submit_" + suffix + "}"
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "HiddenChallenge_" + suffix,
		"description": "hidden challenge test",
		"flag":        flag,
		"points":      100,
		"category":    "misc",
		"state":       "hidden",
	})

	_, _, tokenUser := h.RegisterUserAndLogin("hidden_sub_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// Hidden challenge is not found for regular users.
	h.SubmitFlag(tokenUser, challID, flag, http.StatusNotFound)
}

// TestChallengeVisibilityHidden_BlocksDirectActions verifies that global
// challenge_visibility=hidden protects direct actions even when a challenge row
// itself is visible and the client already knows its IDs.
func TestChallengeVisibilityHidden_BlocksDirectActions(t *testing.T) {
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("global_hidden_" + suffix)
	h.PutAdminConfig(tokenAdmin, "challenge_visibility", "hidden", "string", "desc", http.StatusOK)
	defer h.PutAdminConfig(tokenAdmin, "challenge_visibility", "private", "string", "desc", http.StatusOK)

	flag := "flag{global_hidden_" + suffix + "}"
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "GlobalHidden_" + suffix,
		"description": "global challenge_visibility hidden test",
		"flag":        flag,
		"points":      100,
		"category":    "misc",
		"state":       "visible",
	})
	hintID := h.CreateHint(tokenAdmin, challID, "global hidden hint", 0)
	require.NotEmpty(t, hintID)

	_, _, tokenUser := h.RegisterUserAndLogin("global_hidden_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challID, flag, http.StatusNotFound)
	h.UnlockHint(tokenUser, challID, hintID, http.StatusNotFound)
}

// TestHiddenChallenge_AdminReveals_ThenSubmitSucceeds verifies that once a hidden challenge
// is made visible by an admin, regular users can submit the flag successfully.
func TestHiddenChallenge_AdminReveals_ThenSubmitSucceeds(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("hidden_reveal_" + suffix)

	flag := "flag{hidden_reveal_" + suffix + "}"
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "HiddenReveal_" + suffix,
		"description": "hidden challenge reveal test",
		"flag":        flag,
		"points":      100,
		"category":    "misc",
		"state":       "hidden",
	})

	_, _, tokenUser := h.RegisterUserAndLogin("hidden_reveal_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// Still hidden - not found.
	h.SubmitFlag(tokenUser, challID, flag, http.StatusNotFound)

	// Admin reveals the challenge.
	h.UpdateChallenge(tokenAdmin, challID, map[string]any{
		"title":       "HiddenReveal_" + suffix,
		"description": "hidden challenge reveal test",
		"points":      100,
		"category":    "misc",
		"state":       "visible",
	})

	// Now visible - submit succeeds.
	h.SubmitFlag(tokenUser, challID, flag, http.StatusOK)
}
