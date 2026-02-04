package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// POST /teams/solo + GET /teams/my + POST /teams/join: captain creates solo team; player joins by invite_token; both see same team.
func TestTeam_FullFlow(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]

	captainName := "captain_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateSoloTeam(tokenCap, http.StatusCreated)

	initialTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, initialTeam.JSON200)
	inviteToken := *initialTeam.JSON200.InviteToken
	teamID := *initialTeam.JSON200.ID
	require.Equal(t, captainName, *initialTeam.JSON200.Name)
	require.Len(t, *initialTeam.JSON200.Members, 1)

	playerName := "player_" + suffix
	_, _, tokenPlayer := h.RegisterUserAndLogin(playerName)

	h.JoinTeam(tokenPlayer, inviteToken, false, http.StatusOK)

	teamState := h.GetMyTeam(tokenPlayer, http.StatusOK)
	require.NotNil(t, teamState.JSON200)
	require.Equal(t, teamID, *teamState.JSON200.ID)
	require.Len(t, *teamState.JSON200.Members, 2)
}

// Full flow: competition + challenge + captain creates team + member joins + captain submits flag; GET /scoreboard shows team points.
func TestTeam_Workflow_CreateJoinSolve(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_workflow")

	challengePoints := 500
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Team Work Challenge",
		"description": "Solve this as a team",
		"points":      challengePoints,
		"flag":        "flag{team_work_makes_dream_work}",
		"category":    "misc",
	})

	suffix := uuid.New().String()[:8]
	captainName := "capt_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(captainName)

	teamName := "SuperTeam_" + suffix
	h.CreateTeam(tokenCap, teamName, http.StatusCreated)

	myTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	require.Equal(t, teamName, *myTeam.JSON200.Name)
	inviteToken := *myTeam.JSON200.InviteToken
	teamID := *myTeam.JSON200.ID

	memberName := "member_" + suffix
	_, _, tokenMember := h.RegisterUserAndLogin(memberName)

	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	memberTeam := h.GetMyTeam(tokenMember, http.StatusOK)
	require.NotNil(t, memberTeam.JSON200)
	require.Equal(t, teamID, *memberTeam.JSON200.ID)
	require.Equal(t, teamName, *memberTeam.JSON200.Name)
	require.Len(t, *memberTeam.JSON200.Members, 2)

	h.SubmitFlag(tokenCap, challengeID, "flag{team_work_makes_dream_work}", http.StatusOK)

	h.AssertTeamScore(teamName, challengePoints)
}

// POST /teams: creating team with name that already exists returns 409 Conflict.
func TestTeam_CreateDuplicateName(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]

	_, _, token1 := h.RegisterUserAndLogin("captain1_" + suffix)
	h.CreateSoloTeam(token1, http.StatusCreated)
	myTeam := h.GetMyTeam(token1, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamName1 := *myTeam.JSON200.Name

	_, _, token2 := h.RegisterUserAndLogin("captain2_" + suffix)

	h.CreateTeam(token2, teamName1, http.StatusConflict)
}

// POST /teams/join: invalid invite_token returns 404.
func TestTeam_JoinInvalidToken(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("user_" + uuid.New().String()[:8])

	nonExistentToken := uuid.New().String()
	h.JoinTeam(token, nonExistentToken, false, http.StatusNotFound)
}

// POST /teams/join: user already in a team tries to join another returns 409 Conflict.
func TestTeam_JoinAlreadyInTeam(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]

	_, _, tokenCap := h.RegisterUserAndLogin("captain3_" + suffix)
	h.CreateTeam(tokenCap, "TeamA_"+suffix, http.StatusCreated)
	teamA := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, teamA.JSON200)
	inviteTokenA := *teamA.JSON200.InviteToken

	_, _, tokenUser1 := h.RegisterUserAndLogin("user1_" + suffix)
	h.CreateTeam(tokenUser1, "TeamB_"+suffix, http.StatusCreated)
	teamB := h.GetMyTeam(tokenUser1, http.StatusOK)
	require.NotNil(t, teamB.JSON200)
	inviteTokenB := *teamB.JSON200.InviteToken

	_, _, tokenUser2 := h.RegisterUserAndLogin("user2_" + suffix)

	h.JoinTeam(tokenUser2, inviteTokenB, false, http.StatusOK)

	h.JoinTeam(tokenUser1, inviteTokenA, false, http.StatusConflict)
}

