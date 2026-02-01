package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// GET /statistics/general: returns user_count, team_count, challenge_count, solve_count (public, no auth).
func TestStatistics_General(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_stats")
	h.CreateBasicChallenge(tokenAdmin, "Stats Chall", "flag{stats}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("statsuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	obj := h.GetStatisticsGeneral()
	obj.ContainsKey("user_count")
	obj.ContainsKey("team_count")
	obj.ContainsKey("challenge_count")
	obj.ContainsKey("solve_count")
	obj.Value("user_count").Number().Ge(1)
	obj.Value("team_count").Number().Ge(1)
	obj.Value("challenge_count").Number().Ge(1)
	obj.Value("solve_count").Number().Ge(0)
}

// GET /statistics/challenges: returns array of challenge stats (id, title, points, solve_count, category); public.
func TestStatistics_Challenges(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_stats_chall")
	h.CreateBasicChallenge(tokenAdmin, "Chall A", "flag{a}", 50)
	h.CreateBasicChallenge(tokenAdmin, "Chall B", "flag{b}", 100)

	arr := h.GetStatisticsChallenges()
	arr.Length().Ge(2)
	foundA, foundB := false, false
	for _, val := range arr.Iter() {
		o := val.Object()
		title := o.Value("title").String().Raw()
		if title == "Chall A" {
			foundA = true
		}
		if title == "Chall B" {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Errorf("expected Chall A and Chall B in statistics/challenges, found A=%v B=%v", foundA, foundB)
	}
}

// GET /statistics/scoreboard: returns scoreboard history entries; optional limit query; public.
func TestStatistics_Scoreboard(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_stats_sb")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "SB Chall", "flag{sb}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sbuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{sb}", http.StatusOK)

	arr := h.GetStatisticsScoreboard(5)
	arr.Length().Ge(0)
}

// GET /scoreboard/graph: returns range and teams with timelines; optional top query; public.
func TestStatistics_ScoreboardGraph(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_graph")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Graph Chall", "flag{graph}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("graphuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{graph}", http.StatusOK)

	obj := h.GetScoreboardGraph(10)
	obj.ContainsKey("range")
	obj.ContainsKey("teams")
	obj.Value("teams").Array().Length().Ge(0)
}

// GET /statistics/challenges/{id}: returns challenge detail stats (id, title, category, points, solve_count, first_blood, solves); public.
func TestStatistics_ChallengeDetail_Success(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_detail")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Detail Chall", "flag{detail}", 80)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("detailuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{detail}", http.StatusOK)

	obj := h.GetStatisticsChallengesId(challengeID)
	obj.Value("id").String().IsEqual(challengeID)
	obj.Value("title").String().IsEqual("Detail Chall")
	obj.ContainsKey("category")
	obj.Value("points").Number().IsEqual(80)
	obj.Value("solve_count").Number().Ge(1)
	obj.Value("total_teams").Number().Ge(1)
	obj.ContainsKey("percentage_solved")
	obj.ContainsKey("solves")
	obj.Value("solves").Array().Length().Ge(1)
	if obj.Value("first_blood").Raw() != nil {
		obj.Value("first_blood").Object().ContainsKey("team_id")
		obj.Value("first_blood").Object().ContainsKey("team_name")
		obj.Value("first_blood").Object().ContainsKey("solved_at")
	}
}

// GET /statistics/challenges/{id}: 404 when challenge does not exist.
func TestStatistics_ChallengeDetail_NotFound(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	h.SetupCompetition("admin_detail_404")
	h.GetStatisticsChallengesIdExpectStatus(uuid.New().String(), http.StatusNotFound)
}
