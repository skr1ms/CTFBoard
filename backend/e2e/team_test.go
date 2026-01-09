package e2e

import (
	"context"
	"testing"
)

func TestTeam_FullFlow(t *testing.T) {
	e := setupE2E(t)

	emailCap, passCap := registerUser(e, "captain")
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

	myTeamResp.Value("name").String().IsEqual("captain")
	myTeamResp.Value("invite_token").String().IsEqual(inviteToken)
	myTeamResp.Value("members").Array().Length().IsEqual(1)

	emailPlayer, passPlayer := registerUser(e, "player")
	tokenPlayer := login(e, emailPlayer, passPlayer)

	meResp := e.GET("/api/v1/auth/me").
		WithHeader("Authorization", tokenPlayer).
		Expect().
		Status(200).
		JSON().
		Object()

	playerUserId := meResp.Value("id").String().Raw()

	_, err := TestDB.ExecContext(context.Background(), "UPDATE users SET team_id = NULL WHERE id = ?", playerUserId)
	if err != nil {
		t.Fatalf("failed to remove player from team: %v", err)
	}

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

	email1, pass1 := registerUser(e, "captain1")
	token1 := login(e, email1, pass1)

	myTeamResp1 := e.GET("/api/v1/teams/my").
		WithHeader("Authorization", token1).
		Expect().
		Status(200).
		JSON().
		Object()

	teamName1 := myTeamResp1.Value("name").String().Raw()

	email2, pass2 := registerUser(e, "captain2")
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

	email, pass := registerUser(e, "user")
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

	emailCap, passCap := registerUser(e, "captain3")
	tokenCap := login(e, emailCap, passCap)

	myTeamRespCap := e.GET("/api/v1/teams/my").
		WithHeader("Authorization", tokenCap).
		Expect().
		Status(200).
		JSON().
		Object()

	inviteToken := myTeamRespCap.Value("invite_token").String().Raw()

	emailPlayer, passPlayer := registerUser(e, "player2")
	tokenPlayer := login(e, emailPlayer, passPlayer)

	meResp := e.GET("/api/v1/auth/me").
		WithHeader("Authorization", tokenPlayer).
		Expect().
		Status(200).
		JSON().
		Object()

	playerUserId := meResp.Value("id").String().Raw()

	_, err := TestDB.ExecContext(context.Background(), "UPDATE users SET team_id = NULL WHERE id = ?", playerUserId)
	if err != nil {
		t.Fatalf("failed to remove player from team: %v", err)
	}

	joinTeam(e, tokenPlayer, inviteToken)

	e.POST("/api/v1/teams/join").
		WithHeader("Authorization", tokenPlayer).
		WithJSON(map[string]string{
			"invite_token": inviteToken,
		}).
		Expect().
		Status(409)
}

func TestTeam_GetMyTeamWithoutAuth(t *testing.T) {
	e := setupE2E(t)

	e.GET("/api/v1/teams/my").
		Expect().
		Status(401)
}
