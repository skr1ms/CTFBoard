package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestTeam_Workflow_CreateJoinSolve(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Competition
	_, tokenAdmin := h.SetupCompetition("admin_workflow")

	// 2. Create Challenge
	challengePoints := 500
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Team Work Challenge",
		"description": "Solve this as a team",
		"points":      challengePoints,
		"flag":        "flag{team_work_makes_dream_work}",
		"category":    "misc",
	})

	// 3. Captain Registers and Creates a Custom Team
	suffix := uuid.New().String()[:8]
	captainName := "capt_" + suffix
	_, _, tokenCap := h.RegisterUserAndLogin(captainName)

	teamName := "SuperTeam_" + suffix
	h.CreateTeam(tokenCap, teamName, http.StatusCreated)

	// 4. Obtain Invite Token for the New Team
	myTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	myTeam.Value("name").String().IsEqual(teamName)
	inviteToken := myTeam.Value("invite_token").String().Raw()
	teamID := myTeam.Value("id").String().Raw()

	// 5. New Member Registers and Joins Team
	memberName := "member_" + suffix
	_, _, tokenMember := h.RegisterUserAndLogin(memberName)

	h.JoinTeam(tokenMember, inviteToken, http.StatusOK)

	// 6. Verify Team State for the New Member
	memberTeam := h.GetMyTeam(tokenMember, http.StatusOK)
	memberTeam.Value("id").String().IsEqual(teamID)
	memberTeam.Value("name").String().IsEqual(teamName)
	memberTeam.Value("members").Array().Length().IsEqual(2)

	// 7. Captain Submits Flag
	h.SubmitFlag(tokenCap, challengeID, "flag{team_work_makes_dream_work}", http.StatusOK)

	// 8. Verify Team Score reflects the Points
	h.AssertTeamScore(teamName, challengePoints)
}
