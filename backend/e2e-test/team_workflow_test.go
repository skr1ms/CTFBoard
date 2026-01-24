package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestTeam_Workflow_CreateJoinSolve(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	suffix := uuid.New().String()[:8]

	_, _, tokenAdmin := h.RegisterAdmin("admin_" + suffix)
	h.StartCompetition(tokenAdmin)

	challengePoints := 500
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Team Work Challenge",
		"description": "Solve this as a team",
		"points":      challengePoints,
		"flag":        "flag{team_work_makes_dream_work}",
		"category":    "misc",
	})

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

	h.JoinTeam(tokenMember, inviteToken, http.StatusOK)

	memberTeam := h.GetMyTeam(tokenMember, http.StatusOK)
	memberTeam.Value("id").String().IsEqual(teamID)
	memberTeam.Value("name").String().IsEqual(teamName)
	memberTeam.Value("members").Array().Length().IsEqual(2)

	h.SubmitFlag(tokenCap, challengeID, "flag{team_work_makes_dream_work}", http.StatusOK)

	h.AssertTeamScore(teamName, challengePoints)
}
