package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// POST /admin/awards: create bonus; GET /scoreboard reflects team score = solves + award.
func TestAward_CreateBonus_ScoreboardReflects(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_award")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Award Chall", "flag{award}", 100)

	suffix := uuid.New().String()[:8]
	teamName := "award_team_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(teamName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challengeID, "flag{award}", http.StatusOK)
	h.AssertTeamScore(tokenUser, teamName, 100)

	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))

	h.CreateAward(tokenAdmin, teamID, 50, "bonus for style", http.StatusCreated)
	h.AssertTeamScore(tokenUser, teamName, 150)
}

// POST /admin/awards: create penalty (negative value); GET /scoreboard reflects reduced score.
func TestAward_CreatePenalty_ScoreboardReflects(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_penalty")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Penalty Chall", "flag{penalty}", 100)

	suffix := uuid.New().String()[:8]
	teamName := "penalty_team_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(teamName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challengeID, "flag{penalty}", http.StatusOK)
	h.AssertTeamScore(tokenUser, teamName, 100)

	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))

	h.CreateAward(tokenAdmin, teamID, -30, "rule violation", http.StatusCreated)
	h.AssertTeamScore(tokenUser, teamName, 70)
}

// GET /admin/awards/team/{teamID}: returns list of awards for team; admin only.
func TestAward_GetByTeam(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_award_list")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("awardlist_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))

	h.CreateAward(tokenAdmin, teamID, 10, "first award", http.StatusCreated)
	h.CreateAward(tokenAdmin, teamID, -5, "penalty", http.StatusCreated)

	helper.RequireAwardsCount(t, h.GetAwardsByTeam(tokenAdmin, teamID, http.StatusOK), 2)
}

// POST /admin/awards: invalid team_id returns 400 (validation error).
func TestAward_Create_InvalidTeam(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_award_err")
	h.CreateAward(tokenAdmin, "00000000-0000-0000-0000-000000000000", 10, "bonus", http.StatusBadRequest)
}

// GET /admin/awards/team/{teamID}: non-admin gets 403 Forbidden.
func TestAward_GetByTeam_Forbidden(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_award_gf")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("award_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))

	h.GetAwardsByTeam(tokenUser, teamID, http.StatusForbidden)
}

// GET /admin/awards: admin lists all awards.
func TestAward_AdminListAll_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_awards_list")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("awards_list_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))
	h.CreateAward(tokenAdmin, teamID, 10, "list award", http.StatusCreated)

	resp, err := h.Client().GetAdminAwardsWithResponse(context.Background(), &openapi.GetAdminAwardsParams{}, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin list awards")
}

// GET /admin/awards: non-admin returns 403.
func TestAward_AdminListAll_Forbidden(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("admin_awards_list_f")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("awards_list_f_" + suffix)

	resp, err := h.Client().GetAdminAwardsWithResponse(context.Background(), &openapi.GetAdminAwardsParams{}, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "admin list awards forbidden")
}

// GET /admin/awards/{ID}: admin gets award by ID.
func TestAward_AdminGetByID_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_award_id")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("award_id_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))
	createResp := h.CreateAward(tokenAdmin, teamID, 20, "id award", http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	awardID := createResp.JSON201.ID

	resp, err := h.Client().GetAdminAwardsIDWithResponse(context.Background(), *awardID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin get award by id")
	require.NotNil(t, resp.JSON200)
	require.Equal(t, *awardID, *resp.JSON200.ID)
}

// GET /admin/awards/{ID}: not found returns 404.
func TestAward_AdminGetByID_NotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_award_id_404")

	resp, err := h.Client().GetAdminAwardsIDWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "admin get award not found")
}

// DELETE /admin/awards/{ID}: admin deletes award.
func TestAward_AdminDelete_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_award_del")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("award_del_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))
	createResp := h.CreateAward(tokenAdmin, teamID, 15, "del award", http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	awardID := createResp.JSON201.ID

	delResp, err := h.Client().DeleteAdminAwardsIDWithResponse(context.Background(), *awardID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNoContent, delResp.StatusCode(), delResp.Body, "admin delete award")
}

// DELETE /admin/awards/{ID}: not found returns 204 (idempotent delete).
func TestAward_AdminDelete_NotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_award_del_404")

	resp, err := h.Client().DeleteAdminAwardsIDWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNoContent, resp.StatusCode(), resp.Body, "admin delete award not found")
}

// GET /teams/me/awards: authed gets own team awards.
func TestAward_TeamMe_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("award_team_me")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("award_me_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))
	h.CreateAward(tokenAdmin, teamID, 5, "me award", http.StatusCreated)

	resp, err := h.Client().GetTeamsMeAwardsWithResponse(context.Background(), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get teams me awards")
}

// GET /teams/me/awards: no team returns 404.
func TestAward_TeamMe_NoTeam(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("award_team_me_no")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("award_me_nt_" + suffix)

	resp, err := h.Client().GetTeamsMeAwardsWithResponse(context.Background(), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "get teams me awards no team")
}

// GET /teams/{ID}/awards: authed gets team awards.
func TestAward_TeamByID_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("award_team_id")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("award_id_team_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	teamID := helper.RequireMyTeamOK(t, h.GetMyTeam(tokenUser, http.StatusOK))
	h.CreateAward(tokenAdmin, teamID, 7, "team id award", http.StatusCreated)

	resp, err := h.Client().GetTeamsIDAwardsWithResponse(context.Background(), teamID, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get teams id awards")
}

// GET /teams/{ID}/awards: team not found returns 404.
func TestAward_TeamByID_NotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.SetupCompetition("award_team_id_404")
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("award_id_404_" + suffix)

	resp, err := h.Client().GetTeamsIDAwardsWithResponse(context.Background(), uuid.New().String(), helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "get teams id awards not found")
}
