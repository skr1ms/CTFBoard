package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// TestSettings_RegistrationClosed_BlocksNewUser verifies that setting registration_open=false
// prevents new users from registering, and re-enabling it allows registration again.
// Mutates global app_settings - must not run in parallel.
func TestSettings_RegistrationClosed_BlocksNewUser(t *testing.T) {
	t.Cleanup(resetAppSettingsFull)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("reg_closed_" + suffix)
	_ = tokenAdmin

	// Close registration directly via DB (API blocks changes while competition is active).
	setAppSettingsRegistrationOpen(false)

	// Attempt to register a new user - must be rejected.
	h.RegisterExpectStatus(
		"regclosed_user_"+suffix,
		"regclosed_"+suffix+"@example.com",
		"ValidPass1",
		http.StatusForbidden,
	)

	// Re-open registration.
	setAppSettingsRegistrationOpen(true)

	// Registration now succeeds.
	h.Register(
		"regclosed_user2_"+suffix,
		"regclosed2_"+suffix+"@example.com",
		"ValidPass1",
	)
}

// TestSettings_ScoreboardHidden_BlocksNonAdmin verifies that scoreboard_visible="hidden"
// returns 403 for regular users and 200 for admins, and that the scoreboard becomes
// accessible again after restoring scoreboard_visible="public".
// Mutates global app_settings - must not run in parallel.
func TestSettings_ScoreboardHidden_BlocksNonAdmin(t *testing.T) {
	t.Cleanup(resetAppSettingsFull)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("sb_hidden_" + suffix)

	// Create a challenge and solve it so the scoreboard has data.
	flag := "flag{sb_hidden_" + suffix + "}"
	challID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "SBHidden_" + suffix,
		"description": "scoreboard hidden test",
		"flag":        flag,
		"points":      100,
		"category":    "misc",
		"state":       "visible",
	})
	_, _, tokenUser := h.RegisterUserAndLogin("sb_hidden_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challID, flag, http.StatusOK)

	// Hide the scoreboard directly via DB.
	setAppSettingsScoreboardVisible("hidden")

	// Regular user cannot see the scoreboard.
	sbResp := h.GetScoreboard(tokenUser)
	require.Equal(t, http.StatusForbidden, sbResp.StatusCode(),
		"expected 403 for non-admin when scoreboard is hidden, got %d body=%s", sbResp.StatusCode(), sbResp.Body)

	// Admin can still see the scoreboard.
	adminSbResp := h.GetScoreboard(tokenAdmin)
	require.Equal(t, http.StatusOK, adminSbResp.StatusCode(),
		"expected admin to see scoreboard even when hidden, got %d body=%s", adminSbResp.StatusCode(), adminSbResp.Body)

	// Restore to public.
	setAppSettingsScoreboardVisible("public")

	// Regular user can now see the scoreboard.
	sbResp2 := h.GetScoreboard(tokenUser)
	require.Equal(t, http.StatusOK, sbResp2.StatusCode(),
		"expected 200 after restoring public scoreboard, got %d body=%s", sbResp2.StatusCode(), sbResp2.Body)
}

// TestSettings_MaxTeams_EnforcesLimit verifies that once the team count reaches max_teams,
// further team creation is rejected with 409, and resetting max_teams=0 (unlimited) lifts the cap.
// Mutates global app_settings - must not run in parallel.
func TestSettings_MaxTeams_EnforcesLimit(t *testing.T) {
	t.Cleanup(resetAppSettingsFull)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("maxteams_" + suffix)
	_ = tokenAdmin

	_, _, tok1 := h.RegisterUserAndLogin("maxteams_u1_" + suffix)
	_, _, tok2 := h.RegisterUserAndLogin("maxteams_u2_" + suffix)
	_, _, tok3 := h.RegisterUserAndLogin("maxteams_u3_" + suffix)

	// Get current team count so cap = current + 2 allows exactly two more.
	sbResp := h.GetScoreboard(tokenAdmin)
	currentCount := 0

	if sbResp.JSON200 != nil {
		currentCount = len(*sbResp.JSON200)
	}

	setAppSettingsMaxTeams(currentCount + 2)

	// Two more teams can be created up to the cap.
	h.CreateTeam(tok1, "MaxTeamsA_"+suffix, http.StatusCreated)
	h.CreateTeam(tok2, "MaxTeamsB_"+suffix, http.StatusCreated)

	// Third team exceeds the cap.
	h.CreateTeam(tok3, "MaxTeamsC_"+suffix, http.StatusConflict)

	// Remove the cap (0 = unlimited).
	setAppSettingsMaxTeams(0)

	// Third team can now be created.
	h.CreateTeam(tok3, "MaxTeamsC_"+suffix, http.StatusCreated)
}
