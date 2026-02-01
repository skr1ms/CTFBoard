package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// POST /admin/awards: create bonus; GET /scoreboard reflects team score = solves + award.
func TestAward_CreateBonus_ScoreboardReflects(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_award")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Award Chall", "flag{award}", 100)

	suffix := uuid.New().String()[:8]
	teamName := "award_team_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(teamName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challengeID, "flag{award}", http.StatusOK)
	h.AssertTeamScore(teamName, 100)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	teamID := team.Value("id").String().Raw()

	h.CreateAward(tokenAdmin, teamID, 50, "bonus for style", http.StatusCreated)
	h.AssertTeamScore(teamName, 150)
}

// POST /admin/awards: create penalty (negative value); GET /scoreboard reflects reduced score.
func TestAward_CreatePenalty_ScoreboardReflects(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_penalty")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Penalty Chall", "flag{penalty}", 100)

	suffix := uuid.New().String()[:8]
	teamName := "penalty_team_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(teamName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challengeID, "flag{penalty}", http.StatusOK)
	h.AssertTeamScore(teamName, 100)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	teamID := team.Value("id").String().Raw()

	h.CreateAward(tokenAdmin, teamID, -30, "rule violation", http.StatusCreated)
	h.AssertTeamScore(teamName, 70)
}

// GET /admin/awards/team/{teamID}: returns list of awards for team; admin only.
func TestAward_GetByTeam(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_award_list")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("awardlist_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	team := h.GetMyTeam(tokenUser, http.StatusOK)
	teamID := team.Value("id").String().Raw()

	h.CreateAward(tokenAdmin, teamID, 10, "first award", http.StatusCreated)
	h.CreateAward(tokenAdmin, teamID, -5, "penalty", http.StatusCreated)

	awards := h.GetAwardsByTeam(tokenAdmin, teamID, http.StatusOK)
	awards.Length().IsEqual(2)
}
