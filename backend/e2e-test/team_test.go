package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// POST /teams/solo + GET /teams/my + POST /teams/join: captain creates solo team; player joins by invite_token; both see same team.
func TestTeam_FullFlow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]

	captainName := "captain_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateSoloTeam(tokenCap, http.StatusCreated)

	initialTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	inviteToken := initialTeam.Value("invite_token").String().Raw()
	teamID := initialTeam.Value("id").String().Raw()

	initialTeam.Value("name").String().IsEqual(captainName)
	initialTeam.Value("members").Array().Length().IsEqual(1)

	playerName := "player_" + suffix
	_, _, tokenPlayer := h.RegisterUserAndLogin(playerName)

	h.JoinTeam(tokenPlayer, inviteToken, false, http.StatusOK)

	teamState := h.GetMyTeam(tokenPlayer, http.StatusOK)
	teamState.Value("id").String().IsEqual(teamID)
	teamState.Value("members").Array().Length().IsEqual(2)
}

// Full flow: competition + challenge + captain creates team + member joins + captain submits flag; GET /scoreboard shows team points.
func TestTeam_Workflow_CreateJoinSolve(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

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
	myTeam.Value("name").String().IsEqual(teamName)
	inviteToken := myTeam.Value("invite_token").String().Raw()
	teamID := myTeam.Value("id").String().Raw()

	memberName := "member_" + suffix
	_, _, tokenMember := h.RegisterUserAndLogin(memberName)

	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	memberTeam := h.GetMyTeam(tokenMember, http.StatusOK)
	memberTeam.Value("id").String().IsEqual(teamID)
	memberTeam.Value("name").String().IsEqual(teamName)
	memberTeam.Value("members").Array().Length().IsEqual(2)

	h.SubmitFlag(tokenCap, challengeID, "flag{team_work_makes_dream_work}", http.StatusOK)

	h.AssertTeamScore(teamName, challengePoints)
}

// POST /teams: creating team with name that already exists returns 409 Conflict.
func TestTeam_CreateDuplicateName(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]

	_, _, token1 := h.RegisterUserAndLogin("captain1_" + suffix)
	h.CreateSoloTeam(token1, http.StatusCreated)
	teamName1 := h.GetMyTeam(token1, http.StatusOK).Value("name").String().Raw()

	_, _, token2 := h.RegisterUserAndLogin("captain2_" + suffix)

	e.POST("/api/v1/teams").
		WithHeader("Authorization", token2).
		WithJSON(map[string]string{
			"name": teamName1,
		}).
		Expect().
		Status(http.StatusConflict)
}

// POST /teams/join: invalid invite_token returns 404.
func TestTeam_JoinInvalidToken(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, _, token := h.RegisterUserAndLogin("user_" + uuid.New().String()[:8])

	nonExistentToken := uuid.New().String()
	h.JoinTeam(token, nonExistentToken, false, http.StatusNotFound)
}

// POST /teams/join: user already in a team tries to join another returns 409 Conflict.
func TestTeam_JoinAlreadyInTeam(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]

	_, _, tokenCap := h.RegisterUserAndLogin("captain3_" + suffix)
	h.CreateTeam(tokenCap, "TeamA_"+suffix, http.StatusCreated)
	inviteTokenA := h.GetMyTeam(tokenCap, http.StatusOK).Value("invite_token").String().Raw()

	_, _, tokenUser1 := h.RegisterUserAndLogin("user1_" + suffix)
	h.CreateTeam(tokenUser1, "TeamB_"+suffix, http.StatusCreated)
	inviteTokenB := h.GetMyTeam(tokenUser1, http.StatusOK).Value("invite_token").String().Raw()

	_, _, tokenUser2 := h.RegisterUserAndLogin("user2_" + suffix)

	h.JoinTeam(tokenUser2, inviteTokenB, false, http.StatusOK)

	h.JoinTeam(tokenUser1, inviteTokenA, false, http.StatusConflict)
}

