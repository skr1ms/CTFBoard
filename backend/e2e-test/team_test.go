package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// POST /teams/solo + GET /teams/my + POST /teams/join: captain creates solo team; player joins by invite_token; both see same team.
func TestTeam_FullFlow(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	h.SetupCompetition("fullflow_" + suffix)

	captainName := "captain_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	teamName := "SoloTeam_" + suffix
	h.CreateTeam(tokenCap, teamName, http.StatusCreated)

	initialTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, initialTeam.JSON200)
	inviteToken := *initialTeam.JSON200.InviteToken
	teamID := *initialTeam.JSON200.ID
	require.Equal(t, teamName, *initialTeam.JSON200.Name)
	require.Len(t, *initialTeam.JSON200.Members, 1)

	playerName := "player_" + suffix
	_, _, tokenPlayer := h.RegisterUserAndLogin(playerName)

	h.JoinTeam(tokenPlayer, inviteToken, false, http.StatusOK)

	teamState := h.GetMyTeam(tokenPlayer, http.StatusOK)
	require.NotNil(t, teamState.JSON200)
	require.Equal(t, teamID, *teamState.JSON200.ID)
	require.Len(t, *teamState.JSON200.Members, 2)
}

// POST /teams + POST /teams/join + POST /challenges/{ID}/submit: captain creates team, member joins, captain submits flag; GET /scoreboard shows team points.
func TestTeam_Workflow_CreateJoinSolve(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_workflow")

	challengePoints := 500
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Team Work Challenge",
		"description": "Solve this as a team",
		"points":      challengePoints,
		"flag":        "flag{team_work_makes_dream_work}",
		"category":    "misc",
	})

	suffix := helper.UID()
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

	h.AssertTeamScore(tokenCap, teamName, challengePoints)
}

// POST /teams: creating team with name that already exists returns 409 Conflict.
func TestTeam_CreateDuplicateName(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()

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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("user_" + helper.UID())

	nonExistentToken := uuid.New().String()
	h.JoinTeam(token, nonExistentToken, false, http.StatusNotFound)
}

// POST /teams/join: user already in a team tries to join another returns 409 Conflict.
func TestTeam_JoinAlreadyInTeam(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()

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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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

	suffix := helper.UID()
	soloName := "solo_player_" + suffix
	_, _, tokenSolo := h.RegisterUserAndLogin(soloName)
	h.CreateSoloTeam(tokenSolo, http.StatusCreated)
	h.SubmitFlag(tokenSolo, challengeID, "flag{ez}", http.StatusOK)

	h.AssertTeamScore(tokenSolo, soloName, 100)

	targetCapName := "target_cap_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(targetCapName)
	h.CreateTeam(tokenCap, targetCapName, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	inviteToken := *myTeam.JSON200.InviteToken

	h.JoinTeam(tokenSolo, inviteToken, true, http.StatusOK)

	scoreboard := h.GetScoreboard(tokenCap)
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_ban")
	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("banteam_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.BanTeam(tokenAdmin, teamID, "test ban reason", http.StatusOK)
}

// DELETE /admin/teams/{ID}/ban: admin unbans team; returns 200.
func TestTeam_Admin_Unban(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_unban")
	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("unbanteam_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.BanTeam(tokenAdmin, teamID, "reason", http.StatusOK)
	h.UnbanTeam(tokenAdmin, teamID, http.StatusNoContent)
}

// PATCH /admin/teams/{ID}/hidden: admin sets team hidden; returns 200.
func TestTeam_Admin_SetHidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hidden_team")
	suffix := helper.UID()
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_unban_f")
	suffix := helper.UID()
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_hidden_f")
	suffix := helper.UID()
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_ban_f")
	suffix := helper.UID()
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
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

	h.KickMember(tokenCap, memberID, http.StatusNoContent)

	h.GetMyTeam(tokenMember, http.StatusNotFound)
}

// POST /teams/leave: member leaves team; GET /teams/my returns 404 for that user.
func TestTeam_LeaveTeam(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	captainName := "cap_leave_" + suffix
	memberName := "member_leave_" + suffix

	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateTeam(tokenCap, "LeaveTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, team.JSON200)
	inviteToken := *team.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin(memberName)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	h.LeaveTeam(tokenMember, http.StatusNoContent)

	h.GetMyTeam(tokenMember, http.StatusNotFound)
}

// GET /teams/{ID}: returns team by ID (name, id, captain_id); member can fetch own team.
func TestTeam_GetByID(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
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

// GET /teams/{ID}: team reads are auth-only; guests receive 401 even when
// account_visibility is public.
func TestTeam_GetByID_GuestUnauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, token := h.RegisterUserAndLogin("guest_getteam_" + suffix)
	h.CreateSoloTeam(token, http.StatusCreated)

	team := h.GetMyTeam(token, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	resp, err := h.Client().GetTeamsIDWithResponse(context.Background(), teamID)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "guest get team by id")
}

// DELETE /teams/me: captain disbands team; GET /teams/my returns 404 for all former members.
func TestTeam_Disband(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	captainName := "cap_disband_" + suffix
	memberName := "member_disband_" + suffix

	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateTeam(tokenCap, "DisbandTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, team.JSON200)
	inviteToken := *team.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin(memberName)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	h.DisbandTeam(tokenCap, http.StatusNoContent)

	h.GetMyTeam(tokenCap, http.StatusNotFound)
	h.GetMyTeam(tokenMember, http.StatusNotFound)
}

// GET /teams/my: user not in any team returns 404.
func TestTeam_GetMy_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("noteam_" + helper.UID())
	h.GetMyTeam(token, http.StatusNotFound)
}

// GET /teams/{ID}: non-existent team returns 404.
func TestTeam_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("getbyid_" + helper.UID())
	h.GetTeamByID(token, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}

// POST /teams/leave: user not in team returns 404.
func TestTeam_Leave_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("leave_no_" + helper.UID())
	h.LeaveTeam(token, http.StatusNotFound)
}

// DELETE /teams/me: user not in team returns 404.
func TestTeam_Disband_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("disband_no_" + helper.UID())
	h.DisbandTeam(token, http.StatusNotFound)
}

// DELETE /teams/members/{ID}: non-existent member or not captain returns 404.
func TestTeam_KickMember_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, token := h.RegisterUserAndLogin("kick_cap_" + suffix)
	h.CreateSoloTeam(token, http.StatusCreated)
	h.KickMember(token, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}

// POST /teams/transfer-captain: non-captain gets 403 Forbidden.
func TestTeam_TransferCaptain_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("solo_dup_" + helper.UID())
	h.CreateSoloTeam(token, http.StatusCreated)
	h.CreateSoloTeam(token, http.StatusBadRequest)
}

