package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// TestCompetition_NotStarted_BlocksSubmit verifies that flag submission is rejected when the
// competition start_time is in the future, and succeeds once it is moved to the past.
// Mutates global competition times - must not run in parallel.
func TestCompetition_NotStarted_BlocksSubmit(t *testing.T) {
	t.Cleanup(resetCompetitionToActive)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenAdmin := h.RegisterAdmin("notstarted_admin_" + suffix)

	// Set start_time 1 hour in the future so the competition is not_started.
	now := time.Now().UTC()
	setCompetitionTimes(now.Add(1*time.Hour), now.Add(25*time.Hour), nil)

	flag := "flag{not_started_" + suffix + "}"
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "NotStarted_" + suffix,
		"description": "x",
		"flag":        flag,
		"points":      100,
		"category":    "misc",
		"state":       "visible",
	})

	_, _, tokenUser := h.RegisterUserAndLogin("notstarted_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// Competition has not started - submit must be blocked.
	h.SubmitFlag(tokenUser, challID, flag, http.StatusForbidden)

	// Move start_time to the past to activate the competition.
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), nil)

	// Competition is now active - submit must succeed.
	h.SubmitFlag(tokenUser, challID, flag, http.StatusOK)
}