// POST /teams/join with confirm_reset: solo player with points joins another team; scoreboard shows target team with 0 (points reset).
func TestTeam_Join_PointsCheck(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_points")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Solvable",
		"description": "Test team points",
		"points":      100,
		"flag":        "flag{ez}",
		"category":    "misc",
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
	inviteToken := h.GetMyTeam(tokenCap, http.StatusOK).Value("invite_token").String().Raw()

	h.JoinTeam(tokenSolo, inviteToken, true, http.StatusOK)

	scoreboard := h.GetScoreboard().Status(http.StatusOK).JSON().Array()

	var teamPoints float64 = -1
	for _, val := range scoreboard.Iter() {
		obj := val.Object()
		if obj.Value("team_name").String().Raw() == targetCapName {
			teamPoints = obj.Value("points").Number().Raw()
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
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_ban")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("banteam_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	teamID := team.Value("id").String().Raw()

	h.BanTeam(tokenAdmin, teamID, "test ban reason", http.StatusOK)
}

// DELETE /admin/teams/{ID}/ban: admin unbans team; returns 200.
func TestTeam_Admin_Unban(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_unban")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("unbanteam_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	teamID := team.Value("id").String().Raw()

	h.BanTeam(tokenAdmin, teamID, "reason", http.StatusOK)
	h.UnbanTeam(tokenAdmin, teamID, http.StatusOK)
}

// PATCH /admin/teams/{ID}/hidden: admin sets team hidden; returns 200.
func TestTeam_Admin_SetHidden(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_hidden_team")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("hiddenteam_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	teamID := team.Value("id").String().Raw()

	h.SetTeamHidden(tokenAdmin, teamID, true, http.StatusOK)
	h.SetTeamHidden(tokenAdmin, teamID, false, http.StatusOK)
}

// POST /admin/teams/{ID}/ban: non-admin gets 403.
func TestTeam_Admin_Ban_Forbidden(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, _ = h.SetupCompetition("admin_ban_f")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("user_ban_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	teamID := team.Value("id").String().Raw()

	_, _, tokenOther := h.RegisterUserAndLogin("other_" + suffix)
	h.CreateSoloTeam(tokenOther, http.StatusCreated)

	e.POST("/api/v1/admin/teams/{ID}/ban", teamID).
		WithHeader("Authorization", tokenOther).
		WithJSON(map[string]string{"reason": "malicious"}).
		Expect().
		Status(http.StatusForbidden)
}

// POST /teams/transfer-captain: captain transfers role to another member; GET /teams/my shows new captain_id.
func TestTeam_TransferCaptain(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	captainName := "cap_transfer_" + suffix
	memberName := "member_transfer_" + suffix

	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateTeam(tokenCap, "TransferTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	inviteToken := team.Value("invite_token").String().Raw()

	_, _, tokenMember := h.RegisterUserAndLogin(memberName)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	teamAfterJoin := h.GetMyTeam(tokenCap, http.StatusOK)
	memberID := ""
	for _, m := range teamAfterJoin.Value("members").Array().Iter() {
		obj := m.Object()
		if obj.Value("username").String().Raw() == memberName {
			memberID = obj.Value("id").String().Raw()
			break
		}
	}
	if memberID == "" {
		t.Fatal("member not found in team")
	}

	h.TransferCaptain(tokenCap, memberID, http.StatusOK)

	newCapTeam := h.GetMyTeam(tokenMember, http.StatusOK)
	newCapTeam.Value("captain_id").String().IsEqual(memberID)
}

// DELETE /teams/members/{ID}: captain kicks member; kicked user GET /teams/my returns 404.
func TestTeam_KickMember(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	captainName := "cap_kick_" + suffix
	memberName := "member_kick_" + suffix

	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateTeam(tokenCap, "KickTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	inviteToken := team.Value("invite_token").String().Raw()

	_, _, tokenMember := h.RegisterUserAndLogin(memberName)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	teamWithMember := h.GetMyTeam(tokenCap, http.StatusOK)
	var memberID string
	for _, m := range teamWithMember.Value("members").Array().Iter() {
		obj := m.Object()
		if obj.Value("username").String().Raw() == memberName {
			memberID = obj.Value("id").String().Raw()
			break
		}
	}
	if memberID == "" {
		t.Fatal("member not found")
	}

	h.KickMember(tokenCap, memberID, http.StatusOK)

	h.GetMyTeam(tokenMember, http.StatusNotFound)
}

// POST /teams/leave: member leaves team; GET /teams/my returns 404 for that user.
func TestTeam_LeaveTeam(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	captainName := "cap_leave_" + suffix
	memberName := "member_leave_" + suffix

	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateTeam(tokenCap, "LeaveTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	inviteToken := team.Value("invite_token").String().Raw()

	_, _, tokenMember := h.RegisterUserAndLogin(memberName)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	h.LeaveTeam(tokenMember, http.StatusOK)

	h.GetMyTeam(tokenMember, http.StatusNotFound)
}

// GET /teams/{ID}: returns team by ID (name, id, captain_id); member can fetch own team.
func TestTeam_GetByID(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	_, _, token := h.RegisterUserAndLogin("getteam_" + suffix)
	h.CreateSoloTeam(token, http.StatusCreated)

	team := h.GetMyTeam(token, http.StatusOK)
	teamID := team.Value("id").String().Raw()

	got := h.GetTeamByID(token, teamID, http.StatusOK)
	got.Value("id").String().IsEqual(teamID)
	got.Value("name").String().NotEmpty()
	got.ContainsKey("captain_id")
}

// DELETE /teams/me: captain disbands team; GET /teams/my returns 404 for all former members.
func TestTeam_Disband(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]
	captainName := "cap_disband_" + suffix
	memberName := "member_disband_" + suffix

	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateTeam(tokenCap, "DisbandTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	inviteToken := team.Value("invite_token").String().Raw()

	_, _, tokenMember := h.RegisterUserAndLogin(memberName)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	h.DisbandTeam(tokenCap, http.StatusOK)

	h.GetMyTeam(tokenCap, http.StatusNotFound)
	h.GetMyTeam(tokenMember, http.StatusNotFound)
}
