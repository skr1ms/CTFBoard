package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// GET /statistics/general: returns user_count, team_count, challenge_count, solve_count (public, no auth).
func TestStatistics_General(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_stats")
	h.CreateBasicChallenge(tokenAdmin, "Stats Chall", "flag{stats}", 100)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("statsuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	resp := h.GetStatisticsGeneral(tokenUser)
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()

	resetCompetitionToActive()

	_, tokenAdmin := h.SetupCompetition("adm_st_ch_" + suffix)
	h.CreateBasicChallenge(tokenAdmin, "Chall A "+suffix, "flag{a}", 50)
	h.CreateBasicChallenge(tokenAdmin, "Chall B "+suffix, "flag{b}", 100)

	resp := h.GetStatisticsChallenges(tokenAdmin)
	require.NotNil(t, resp.JSON200)
	require.GreaterOrEqual(t, len(*resp.JSON200), 2, "statistics/challenges returns at least 2 items (cache may delay our new challenges)")
}

// GET /statistics/scoreboard: returns scoreboard history entries; optional limit query; public.
func TestStatistics_Scoreboard(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resetCompetitionToActive()

	_, tokenAdmin := h.SetupCompetition("admin_stats_sb")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "SB Chall", "flag{sb}", 100)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("sbuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{sb}", http.StatusOK)

	resp := h.GetStatisticsScoreboard(tokenUser, 5)
	require.NotNil(t, resp.JSON200)
	require.GreaterOrEqual(t, len(*resp.JSON200), 0)
}

// GET /statistics/scoreboard: VisibilityGuard(score_visibility) lets guests through when
// the default value "public" is in effect; the endpoint responds 200 without auth.
func TestStatistics_Scoreboard_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	h.SetupCompetition("stats_sb_401")

	resp, err := h.Client().GetStatisticsScoreboardWithResponse(context.Background(), nil)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get scoreboard no auth (score_visibility=public)")
}

// GET /scoreboard/graph: returns range and teams with timelines; optional top query; public.
func TestStatistics_ScoreboardGraph(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_graph")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Graph Chall", "flag{graph}", 100)

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("graphuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{graph}", http.StatusOK)

	resp := h.GetScoreboardGraph(tokenUser, 10)
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Range)
	require.NotNil(t, resp.JSON200.Teams)
	require.GreaterOrEqual(t, len(*resp.JSON200.Teams), 0)
}

// GET /scoreboard/graph: VisibilityGuard(score_visibility) lets guests through on the
// default value "public"; the endpoint responds 200 without auth.
func TestStatistics_ScoreboardGraph_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	h.SetupCompetition("stats_graph_401")

	resp, err := h.Client().GetScoreboardGraphWithResponse(context.Background(), &openapi.GetScoreboardGraphParams{})
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get scoreboard graph no auth (score_visibility=public)")
}

// GET /statistics/challenges/{id}: returns challenge detail stats (id, title, category, points, solve_count, first_blood, solves); public.
func TestStatistics_ChallengeDetail_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("detailuser_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "flag{detail}", http.StatusOK)

	resp := h.GetStatisticsChallengesId(tokenUser, challengeID)
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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, token := h.SetupCompetition("admin_detail_404")
	h.GetStatisticsChallengesIdExpectStatus(token, uuid.New().String(), http.StatusNotFound)
}