// POST /teams/join with confirm_reset: solo player with points joins another team; scoreboard shows target team with 0 (points reset).
func TestTeam_Join_PointsCheck(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_points")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Solvable",
		"description":   "Test team points",
		"points":        100,
		"flag":          "flag{ez}",
		"category":      "misc",
		"initial_value": 100,
		"min_value":     100,
		"decay":         1,
	})

	suffix := uuid.New().String()[:8]
	soloName := "solo_player_" + suffix
	_, _, tokenSolo := h.RegisterUserAndLogin(soloName)
	h.CreateSoloTeam(tokenSolo, http.StatusCreated)
	h.SubmitFlag(tokenSolo, challengeID, "flag{ez}", http.StatusOK)

	h.AssertTeamScore(soloName, 100)

	targetCapName := "target_cap_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(targetCapName)
	h.CreateTeam(tokenCap, targetCapName, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	inviteToken := *myTeam.JSON200.InviteToken

	h.JoinTeam(tokenSolo, inviteToken, true, http.StatusOK)

	scoreboard := h.GetScoreboard()
	require.NotNil(t, scoreboard.JSON200)
	teamPoints := -1
	for _, entry := range *scoreboard.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == targetCapName {
			if entry.Points != nil {
				teamPoints = *entry.Points
			} else {
				teamPoints = 0
			}
			break
		}
	}

	if teamPoints == -1 {
		t.Log("Target team not found in scoreboard (likely 0 points)")
	} else if teamPoints != 0 {
		t.Errorf("Points PERSISTED! Unexpected behavior. Points: %v", teamPoints)
	} else {
		t.Log("Points were reset as expected.")
	}
}

// POST /admin/teams/{ID}/ban: admin bans team; returns 200.
func TestTeam_Admin_Ban(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_ban")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("banteam_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.BanTeam(tokenAdmin, teamID, "test ban reason", http.StatusOK)
}

// DELETE /admin/teams/{ID}/ban: admin unbans team; returns 200.
func TestTeam_Admin_Unban(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_unban")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("unbanteam_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.BanTeam(tokenAdmin, teamID, "reason", http.StatusOK)
	h.UnbanTeam(tokenAdmin, teamID, http.StatusOK)
}

// PATCH /admin/teams/{ID}/hidden: admin sets team hidden; returns 200.
func TestTeam_Admin_SetHidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hidden_team")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("hiddenteam_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.SetTeamHidden(tokenAdmin, teamID, true, http.StatusOK)
	h.SetTeamHidden(tokenAdmin, teamID, false, http.StatusOK)
}

// DELETE /admin/teams/{ID}/ban: non-admin gets 403 Forbidden.
func TestTeam_Admin_Unban_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_unban_f")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("user_unban_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID
	_, _, tokenOther := h.RegisterUserAndLogin("other_unban_" + suffix)
	h.CreateSoloTeam(tokenOther, http.StatusCreated)
	h.UnbanTeam(tokenOther, teamID, http.StatusForbidden)
}

// PATCH /admin/teams/{ID}/hidden: non-admin gets 403 Forbidden.
func TestTeam_Admin_SetHidden_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_hidden_f")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("user_hidden_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID
	_, _, tokenOther := h.RegisterUserAndLogin("other_hidden_" + suffix)
	h.CreateSoloTeam(tokenOther, http.StatusCreated)
	h.SetTeamHidden(tokenOther, teamID, true, http.StatusForbidden)
}

// POST /admin/teams/{ID}/ban: non-admin gets 403.
func TestTeam_Admin_Ban_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_ban_f")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("user_ban_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	_, _, tokenOther := h.RegisterUserAndLogin("other_" + suffix)
	h.CreateSoloTeam(tokenOther, http.StatusCreated)

	h.BanTeam(tokenOther, teamID, "malicious", http.StatusForbidden)
}

// POST /teams/transfer-captain: captain transfers role to another member; GET /teams/my shows new captain_id.
func TestTeam_TransferCaptain(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	captainName := "cap_transfer_" + suffix
	memberName := "member_transfer_" + suffix

	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateTeam(tokenCap, "TransferTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, team.JSON200)
	inviteToken := *team.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin(memberName)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	teamAfterJoin := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, teamAfterJoin.JSON200)
	var memberID string
	for _, m := range *teamAfterJoin.JSON200.Members {
		if m.Username != nil && *m.Username == memberName {
			require.NotNil(t, m.ID)
			memberID = *m.ID
			break
		}
	}
	require.NotEmpty(t, memberID, "member not found in team")

	h.TransferCaptain(tokenCap, memberID, http.StatusOK)

	newCapTeam := h.GetMyTeam(tokenMember, http.StatusOK)
	require.NotNil(t, newCapTeam.JSON200)
	require.NotNil(t, newCapTeam.JSON200.CaptainID)
	require.Equal(t, memberID, *newCapTeam.JSON200.CaptainID)
}

