package e2e_test

import (
	"net/http"
	"testing"

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

// TestSettings_ScoreboardHidden_BlocksNonAdmin was wired against the legacy
// app_settings.scoreboard_visible column. The scoreboard route now uses
// middleware.VisibilityGuard(score_visibility) reading from the configs table via
// CompetitionParamUseCase, which internally holds a closure-local 30s cachekit.CachedValue
// that cannot be invalidated from tests. A dedicated test for the new route belongs in
// competition_params_test.go and should toggle configs.score_visibility via the API.
func TestSettings_ScoreboardHidden_BlocksNonAdmin(t *testing.T) {
	t.Skip("legacy app_settings.scoreboard_visible is no longer wired; superseded by VisibilityGuard(score_visibility) on configs")
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
