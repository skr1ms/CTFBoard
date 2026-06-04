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

// TestSettings_ScoreVisibility_EnforcesModes verifies the active scoreboard
// visibility path: configs.score_visibility -> CompetitionParamUseCase cache ->
// VisibilityGuard. Mutates global configs, so it must not run in parallel.
func TestSettings_ScoreVisibility_EnforcesModes(t *testing.T) {
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("score_visibility_" + suffix)
	_, _, tokenUser := h.RegisterUserAndLogin("scorevis_user_" + suffix)

	t.Cleanup(func() {
		h.PutAdminConfig(tokenAdmin, "score_visibility", "public", "string", "reset score visibility", http.StatusOK)
	})

	respGuestPublic := h.GetScoreboard("")
	helper.RequireStatus(t, http.StatusOK, respGuestPublic.StatusCode(), respGuestPublic.Body, "scoreboard public guest")
	require.NotNil(t, respGuestPublic.JSON200)

	h.PutAdminConfig(tokenAdmin, "score_visibility", "private", "string", "private score visibility", http.StatusOK)
	respGuestPrivate := h.GetScoreboard("")
	helper.RequireStatus(t, http.StatusUnauthorized, respGuestPrivate.StatusCode(), respGuestPrivate.Body, "scoreboard private guest")

	respUserPrivate := h.GetScoreboard(tokenUser)
	helper.RequireStatus(t, http.StatusOK, respUserPrivate.StatusCode(), respUserPrivate.Body, "scoreboard private user")

	h.PutAdminConfig(tokenAdmin, "score_visibility", "hidden", "string", "hidden score visibility", http.StatusOK)
	respUserHidden := h.GetScoreboard(tokenUser)
	helper.RequireStatus(t, http.StatusNotFound, respUserHidden.StatusCode(), respUserHidden.Body, "scoreboard hidden user")

	respAdminHidden := h.GetScoreboard(tokenAdmin)
	helper.RequireStatus(t, http.StatusOK, respAdminHidden.StatusCode(), respAdminHidden.Body, "scoreboard hidden admin")

	h.PutAdminConfig(tokenAdmin, "score_visibility", "admins_only", "string", "admin-only score visibility", http.StatusOK)
	respGuestAdminsOnly := h.GetScoreboard("")
	helper.RequireStatus(t, http.StatusNotFound, respGuestAdminsOnly.StatusCode(), respGuestAdminsOnly.Body, "scoreboard admins_only guest")

	respUserAdminsOnly := h.GetScoreboard(tokenUser)
	helper.RequireStatus(t, http.StatusNotFound, respUserAdminsOnly.StatusCode(), respUserAdminsOnly.Body, "scoreboard admins_only user")

	respAdminAdminsOnly := h.GetScoreboard(tokenAdmin)
	helper.RequireStatus(t, http.StatusOK, respAdminAdminsOnly.StatusCode(), respAdminAdminsOnly.Body, "scoreboard admins_only admin")
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
