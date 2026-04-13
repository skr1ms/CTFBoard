package e2e_test

import (
	"net/http"
	"testing"

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
