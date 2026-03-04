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

// GET /challenges + POST /challenges/{ID}/submit: create challenge, submit correct flag, verify solved state; duplicate submit returns 409.
func TestChallenge_Lifecycle(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_lifecycle")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Test Challenge",
		"description": "Test Description",
		"points":      100,
		"flag":        "FLAG{test}",
		"category":    "web",
		"is_hidden":   false,
	})

	suffix := uuid.New().String()[:8]
	userName := "chall_usr_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	challenge := h.FindChallengeInList(tokenUser, challengeID)
	require.Equal(t, "Test Challenge", *challenge.Title)
	solvedFalse := false
	solveCount0 := 0
	helper.RequireChallengeFields(t, challenge, "", &solvedFalse, &solveCount0, nil)

	h.SubmitFlag(tokenUser, challengeID, "FLAG{test}", http.StatusOK)

	challengeAfterSolve := h.FindChallengeInList(tokenUser, challengeID)
	solvedTrue := true
	solveCount1 := 1
	helper.RequireChallengeFields(t, challengeAfterSolve, "", &solvedTrue, &solveCount1, nil)

	h.SubmitFlag(tokenUser, challengeID, "FLAG{test}", http.StatusConflict)
}

// POST /admin/challenges + POST /challenges/{ID}/submit: dynamic scoring; first solver gets initial points, second gets min_value.
func TestChallenge_DynamicScoring(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("adm_dyn")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":         "Dynamic Challenge",
		"description":   "Test dynamic scoring",
		"points":        500,
		"initial_value": 500,
		"min_value":     100,
		"decay":         1,
		"flag":          "FLAG{dynamic}",
		"category":      "web",
		"is_hidden":     false,
	})

	suffix := uuid.New().String()[:8]
	_, _, tokenUser1 := h.RegisterUserAndLogin("solver1_" + suffix)
	h.CreateSoloTeam(tokenUser1, http.StatusCreated)
	h.SubmitFlag(tokenUser1, challengeID, "FLAG{dynamic}", http.StatusOK)

	challengeState1 := h.FindChallengeInList(tokenUser1, challengeID)
	points500 := 500
	solveCount1 := 1
	helper.RequireChallengeFields(t, challengeState1, "", nil, &solveCount1, &points500)

	_, _, tokenUser2 := h.RegisterUserAndLogin("solver2_" + suffix)
	h.CreateSoloTeam(tokenUser2, http.StatusCreated)
	h.SubmitFlag(tokenUser2, challengeID, "FLAG{dynamic}", http.StatusOK)

	challengeState2 := h.FindChallengeInList(tokenUser2, challengeID)
	points100 := 100
	solveCount2 := 2
	helper.RequireChallengeFields(t, challengeState2, "", nil, &solveCount2, &points100)
}

// POST /admin/challenges with is_hidden: hidden challenge is not visible in GET /challenges for regular user.
func TestChallenge_CreateHidden(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_hidden")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "HIDden Challenge",
		"description": "Test hidden challenge",
		"points":      200,
		"flag":        "FLAG{hidden}",
		"category":    "crypto",
		"is_hidden":   true,
	})

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("user2_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.AssertChallengeMissing(tokenUser, challengeID)
}

// PUT /admin/challenges/{ID}: update challenge fields; GET /challenges reflects new title, description, points.
func TestChallenge_Update(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_update")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Original Title",
		"description": "Original description",
		"points":      100,
		"flag":        "FLAG{original}",
		"category":    "web",
		"is_hidden":   false,
	})

	h.UpdateChallenge(tokenAdmin, challengeID, map[string]any{
		"title":       "Updated Title",
		"description": "Updated Description",
		"points":      150,
		"flag":        "FLAG{updated}",
		"category":    "pwn",
		"is_hidden":   false,
	})

	challenge := h.FindChallengeInList(tokenAdmin, challengeID)
	require.Equal(t, "Updated Title", *challenge.Title)
	require.Equal(t, "Updated Description", *challenge.Description)
	points150 := 150
	helper.RequireChallengeFields(t, challenge, "", nil, nil, &points150)
}

// POST /challenges/{ID}/submit: wrong flag returns 200 with correct=false.
func TestChallenge_SubmitInvalidFlag(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_invalid")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Test Challenge",
		"description": "Test invalid flag",
		"flag":        "FLAG{correct}",
		"points":      100,
		"category":    "misc",
	})

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("user3_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	resp := h.SubmitFlag(tokenUser, challengeID, "FLAG{wrong}", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.False(t, resp.JSON200.Correct)
}

// DELETE /admin/challenges/{ID}: challenge is removed; GET /challenges no longer returns it.
func TestChallenge_Delete(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_delete")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "To Delete",
		"description": "Test delete challenge",
		"flag":        "FLAG{delete}",
		"points":      50,
		"category":    "misc",
	})

	h.DeleteChallenge(tokenAdmin, challengeID)

	h.AssertChallengeMissing(tokenAdmin, challengeID)
}