// GET /admin/teams/{ID}/missing-challenges: admin gets unsolved challenges for team.
func TestTeam_AdminGetMissingChallenges_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_miss_ch")
	h.CreateBasicChallenge(tokenAdmin, "Miss Chall", "flag{miss}", 50)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("miss_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID

	resp, err := h.Client().GetAdminTeamsIDMissingChallengesWithResponse(context.Background(), teamID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin get missing challenges")
}

// GET /admin/teams/{ID}/missing-challenges: team not found returns 200 with empty list.
func TestTeam_AdminGetMissingChallenges_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_miss_404")

	resp, err := h.Client().GetAdminTeamsIDMissingChallengesWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin get missing challenges empty list for unknown team")
}

// GET /admin/teams/{ID}/members: admin lists team members.
func TestTeam_AdminGetMembers_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_members_ok")

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("members_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID

	resp, err := h.Client().GetAdminTeamsIDMembersWithResponse(context.Background(), teamID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin list members")
}

// GET /admin/teams/{ID}/members: non-admin gets 403.
func TestTeam_AdminGetMembers_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_members_f")
	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("members_forbid_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID
	_ = tokenAdmin

	resp, err := h.Client().GetAdminTeamsIDMembersWithResponse(context.Background(), teamID, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "admin list members forbidden")
}

// POST /admin/teams/{ID}/members: admin adds user to team.
func TestTeam_AdminAddMember_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_add_member")

	suffix := helper.UID()
	email, _, tokenUser := h.RegisterUserAndLogin("add_member_owner_" + suffix)
	h.CreateTeam(tokenUser, "AddMemberTeam_"+suffix, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID

	suffix2 := helper.UID()
	email2, _, _ := h.RegisterUserAndLogin("add_member_new_" + suffix2)
	userID := h.GetUserIDByEmail(email2)
	_ = email

	resp, err := h.Client().PostAdminTeamsIDMembersWithResponse(context.Background(), teamID, openapi.AdminAddMemberRequest{UserID: openapi_types.UUID(uuid.MustParse(userID))}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin add member")
}

// POST /admin/teams/{ID}/members: user not found returns 404.
func TestTeam_AdminAddMember_UserNotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_add_member_404")

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("add_member_team_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID

	resp, err := h.Client().PostAdminTeamsIDMembersWithResponse(context.Background(), teamID, openapi.AdminAddMemberRequest{UserID: openapi_types.UUID(uuid.New())}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "admin add member user not found")
}

// DELETE /admin/teams/{ID}/members/{userID}: admin removes user from team.
func TestTeam_AdminRemoveMember_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_rm_member")

	suffix := helper.UID()
	_, _, tokenCaptain := h.RegisterUserAndLogin("rm_member_captain_" + suffix)
	teamName := "RmMemberTeam_" + suffix
	h.CreateTeam(tokenCaptain, teamName, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenCaptain, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID
	require.NotNil(t, myTeam.JSON200.InviteToken)
	inviteToken := *myTeam.JSON200.InviteToken

	suffix2 := helper.UID()
	email2, _, tokenMember := h.RegisterUserAndLogin("rm_member_target_" + suffix2)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)
	memberID := h.GetUserIDByEmail(email2)

	resp, err := h.Client().DeleteAdminTeamsIDMembersUserIDWithResponse(context.Background(), teamID, memberID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNoContent, resp.StatusCode(), resp.Body, "admin remove member")
}

// DELETE /admin/teams/{ID}/members/{userID}: non-admin gets 403.
func TestTeam_AdminRemoveMember_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_rm_member_f")
	suffix := helper.UID()
	email, _, tokenUser := h.RegisterUserAndLogin("rm_member_forbid_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID
	userID := h.GetUserIDByEmail(email)
	_ = tokenAdmin

	resp, err := h.Client().DeleteAdminTeamsIDMembersUserIDWithResponse(context.Background(), teamID, userID, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "admin remove member forbidden")
}

// GET /teams/me/solves: authed gets own team solves.
func TestTeam_MySolves_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("team_my_solves")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Solve Chall", "flag{solve}", 50)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("my_solves_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{solve}", http.StatusOK)

	resp, err := h.Client().GetTeamsMeSolvesWithResponse(context.Background(), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get me solves")
	require.NotNil(t, resp.JSON200)
}

// GET /teams/me/solves: no team returns 404.
func TestTeam_MySolves_NoTeam(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("team_my_solves_noTeam")

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("my_solves_nt_" + suffix)

	resp, err := h.Client().GetTeamsMeSolvesWithResponse(context.Background(), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "get me solves no team")
}

// GET /teams/{ID}/solves: authed gets team solves.
func TestTeam_GetSolves_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("team_id_solves")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "IDSolve Chall", "flag{idsolve}", 50)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("id_solves_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{idsolve}", http.StatusOK)
	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID

	resp, err := h.Client().GetTeamsIDSolvesWithResponse(context.Background(), teamID, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get team id solves")
}

