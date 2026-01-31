package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestTeam_FullFlow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]

	// 1. Captain registers
	captainName := "captain_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(captainName)
	h.CreateSoloTeam(tokenCap, http.StatusCreated)

	// 2. Verify Initial Team (Solo)
	initialTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	inviteToken := initialTeam.Value("invite_token").String().Raw()
	teamID := initialTeam.Value("id").String().Raw()

	initialTeam.Value("name").String().IsEqual(captainName)
	initialTeam.Value("members").Array().Length().IsEqual(1)

	// 3. Player registers
	playerName := "player_" + suffix
	_, _, tokenPlayer := h.RegisterUserAndLogin(playerName)

	// 4. Player Joins Team
	h.JoinTeam(tokenPlayer, inviteToken, false, http.StatusOK)

	// 5. Verify Team State for Player
	teamState := h.GetMyTeam(tokenPlayer, http.StatusOK)
	teamState.Value("id").String().IsEqual(teamID)
	teamState.Value("members").Array().Length().IsEqual(2)
}

func TestTeam_CreateDuplicateName(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]

	// 1. User 1 registers (gets team name = username)
	_, _, token1 := h.RegisterUserAndLogin("captain1_" + suffix)
	h.CreateSoloTeam(token1, http.StatusCreated)
	teamName1 := h.GetMyTeam(token1, http.StatusOK).Value("name").String().Raw()

	// 2. User 2 registers
	_, _, token2 := h.RegisterUserAndLogin("captain2_" + suffix)

	// 3. User 2 tries to rename team to match User 1 (Expect Conflict)
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
	h := NewE2EHelper(t, e, TestPool)

	// 1. Register User
	_, _, token := h.RegisterUserAndLogin("user_" + uuid.New().String()[:8])

	// 2. Join with fake token (Expect NotFound)
	nonExistentToken := uuid.New().String()
	h.JoinTeam(token, nonExistentToken, false, http.StatusNotFound)
}

func TestTeam_JoinAlreadyInTeam(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]

	// 1. Captain (Team A)
	_, _, tokenCap := h.RegisterUserAndLogin("captain3_" + suffix)
	h.CreateTeam(tokenCap, "TeamA_"+suffix, http.StatusCreated)
	inviteTokenA := h.GetMyTeam(tokenCap, http.StatusOK).Value("invite_token").String().Raw()

	// 2. User 1 (Team B)
	_, _, tokenUser1 := h.RegisterUserAndLogin("user1_" + suffix)
	h.CreateTeam(tokenUser1, "TeamB_"+suffix, http.StatusCreated)
	inviteTokenB := h.GetMyTeam(tokenUser1, http.StatusOK).Value("invite_token").String().Raw()

	// 3. User 2 (Team C)
	_, _, tokenUser2 := h.RegisterUserAndLogin("user2_" + suffix)

	// 4. User 2 joins Team B (Success)
	h.JoinTeam(tokenUser2, inviteTokenB, false, http.StatusOK)

	// 5. User 1 (Team B Leader) tries to join Team A (Expect Conflict)
	// (Logic: User is already in a team they can't leave implicitly?)
	h.JoinTeam(tokenUser1, inviteTokenA, false, http.StatusConflict)
}

func TestTeam_Join_PointsCheck(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_points")

	// 2. Create Challenge
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Solvable",
		"description": "Test team points",
		"points":      100,
		"flag":        "flag{ez}",
		"category":    "misc",
	})

	// 3. Solo Player Solves Challenge
	suffix := uuid.New().String()[:8]
	soloName := "solo_player_" + suffix
	_, _, tokenSolo := h.RegisterUserAndLogin(soloName)
	h.CreateSoloTeam(tokenSolo, http.StatusCreated)
	h.SubmitFlag(tokenSolo, challengeID, "flag{ez}", http.StatusOK)

	// 4. Verify Score (100)
	h.AssertTeamScore(soloName, 100)

	// 5. Target Team Captain Registers
	targetCapName := "target_cap_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(targetCapName)
	h.CreateTeam(tokenCap, targetCapName, http.StatusCreated)
	inviteToken := h.GetMyTeam(tokenCap, http.StatusOK).Value("invite_token").String().Raw()

	// 6. Solo Player Joins Target Team
	// (Points should be reset/merged? logic says reset for safety usually)
	// 6. Solo Player Joins Target Team
	// (Points should be reset/merged? logic says reset for safety usually)
	h.JoinTeam(tokenSolo, inviteToken, true, http.StatusOK)

	// 7. Verify Scoreboard (Points should be gone or 0)
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