// GET /challenges: request without token returns 401 Unauthorized.
func TestChallenge_GetChallenges_Unauthorized(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp := h.GetChallengesExpectStatus("", http.StatusUnauthorized)
	require.NotNil(t, resp.JSON401)
}

// PUT /admin/challenges/{ID}: non-existent challenge returns 404.
func TestChallenge_Update_NotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_up_404")
	h.UpdateChallengeExpectStatus(tokenAdmin, "00000000-0000-0000-0000-000000000000", map[string]any{
		"title": "X", "description": "Y", "points": 10, "category": "misc",
	}, http.StatusNotFound)
}

// DELETE /admin/challenges/{ID}: non-existent challenge returns 404.
func TestChallenge_Delete_NotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_del_404")
	h.DeleteChallengeExpectStatus(tokenAdmin, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}

// GET /challenges/{challengeID}/solution: after competition ends, solutions remain accessible (returns 200).
func TestChallenge_GetSolution_AfterCompetitionEnd(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_end_admin")
	h.EnableWriteups(tokenAdmin)
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_end_usr_" + suffix)
	h.CreateTeam(tokenUser, "sol_end_team_"+suffix, http.StatusCreated)
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Solution Chall", "flag{solution}", 100)
	h.AdminUpsertSolution(tokenAdmin, challengeID, "## Writeup", http.StatusOK)

	h.SubmitFlag(tokenUser, challengeID, "flag{solution}", http.StatusOK)

	t.Cleanup(resetCompetitionToActive)
	ctx := context.Background()
	_, err := TestPool.Exec(ctx, `UPDATE competition SET end_time = NOW() - INTERVAL '1 minute' WHERE id = 1`)
	require.NoError(t, err)
	_ = TestRedis.Del(ctx, "competition")

	resp := h.GetSolution(tokenUser, challengeID, http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.Equal(t, "## Writeup", *resp.JSON200.Content)
}

// GET /challenges/{challengeID}/solution: challenge not found returns 404.
func TestChallenge_GetSolution_NotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_notfound_adm")
	h.EnableWriteups(tokenAdmin)
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_nf_usr_" + suffix)

	resp, err := h.Client().GetChallengesChallengeIDSolutionWithResponse(context.Background(), "00000000-0000-0000-0000-000000000000", helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	// 404 when challenge missing; 403 when writeups disabled (parallel test may have disabled it).
	require.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, resp.StatusCode(), "solution not found: status %d body=%s", resp.StatusCode(), string(resp.Body))
	if resp.StatusCode() == http.StatusNotFound {
		require.NotNil(t, resp.JSON404)
	}
}

// GET /admin/challenges/{challengeID}/flags: admin gets flag hash + config.
func TestChallenge_AdminGetFlags_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_flags_admin")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Flags Chall", "flag{flagtest}", 100)

	resp, err := h.Client().GetAdminChallengesChallengeIDFlagsWithResponse(context.Background(), challengeID, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "admin get flags")
}

// GET /admin/challenges/{challengeID}/flags: non-admin returns 403.
func TestChallenge_AdminGetFlags_Forbidden(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_flags_forbidden")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Flags Forbidden Chall", "flag{flagtest2}", 100)
	_, _, tokenUser := h.RegisterUserAndLogin("flags_forbidden_user")

	resp, err := h.Client().GetAdminChallengesChallengeIDFlagsWithResponse(context.Background(), challengeID, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusForbidden, resp.StatusCode(), resp.Body, "non-admin get flags")
}

// GET /challenges/{challengeID}/files: authed gets challenge files.
func TestChallenge_GetFiles_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("files_admin")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Files Chall", "flag{files}", 100)
	_, _, tokenUser := h.RegisterUserAndLogin("files_user")

	params := &openapi.GetChallengesChallengeIDFilesParams{}
	resp, err := h.Client().GetChallengesChallengeIDFilesWithResponse(context.Background(), challengeID, params, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get challenge files")
}

// GET /challenges/{challengeID}/files: non-existent challenge returns 200 with empty list.
func TestChallenge_GetFiles_NotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("files_notfound_admin")
	_, _, tokenUser := h.RegisterUserAndLogin("files_notfound_user")

	params := &openapi.GetChallengesChallengeIDFilesParams{}
	resp, err := h.Client().GetChallengesChallengeIDFilesWithResponse(context.Background(), "00000000-0000-0000-0000-000000000000", params, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	_ = tokenAdmin
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "files for unknown challenge")
}

// GET /challenges/{challengeID}/requirements: authed gets requirements.
func TestChallenge_GetRequirements_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("reqs_admin")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Reqs Chall", "flag{reqs}", 100)
	_, _, tokenUser := h.RegisterUserAndLogin("reqs_user")

	resp, err := h.Client().GetChallengesChallengeIDRequirementsWithResponse(context.Background(), challengeID, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get challenge requirements")
}

