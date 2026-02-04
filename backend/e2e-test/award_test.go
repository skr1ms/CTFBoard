package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
)

// POST /admin/awards: create bonus; GET /scoreboard reflects team score = solves + award.
func TestAward_CreateBonus_ScoreboardReflects(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_award")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Award Chall", "flag{award}", 100)

	suffix := uuid.New().String()[:8]
	teamName := "award_team_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(teamName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challengeID, "flag{award}", http.StatusOK)
	h.AssertTeamScore(teamName, 100)

	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))

	h.CreateAward(tokenAdmin, teamID, 50, "bonus for style", http.StatusCreated)
	h.AssertTeamScore(teamName, 150)
}

// POST /admin/awards: create penalty (negative value); GET /scoreboard reflects reduced score.
func TestAward_CreatePenalty_ScoreboardReflects(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_penalty")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Penalty Chall", "flag{penalty}", 100)

	suffix := uuid.New().String()[:8]
	teamName := "penalty_team_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(teamName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challengeID, "flag{penalty}", http.StatusOK)
	h.AssertTeamScore(teamName, 100)

	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))

	h.CreateAward(tokenAdmin, teamID, -30, "rule violation", http.StatusCreated)
	h.AssertTeamScore(teamName, 70)
}

// GET /admin/awards/team/{teamID}: returns list of awards for team; admin only.
func TestAward_GetByTeam(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_award_list")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("awardlist_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))

	h.CreateAward(tokenAdmin, teamID, 10, "first award", http.StatusCreated)
	h.CreateAward(tokenAdmin, teamID, -5, "penalty", http.StatusCreated)

	helper.RequireAwardsCount(t, h.GetAwardsByTeam(tokenAdmin, teamID, http.StatusOK), 2)
}

// POST /admin/awards: invalid team_id returns 500.
func TestAward_Create_InvalidTeam(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_award_err")
	h.CreateAward(tokenAdmin, "00000000-0000-0000-0000-000000000000", 10, "bonus", http.StatusInternalServerError)
}

// GET /admin/awards/team/{teamID}: non-admin gets 403 Forbidden.
func TestAward_GetByTeam_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_award_gf")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("award_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))

	h.GetAwardsByTeam(tokenUser, teamID, http.StatusForbidden)
}
