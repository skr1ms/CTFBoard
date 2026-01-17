package e2e

import (
	"testing"

	"github.com/google/uuid"
)

func TestTeam_FullFlow(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	emailCap, passCap := registerUser(e, "captain_"+suffix)
	tokenCap := login(e, emailCap, passCap)

	myTeamRespInitial := e.GET("/api/v1/teams/my").
		WithHeader("Authorization", tokenCap).
		Expect().
		Status(200).
		JSON().
		Object()

	inviteToken := myTeamRespInitial.Value("invite_token").String().Raw()
	teamId := myTeamRespInitial.Value("id").String().Raw()

	myTeamResp := e.GET("/api/v1/teams/my").
		WithHeader("Authorization", tokenCap).
		Expect().
		Status(200).
		JSON().
		Object()

	myTeamResp.Value("name").String().IsEqual("captain_" + suffix)
	myTeamResp.Value("invite_token").String().IsEqual(inviteToken)
	myTeamResp.Value("members").Array().Length().IsEqual(1)

	emailPlayer, passPlayer := registerUser(e, "player_"+suffix)
	tokenPlayer := login(e, emailPlayer, passPlayer)

	meResp := e.GET("/api/v1/auth/me").
		WithHeader("Authorization", tokenPlayer).
		Expect().
		Status(200).
		JSON().
		Object()

	meResp.Value("id").String().Raw()

	joinTeam(e, tokenPlayer, inviteToken)

	myTeamResp2 := e.GET("/api/v1/teams/my").
		WithHeader("Authorization", tokenPlayer).
		Expect().
		Status(200).
		JSON().
		Object()

	myTeamResp2.Value("id").String().IsEqual(teamId)
	myTeamResp2.Value("members").Array().Length().IsEqual(2)
}

func TestTeam_CreateDuplicateName(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	email1, pass1 := registerUser(e, "captain1_"+suffix)
	token1 := login(e, email1, pass1)

	myTeamResp1 := e.GET("/api/v1/teams/my").
		WithHeader("Authorization", token1).
		Expect().
		Status(200).
		JSON().
		Object()

	teamName1 := myTeamResp1.Value("name").String().Raw()

	email2, pass2 := registerUser(e, "captain2_"+suffix)
	token2 := login(e, email2, pass2)

	e.POST("/api/v1/teams").
		WithHeader("Authorization", token2).
		WithJSON(map[string]string{
			"name": teamName1,
		}).
		Expect().
		Status(409)
}

func TestTeam_JoinInvalidToken(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	email, pass := registerUser(e, "user_"+suffix)
	token := login(e, email, pass)

	e.POST("/api/v1/teams/join").
		WithHeader("Authorization", token).
		WithJSON(map[string]string{
			"invite_token": "invalid-token",
		}).
		Expect().
		Status(404)
}

func TestTeam_JoinAlreadyInTeam(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	emailCap, passCap := registerUser(e, "captain3_"+suffix)
	tokenCap := login(e, emailCap, passCap)
	myTeamRespCap := e.GET("/api/v1/teams/my").WithHeader("Authorization", tokenCap).Expect().Status(200).JSON().Object()
	inviteTokenA := myTeamRespCap.Value("invite_token").String().Raw()

	emailUser1, passUser1 := registerUser(e, "user1_"+suffix)
	tokenUser1 := login(e, emailUser1, passUser1)
	myTeamRespUser1 := e.GET("/api/v1/teams/my").WithHeader("Authorization", tokenUser1).Expect().Status(200).JSON().Object()
	inviteTokenB := myTeamRespUser1.Value("invite_token").String().Raw()

	emailUser2, passUser2 := registerUser(e, "user2_"+suffix)
	tokenUser2 := login(e, emailUser2, passUser2)
	joinTeam(e, tokenUser2, inviteTokenB)

	e.POST("/api/v1/teams/join").
		WithHeader("Authorization", tokenUser1).
		WithJSON(map[string]string{
			"invite_token": inviteTokenA,
		}).
		Expect().
		Status(409)
}

func TestTeam_Join_PointsCheck(t *testing.T) {
	e := setupE2E(t)

	suffix := uuid.New().String()[:8]
	emailSolo, passSolo := registerUser(e, "solo_player_"+suffix)
	tokenSolo := login(e, emailSolo, passSolo)

	adminName := "admin_" + uuid.New().String()[:8]
	_, _, adminToken := registerAdmin(e, adminName)
	challengeID := createChallenge(e, adminToken, map[string]interface{}{
		"title":       "Solvable",
		"description": "Solvable Description",
		"category":    "Web",
		"points":      100,
		"flag":        "flag{ez}",
		"is_hidden":   false,
	})

	submitFlag(e, tokenSolo, challengeID, "flag{ez}")

	scoreboard := e.GET("/api/v1/scoreboard").Expect().Status(200).JSON().Array()
	var soloPoints float64
	for _, val := range scoreboard.Iter() {
		obj := val.Object()
		if obj.Value("team_name").String().Raw() == "solo_player_"+suffix {
			soloPoints = obj.Value("points").Number().Raw()
			break
		}
	}
	if soloPoints != 100 {
		t.Fatalf("expected 100 points, got %v", soloPoints)
	}

	emailCap, passCap := registerUser(e, "target_cap_"+suffix)
	tokenCap := login(e, emailCap, passCap)
	myTeamResp := e.GET("/api/v1/teams/my").WithHeader("Authorization", tokenCap).Expect().JSON().Object()
	inviteToken := myTeamResp.Value("invite_token").String().Raw()

	joinTeam(e, tokenSolo, inviteToken)

	scoreboard2 := e.GET("/api/v1/scoreboard").Expect().Status(200).JSON().Array()
	var teamPoints float64
	foundTeam := false
	for _, val := range scoreboard2.Iter() {
		obj := val.Object()
		if obj.Value("team_name").String().Raw() == "target_cap_"+suffix {
			teamPoints = obj.Value("points").Number().Raw()
			foundTeam = true
			break
		}
	}

	if !foundTeam {
		t.Log("target team not found in scoreboard (likely 0 points)")
		return
	}

	if teamPoints != 0 {
		t.Errorf("Points PERSISTED! Unexpected behavior given current schema. Points: %v", teamPoints)
	} else {
		t.Log("Points were deleted as expected (ON DELETE CASCADE).")
	}
}

func TestTeam_GetMyTeamWithoutAuth(t *testing.T) {
	e := setupE2E(t)

	e.GET("/api/v1/teams/my").
		Expect().
		Status(401)
}
