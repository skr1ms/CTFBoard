package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// GET /statistics/general: returns user_count, team_count, challenge_count, solve_count (public, no auth).
func TestStatistics_General(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_stats")
	h.CreateBasicChallenge(tokenAdmin, "Stats Chall", "flag{stats}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("statsuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	resp := h.GetStatisticsGeneral()
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.UserCount)
	require.NotNil(t, resp.JSON200.TeamCount)
	require.NotNil(t, resp.JSON200.ChallengeCount)
	require.NotNil(t, resp.JSON200.SolveCount)
	require.GreaterOrEqual(t, *resp.JSON200.UserCount, 1)
	require.GreaterOrEqual(t, *resp.JSON200.TeamCount, 1)
	require.GreaterOrEqual(t, *resp.JSON200.ChallengeCount, 1)
	require.GreaterOrEqual(t, *resp.JSON200.SolveCount, 0)
}

// GET /statistics/challenges: returns array of challenge stats (id, title, points, solve_count, category); public.
func TestStatistics_Challenges(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_stats_chall")
	h.CreateBasicChallenge(tokenAdmin, "Chall A", "flag{a}", 50)
	h.CreateBasicChallenge(tokenAdmin, "Chall B", "flag{b}", 100)

	resp := h.GetStatisticsChallenges()
	require.NotNil(t, resp.JSON200)
	require.GreaterOrEqual(t, len(*resp.JSON200), 2)
	foundA, foundB := false, false
	for _, c := range *resp.JSON200 {
		if c.Title != nil {
			if *c.Title == "Chall A" {
				foundA = true
			}
			if *c.Title == "Chall B" {
				foundB = true
			}
		}
	}
	require.True(t, foundA && foundB, "expected Chall A and Chall B in statistics/challenges, found A=%v B=%v", foundA, foundB)
}

// GET /statistics/scoreboard: returns scoreboard history entries; optional limit query; public.
func TestStatistics_Scoreboard(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_stats_sb")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "SB Chall", "flag{sb}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sbuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{sb}", http.StatusOK)

	resp := h.GetStatisticsScoreboard(5)
	require.NotNil(t, resp.JSON200)
	require.GreaterOrEqual(t, len(*resp.JSON200), 0)
}

// GET /scoreboard/graph: returns range and teams with timelines; optional top query; public.
func TestStatistics_ScoreboardGraph(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_graph")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Graph Chall", "flag{graph}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("graphuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{graph}", http.StatusOK)

	resp := h.GetScoreboardGraph(10)
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Range)
	require.NotNil(t, resp.JSON200.Teams)
	require.GreaterOrEqual(t, len(*resp.JSON200.Teams), 0)
}

// GET /statistics/challenges/{id}: returns challenge detail stats (id, title, category, points, solve_count, first_blood, solves); public.
func TestStatistics_ChallengeDetail_Success(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_detail")
	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Detail Chall",
		"description":   "desc",
		"points":        80,
		"flag":          "flag{detail}",
		"category":      "misc",
		"initial_value": 80,
		"min_value":     80,
		"decay":         1,
	})

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("detailuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{detail}", http.StatusOK)

	resp := h.GetStatisticsChallengesId(challengeID)
	require.NotNil(t, resp.JSON200)
	require.Equal(t, challengeID, *resp.JSON200.ID)
	require.Equal(t, "Detail Chall", *resp.JSON200.Title)
	require.NotNil(t, resp.JSON200.Category)
	require.NotNil(t, resp.JSON200.Points)
	require.Equal(t, 80, *resp.JSON200.Points)
	require.NotNil(t, resp.JSON200.SolveCount)
	require.GreaterOrEqual(t, *resp.JSON200.SolveCount, 1)
	require.NotNil(t, resp.JSON200.TotalTeams)
	require.GreaterOrEqual(t, *resp.JSON200.TotalTeams, 1)
	require.NotNil(t, resp.JSON200.PercentageSolved)
	require.NotNil(t, resp.JSON200.Solves)
	require.GreaterOrEqual(t, len(*resp.JSON200.Solves), 1)
	if resp.JSON200.FirstBlood != nil {
		require.NotNil(t, resp.JSON200.FirstBlood.TeamID)
		require.NotNil(t, resp.JSON200.FirstBlood.TeamName)
		require.NotNil(t, resp.JSON200.FirstBlood.SolvedAt)
	}
}

// GET /statistics/challenges/{id}: 404 when challenge does not exist.
func TestStatistics_ChallengeDetail_NotFound(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	h.SetupCompetition("admin_detail_404")
	h.GetStatisticsChallengesIdExpectStatus(uuid.New().String(), http.StatusNotFound)
}