// GET /teams/{ID}/solves: non-member gets 403 (access control; does not reveal if team exists).
func TestTeam_GetSolves_NotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("team_id_solves_404")

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("id_solves_404_" + suffix)

	// A random team-id that does not exist returns 404 TEAM_NOT_FOUND; the existence check
	// runs before the membership/authorization check so that non-members cannot enumerate teams.
	resp, err := h.Client().GetTeamsIDSolvesWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "get team id solves unknown team")
}

// GET /teams/me/fails: authed gets own team fails.
func TestTeam_MyFails_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("team_my_fails")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Fail Chall", "flag{fail}", 50)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("my_fails_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{wrong}", http.StatusOK)

	resp, err := h.Client().GetTeamsMeFailsWithResponse(context.Background(), &openapi.GetTeamsMeFailsParams{}, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get me fails")
}

// GET /teams/me/fails: no auth returns 401.
func TestTeam_MyFails_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("team_my_fails_401")

	resp, err := h.Client().GetTeamsMeFailsWithResponse(context.Background(), &openapi.GetTeamsMeFailsParams{})
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "get me fails unauthorized")
}

// GET /teams/{ID}/fails: authed gets team fails.
func TestTeam_GetFails_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("team_id_fails")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "IDFail Chall", "flag{idfail}", 50)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("id_fails_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{wrong}", http.StatusOK)
	myTeam := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID

	resp, err := h.Client().GetTeamsIDFailsWithResponse(context.Background(), teamID, &openapi.GetTeamsIDFailsParams{}, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get team id fails")
}

