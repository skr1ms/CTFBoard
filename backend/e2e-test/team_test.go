package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestTeam_FullFlow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

	suffix := uuid.New().String()[:8]

	captainName := "captain_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(captainName)

	initialTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	inviteToken := initialTeam.Value("invite_token").String().Raw()
	teamID := initialTeam.Value("id").String().Raw()

	initialTeam.Value("name").String().IsEqual(captainName)
	initialTeam.Value("members").Array().Length().IsEqual(1)

	playerName := "player_" + suffix
	_, _, tokenPlayer := h.RegisterUserAndLogin(playerName)

	h.JoinTeam(tokenPlayer, inviteToken, http.StatusOK)

	teamState := h.GetMyTeam(tokenPlayer, http.StatusOK)
	teamState.Value("id").String().IsEqual(teamID)
	teamState.Value("members").Array().Length().IsEqual(2)
}

func TestTeam_CreateDuplicateName(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

	suffix := uuid.New().String()[:8]

	_, _, token1 := h.RegisterUserAndLogin("captain1_" + suffix)
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

func TestTeam_JoinInvalidToken(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

	_, _, token := h.RegisterUserAndLogin("user_" + uuid.New().String()[:8])

	h.JoinTeam(token, "invalid-token", http.StatusNotFound)
}

func TestTeam_JoinAlreadyInTeam(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

	suffix := uuid.New().String()[:8]

	_, _, tokenCap := h.RegisterUserAndLogin("captain3_" + suffix)
	inviteTokenA := h.GetMyTeam(tokenCap, http.StatusOK).Value("invite_token").String().Raw()

	_, _, tokenUser1 := h.RegisterUserAndLogin("user1_" + suffix)
	inviteTokenB := h.GetMyTeam(tokenUser1, http.StatusOK).Value("invite_token").String().Raw()

	_, _, tokenUser2 := h.RegisterUserAndLogin("user2_" + suffix)
	h.JoinTeam(tokenUser2, inviteTokenB, http.StatusOK)

	h.JoinTeam(tokenUser1, inviteTokenA, http.StatusConflict)
}

func TestTeam_Join_PointsCheck(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

	suffix := uuid.New().String()[:8]
	soloName := "solo_player_" + suffix

	_, _, tokenAdmin := h.RegisterAdmin("admin_" + suffix)
	challengeID := h.CreateChallenge(tokenAdmin, map[string]interface{}{
		"title":       "Solvable",
		"description": "Test team points",
		"points":      100,
		"flag":        "flag{ez}",
		"category":    "misc",
	})

	_, _, tokenSolo := h.RegisterUserAndLogin(soloName)
	h.SubmitFlag(tokenSolo, challengeID, "flag{ez}", http.StatusOK)

	h.AssertTeamScore(soloName, 100)

	targetCapName := "target_cap_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(targetCapName)
	inviteToken := h.GetMyTeam(tokenCap, http.StatusOK).Value("invite_token").String().Raw()

	h.JoinTeam(tokenSolo, inviteToken, http.StatusOK)

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

func TestTeam_GetMyTeamWithoutAuth(t *testing.T) {
	e := setupE2E(t)
	e.GET("/api/v1/teams/my").
		Expect().
		Status(http.StatusUnauthorized)
}