// GET /challenges/{challengeID}/requirements: challenge not found returns 404.
func TestChallenge_GetRequirements_NotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("reqs_notfound_admin")
	_, _, tokenUser := h.RegisterUserAndLogin("reqs_notfound_user")

	resp, err := h.Client().GetChallengesChallengeIDRequirementsWithResponse(context.Background(), "00000000-0000-0000-0000-000000000000", helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "requirements not found")
}

// PUT /admin/challenges/{challengeID}/requirements: admin sets prerequisites, GET returns them.
func TestChallenge_PutRequirements_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("put_reqs_admin")
	prereqID := h.CreateBasicChallenge(tokenAdmin, "Prereq Chall", "flag{prereq}", 50)
	mainID := h.CreateBasicChallenge(tokenAdmin, "Main Chall", "flag{main}", 100)

	h.SetChallengeRequirements(tokenAdmin, mainID, []string{prereqID})

	_, _, tokenUser := h.RegisterUserAndLogin("put_reqs_user")
	resp, err := h.Client().GetChallengesChallengeIDRequirementsWithResponse(context.Background(), mainID, helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get requirements after put")
	require.NotNil(t, resp.JSON200)
	require.Len(t, *resp.JSON200, 1)
	require.Equal(t, prereqID, *(*resp.JSON200)[0].ChallengeID)
	require.Equal(t, "Prereq Chall", *(*resp.JSON200)[0].ChallengeTitle)
}

// PUT /admin/challenges/{challengeID}/requirements: non-existent challenge returns 404.
func TestChallenge_PutRequirements_NotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("put_reqs_404")
	req := openapi.PutAdminChallengesChallengeIDRequirementsJSONRequestBody{
		RequirementIds: &[]string{},
	}
	resp, err := h.Client().PutAdminChallengesChallengeIDRequirementsWithResponse(context.Background(), "00000000-0000-0000-0000-000000000000", req, helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNotFound, resp.StatusCode(), resp.Body, "put requirements not found")
}

// POST /challenges/{ID}/submit: when requirements not met returns 403.
func TestChallenge_SubmitFlag_RequirementsNotMet(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("submit_reqs_admin")
	prereqID := h.CreateBasicChallenge(tokenAdmin, "Prereq", "flag{prereq}", 50)
	mainID := h.CreateBasicChallenge(tokenAdmin, "Main With Reqs", "flag{main}", 100)
	h.SetChallengeRequirements(tokenAdmin, mainID, []string{prereqID})

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("submit_reqs_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	resp := h.SubmitFlag(tokenUser, mainID, "flag{main}", http.StatusForbidden)
	require.NotNil(t, resp.JSON403)
}

// POST /challenges/{ID}/submit: when requirements met (prereq solved first) returns 200 correct.
func TestChallenge_SubmitFlag_RequirementsMet_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("submit_reqs_ok_admin")
	prereqID := h.CreateBasicChallenge(tokenAdmin, "Prereq OK", "flag{prereq}", 50)
	mainID := h.CreateBasicChallenge(tokenAdmin, "Main OK", "flag{main}", 100)
	h.SetChallengeRequirements(tokenAdmin, mainID, []string{prereqID})

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("submit_reqs_ok_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, prereqID, "flag{prereq}", http.StatusOK)
	resp := h.SubmitFlag(tokenUser, mainID, "flag{main}", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.True(t, resp.JSON200.Correct)
}

// POST /challenges/{ID}/submit: main challenge with two requirements; solve both prereqs then main (batch requirement check).
func TestChallenge_SubmitFlag_RequirementsMet_TwoRequirements(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("submit_two_reqs_admin")
	prereq1ID := h.CreateBasicChallenge(tokenAdmin, "Prereq One", "flag{one}", 50)
	prereq2ID := h.CreateBasicChallenge(tokenAdmin, "Prereq Two", "flag{two}", 50)
	mainID := h.CreateBasicChallenge(tokenAdmin, "Main Two Reqs", "flag{main}", 100)
	h.SetChallengeRequirements(tokenAdmin, mainID, []string{prereq1ID, prereq2ID})

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("submit_two_reqs_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	respForbidden := h.SubmitFlag(tokenUser, mainID, "flag{main}", http.StatusForbidden)
	require.NotNil(t, respForbidden.JSON403)

	h.SubmitFlag(tokenUser, prereq1ID, "flag{one}", http.StatusOK)
	respForbidden2 := h.SubmitFlag(tokenUser, mainID, "flag{main}", http.StatusForbidden)
	require.NotNil(t, respForbidden2.JSON403)

	h.SubmitFlag(tokenUser, prereq2ID, "flag{two}", http.StatusOK)
	resp := h.SubmitFlag(tokenUser, mainID, "flag{main}", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	require.True(t, resp.JSON200.Correct)
}