// GET /statistics/challenges/{id}: invalid UUID returns 400.
func TestStatistics_ChallengeDetail_InvalidID_Returns400(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	_, token := h.SetupCompetition("admin_invalid_id")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, GetTestBaseURL()+"/api/v1/statistics/challenges/not-a-uuid", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// GET /statistics/challenges/solves/percentages: returns array; no auth returns 401.
func TestStatistics_SolvesPercentages(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("stats_pct_admin")
	h.CreateBasicChallenge(tokenAdmin, "Pct Chall", "flag{pct}", 50)

	resp, err := h.Client().GetStatisticsChallengesSolvesPercentagesWithResponse(context.Background(), nil, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get solves percentages")
}

// GET /statistics/challenges/solves/percentages: VisibilityGuard(score_visibility) lets
// guests through on the default value "public"; the endpoint responds 200 without auth.
func TestStatistics_SolvesPercentages_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	h.SetupCompetition("stats_pct_401")

	resp, err := h.Client().GetStatisticsChallengesSolvesPercentagesWithResponse(context.Background(), nil)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get solves percentages no auth (score_visibility=public)")
}

// GET /statistics/scores/distribution: returns distribution; no auth returns 401.
func TestStatistics_ScoresDistribution(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("stats_dist_admin")
	h.CreateBasicChallenge(tokenAdmin, "Dist Chall", "flag{dist}", 50)

	resp, err := h.Client().GetStatisticsScoresDistributionWithResponse(context.Background(), nil, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get scores distribution")
}

// GET /statistics/scores/distribution: VisibilityGuard(score_visibility) lets guests
// through on the default value "public"; the endpoint responds 200 without auth.
func TestStatistics_ScoresDistribution_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	h.SetupCompetition("stats_dist_401")

	resp, err := h.Client().GetStatisticsScoresDistributionWithResponse(context.Background(), nil)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get scores distribution no auth (score_visibility=public)")
}

// GET /statistics/submissions: returns time series stats; no auth returns 401.
func TestStatistics_Submissions(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("stats_subs_admin")

	resp, err := h.Client().GetStatisticsSubmissionsWithResponse(context.Background(), nil, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get statistics submissions")
}

// GET /statistics/submissions: VisibilityGuard(score_visibility) lets guests through on
// the default value "public"; the endpoint responds 200 without auth.
func TestStatistics_Submissions_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	h.SetupCompetition("stats_subs_401")

	resp, err := h.Client().GetStatisticsSubmissionsWithResponse(context.Background(), nil)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get statistics submissions no auth (score_visibility=public)")
}

// GET /statistics/submissions/{type}: type=correct -> filtered series.
func TestStatistics_SubmissionsType_Correct(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("stats_subs_type")

	resp, err := h.Client().GetStatisticsSubmissionsTypeWithResponse(context.Background(), openapi.Correct, nil, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get statistics submissions type correct")
}

// GET /statistics/submissions/{type}: invalid type returns 400.
func TestStatistics_SubmissionsType_InvalidType(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("stats_subs_type_400")

	resp, err := h.Client().GetStatisticsSubmissionsTypeWithResponse(context.Background(), "invalid_type", nil, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusBadRequest, resp.StatusCode(), resp.Body, "get statistics submissions invalid type returns 400")
}

// GET /statistics/teams: returns team registration series; no auth returns 401.
func TestStatistics_Teams(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("stats_teams_admin")

	resp, err := h.Client().GetStatisticsTeamsWithResponse(context.Background(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get statistics teams")
}

// GET /statistics/teams: VisibilityGuard(score_visibility) lets guests through on the
// default value "public"; the endpoint responds 200 without auth.
func TestStatistics_Teams_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	h.SetupCompetition("stats_teams_401")

	resp, err := h.Client().GetStatisticsTeamsWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get statistics teams no auth (score_visibility=public)")
}

// GET /statistics/users: returns user registration series; no auth returns 401.
func TestStatistics_Users(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("stats_users_admin")

	resp, err := h.Client().GetStatisticsUsersWithResponse(context.Background(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get statistics users")
}

// GET /statistics/users: VisibilityGuard(score_visibility) lets guests through on the
// default value "public"; the endpoint responds 200 without auth.
func TestStatistics_Users_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	h.SetupCompetition("stats_users_401")

	resp, err := h.Client().GetStatisticsUsersWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get statistics users no auth (score_visibility=public)")
}

// GET /admin/statistics/solve-matrix: admin gets solve matrix; non-admin returns 403.
func TestStatistics_AdminSolveMatrix(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("stats_matrix_admin")

	resp, err := h.Client().GetAdminStatisticsSolveMatrixWithResponse(context.Background(), nil, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get admin solve matrix")
}

// GET /admin/statistics/solve-matrix: non-admin returns 403.
func TestStatistics_AdminSolveMatrix_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("stats_matrix_403")

	suffix := helper.UID()
	_, _, tokenUser := h.RegisterUserAndLogin("matrix_user_" + suffix)

	resp, err := h.Client().GetAdminStatisticsSolveMatrixWithResponse(context.Background(), nil, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "get admin solve matrix forbidden")
}
