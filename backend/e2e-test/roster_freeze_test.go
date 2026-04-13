package e2e_test

import (
	"net/http"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// TestRosterFreeze_BlocksTeamOperations verifies that when allow_team_switch=false is set on
// an active competition, team join/leave/create are all blocked with 403, and they succeed
// again once allow_team_switch is re-enabled.
// Mutates global competition allow_team_switch - must not run in parallel.
func TestRosterFreeze_BlocksTeamOperations(t *testing.T) {
	t.Cleanup(resetCompetitionAllowTeamSwitch)

	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()

	// Ensure competition is active with team switching allowed so users can set up their teams.
	resetCompetitionToActive()

	// Captain creates a team; member has no team yet.
	_, _, tokenCap := h.RegisterUserAndLogin("roster_cap_" + suffix)
	h.CreateTeam(tokenCap, "RosterTeam_"+suffix, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	inviteToken := *myTeam.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin("roster_mem_" + suffix)
	_, _, tokenOther := h.RegisterUserAndLogin("roster_other_" + suffix)

	// Freeze roster via direct DB update (API blocks allow_team_switch changes while active).
	setCompetitionAllowTeamSwitch(false)

	// Join is blocked (ROSTER_FROZEN -> 403).
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusForbidden)

	// Leave is blocked for the captain.
	h.LeaveTeam(tokenCap, http.StatusForbidden)

	// Creating a new team is blocked.
	h.CreateTeam(tokenOther, "ShouldFail_"+suffix, http.StatusForbidden)

	// Re-enable team switching.
	setCompetitionAllowTeamSwitch(true)

	// Join now succeeds.
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)
}