// DELETE /teams/members/{ID}: captain kicks member; kicked user GET /teams/my returns 404.
func TestTeam_KickMember(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	captainName := "cap_kick_" + suffix
	memberName := "member_kick_" + suffix

	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateTeam(tokenCap, "KickTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, team.JSON200)
	inviteToken := *team.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin(memberName)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	teamWithMember := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, teamWithMember.JSON200)
	var memberID string
	for _, m := range *teamWithMember.JSON200.Members {
		if m.Username != nil && *m.Username == memberName {
			require.NotNil(t, m.ID)
			memberID = *m.ID
			break
		}
	}
	require.NotEmpty(t, memberID, "member not found")

	h.KickMember(tokenCap, memberID, http.StatusOK)

	h.GetMyTeam(tokenMember, http.StatusNotFound)
}

// POST /teams/leave: member leaves team; GET /teams/my returns 404 for that user.
func TestTeam_LeaveTeam(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	captainName := "cap_leave_" + suffix
	memberName := "member_leave_" + suffix

	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateTeam(tokenCap, "LeaveTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, team.JSON200)
	inviteToken := *team.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin(memberName)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	h.LeaveTeam(tokenMember, http.StatusOK)

	h.GetMyTeam(tokenMember, http.StatusNotFound)
}

// GET /teams/{ID}: returns team by ID (name, id, captain_id); member can fetch own team.
func TestTeam_GetByID(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, token := h.RegisterUserAndLogin("getteam_" + suffix)
	h.CreateSoloTeam(token, http.StatusCreated)

	team := h.GetMyTeam(token, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	got := h.GetTeamByID(token, teamID, http.StatusOK)
	require.NotNil(t, got.JSON200)
	require.Equal(t, teamID, *got.JSON200.ID)
	require.NotEmpty(t, *got.JSON200.Name)
	require.NotNil(t, got.JSON200.CaptainID)
}

// DELETE /teams/me: captain disbands team; GET /teams/my returns 404 for all former members.
func TestTeam_Disband(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	captainName := "cap_disband_" + suffix
	memberName := "member_disband_" + suffix

	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateTeam(tokenCap, "DisbandTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, team.JSON200)
	inviteToken := *team.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin(memberName)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	h.DisbandTeam(tokenCap, http.StatusOK)

	h.GetMyTeam(tokenCap, http.StatusNotFound)
	h.GetMyTeam(tokenMember, http.StatusNotFound)
}

// GET /teams/my: user not in any team returns 404.
func TestTeam_GetMy_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("noteam_" + uuid.New().String()[:8])
	h.GetMyTeam(token, http.StatusNotFound)
}

// GET /teams/{ID}: non-existent team returns 404.
func TestTeam_GetByID_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("getbyid_" + uuid.New().String()[:8])
	h.GetTeamByID(token, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}

// POST /teams/leave: user not in team returns 404.
func TestTeam_Leave_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("leave_no_" + uuid.New().String()[:8])
	h.LeaveTeam(token, http.StatusNotFound)
}

// DELETE /teams/me: user not in team returns 404.
func TestTeam_Disband_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("disband_no_" + uuid.New().String()[:8])
	h.DisbandTeam(token, http.StatusNotFound)
}

// DELETE /teams/members/{ID}: non-existent member or not captain returns 404.
func TestTeam_KickMember_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, token := h.RegisterUserAndLogin("kick_cap_" + suffix)
	h.CreateSoloTeam(token, http.StatusCreated)
	h.KickMember(token, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}

// POST /teams/transfer-captain: non-captain gets 403 Forbidden.
func TestTeam_TransferCaptain_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	capName := "cap_tf_" + suffix
	memName := "mem_tf_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(capName)
	h.CreateTeam(tokenCap, "TfTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, team.JSON200)
	inviteToken := *team.JSON200.InviteToken
	_, _, tokenMem := h.RegisterUserAndLogin(memName)
	h.JoinTeam(tokenMem, inviteToken, false, http.StatusOK)
	teamAfter := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, teamAfter.JSON200)
	var capID string
	for _, m := range *teamAfter.JSON200.Members {
		if m.Username != nil && *m.Username == capName {
			require.NotNil(t, m.ID)
			capID = *m.ID
			break
		}
	}
	require.NotEmpty(t, capID)
	h.TransferCaptain(tokenMem, capID, http.StatusForbidden)
}

// POST /teams/solo: user already in team gets 400 Conflict.
func TestTeam_CreateSolo_Conflict(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("solo_dup_" + uuid.New().String()[:8])
	h.CreateSoloTeam(token, http.StatusCreated)
	h.CreateSoloTeam(token, http.StatusBadRequest)
}