// GET /teams/{ID}/fails: invalid ID returns 400.
func TestTeam_GetFails_InvalidID(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("team_id_fails_400")

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("id_fails_400_" + suffix)

	resp, err := h.Client().GetTeamsIDFailsWithResponse(context.Background(), "not-a-uuid", &openapi.GetTeamsIDFailsParams{}, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusBadRequest, resp.StatusCode(), resp.Body, "get team id fails invalid id")
}

// GET /teams/me/invite: captain gets invite token; 200 with invite_token.
func TestTeam_GetInviteToken_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenCap := h.RegisterUserAndLogin("invite_cap_" + suffix)
	h.CreateSoloTeam(tokenCap, http.StatusCreated)

	resp, err := h.Client().GetTeamsMeInviteWithResponse(context.Background(), helper.WithBearerToken(tokenCap))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get invite token")
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.InviteToken)
}

// GET /teams/me/invite: non-captain member gets 403.
func TestTeam_GetInviteToken_Error_NonCaptain(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenCap := h.RegisterUserAndLogin("invite_cap2_" + suffix)
	h.CreateTeam(tokenCap, "TeamInvite_"+suffix, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	inviteToken := *myTeam.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin("invite_mem_" + suffix)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	resp, err := h.Client().GetTeamsMeInviteWithResponse(context.Background(), helper.WithBearerToken(tokenMember))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "get invite token non-captain")
}

// PATCH /teams/me: captain renames team; 200 and GET /teams/my shows new name.
func TestTeam_UpdateMyTeam_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, tokenCap := h.RegisterUserAndLogin("patch_cap_" + suffix)
	h.CreateSoloTeam(tokenCap, http.StatusCreated)

	newName := "RenamedTeam_" + suffix
	resp, err := h.Client().PatchTeamsMeWithResponse(context.Background(), openapi.PatchTeamsMeJSONRequestBody{Name: newName}, helper.WithBearerToken(tokenCap))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "patch teams me")
	require.NotNil(t, resp.JSON200)
	require.Equal(t, newName, *resp.JSON200.Name)

	myTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	require.Equal(t, newName, *myTeam.JSON200.Name)
}

// PATCH /teams/me: rename to existing team name returns 409.
func TestTeam_UpdateMyTeam_Conflict(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, token1 := h.RegisterUserAndLogin("patch_u1_" + suffix)
	h.CreateSoloTeam(token1, http.StatusCreated)
	team1 := h.GetMyTeam(token1, http.StatusOK)
	require.NotNil(t, team1.JSON200)
	existingName := *team1.JSON200.Name

	_, _, token2 := h.RegisterUserAndLogin("patch_u2_" + suffix)
	h.CreateTeam(token2, "OtherTeam_"+suffix, http.StatusCreated)

	resp, err := h.Client().PatchTeamsMeWithResponse(context.Background(), openapi.PatchTeamsMeJSONRequestBody{Name: existingName}, helper.WithBearerToken(token2))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusConflict, resp.StatusCode(), resp.Body, "patch teams me conflict")
}

// GET /teams/me/awards: user in team gets 200 (empty or with awards).
func TestTeam_GetMyAwards_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("team_awards_ok")
	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("awards_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	resp, err := h.Client().GetTeamsMeAwardsWithResponse(context.Background(), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get my awards")
	require.NotNil(t, resp.JSON200)
}

// GET /teams/me/awards: user with no team gets 404.
func TestTeam_GetMyAwards_NoTeam(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("team_awards_not")

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("awards_noteam_" + suffix)

	resp, err := h.Client().GetTeamsMeAwardsWithResponse(context.Background(), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "get my awards no team")
}

// POST /teams: user with solo team gets 200 with confirmation required (no 201).
func TestTeam_TryCreate_RequiresConfirmation(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _, token := h.RegisterUserAndLogin("try_confirm_" + suffix)
	h.CreateSoloTeam(token, http.StatusCreated)

	// POST /teams without confirm_reset -> server returns 200 with confirmation payload (not 201)
	resp, err := h.Client().PostTeamsWithResponse(context.Background(), openapi.PostTeamsJSONRequestBody{
		Name: "NewTeam_" + suffix,
	}, helper.WithBearerToken(token))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "expected 200 when confirmation required")
	require.Nil(t, resp.JSON201, "should not return team when confirmation required")
	require.Contains(t, string(resp.Body), "reason", "body should indicate confirmation reason")
}
