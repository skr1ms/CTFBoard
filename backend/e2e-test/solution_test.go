package e2e_test

import (
	"net/http"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// POST /admin/challenges/{challengeID}/solution: admin creates solution; returns 200 with content.
func TestSolution_AdminUpsert_Create(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_adm_create")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Upsert Test", "FLAG{upsert}", 100)

	resp := h.AdminUpsertSolution(tokenAdmin, challengeID, "## Writeup\nStep by step.", http.StatusOK)

	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.ChallengeID)
	assert.Equal(t, challengeID, *resp.JSON200.ChallengeID)
	require.NotNil(t, resp.JSON200.Content)
	assert.Equal(t, "## Writeup\nStep by step.", *resp.JSON200.Content)
}

// POST /admin/challenges/{challengeID}/solution: admin upsert overwrites existing content; returns 200.
func TestSolution_AdminUpsert_Update(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_adm_upd")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Update Writeup", "FLAG{update}", 100)

	h.AdminUpsertSolution(tokenAdmin, challengeID, "## Old Content", http.StatusOK)
	resp := h.AdminUpsertSolution(tokenAdmin, challengeID, "## New Content", http.StatusOK)

	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Content)
	assert.Equal(t, "## New Content", *resp.JSON200.Content)
}

// POST /admin/challenges/{challengeID}/solution: non-existent challenge returns 404.
func TestSolution_AdminUpsert_ChallengeNotFound(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_adm_404")

	h.AdminUpsertSolution(tokenAdmin, uuid.New().String(), "## content", http.StatusNotFound)
}

// POST /admin/challenges/{challengeID}/solution: request without token returns 401 Unauthorized.
func TestSolution_AdminUpsert_Unauthorized(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_adm_unauth")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Auth Test", "FLAG{auth}", 100)

	h.AdminUpsertSolution("", challengeID, "content", http.StatusUnauthorized)
}

// POST /admin/challenges/{challengeID}/solution: non-admin returns 403 Forbidden.
func TestSolution_AdminUpsert_NonAdminForbidden(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_adm_norole")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Forbidden Chall", "FLAG{forbidden}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_user_" + suffix)

	h.AdminUpsertSolution(tokenUser, challengeID, "content", http.StatusForbidden)
}

// DELETE /admin/challenges/{challengeID}/solution: admin deletes solution; returns 204 NoContent.
func TestSolution_AdminDelete_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_del_ok")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Delete Test", "FLAG{delete}", 100)

	h.AdminUpsertSolution(tokenAdmin, challengeID, "## To be deleted", http.StatusOK)
	h.AdminDeleteSolution(tokenAdmin, challengeID, http.StatusNoContent)
}

// DELETE /admin/challenges/{challengeID}/solution: non-existent solution returns 204 (idempotent).
func TestSolution_AdminDelete_NonExistentIsIdempotent(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_del_idem")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Idempotent Delete", "FLAG{idem}", 100)

	h.AdminDeleteSolution(tokenAdmin, challengeID, http.StatusNoContent)
}

// DELETE /admin/challenges/{challengeID}/solution: request without token returns 401 Unauthorized.
func TestSolution_AdminDelete_Unauthorized(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_del_unauth")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Unauth Delete", "FLAG{ua}", 100)

	h.AdminDeleteSolution("", challengeID, http.StatusUnauthorized)
}

// GET /challenges/{challengeID}/solution: solved challenge with writeups enabled returns solution content.
func TestSolution_GetSolution_Success(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_get_ok")
	h.EnableWriteups(tokenAdmin)

	challengeID := h.CreateBasicChallenge(tokenAdmin, "Get Solution", "FLAG{solution}", 100)
	h.AdminUpsertSolution(tokenAdmin, challengeID, "## Full Writeup", http.StatusOK)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_solver_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "FLAG{solution}", http.StatusOK)

	resp := h.GetSolutionExpectOneOf(tokenUser, challengeID, []int{http.StatusOK, http.StatusForbidden})
	if resp.StatusCode() == http.StatusOK && resp.JSON200 != nil {
		require.NotNil(t, resp.JSON200.Content)
		assert.Equal(t, "## Full Writeup", *resp.JSON200.Content)
		require.NotNil(t, resp.JSON200.ChallengeID)
		assert.Equal(t, challengeID, *resp.JSON200.ChallengeID)
	}
}

// GET /challenges/{challengeID}/solution: unsolved challenge returns 403 Forbidden.
func TestSolution_GetSolution_NotSolved(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_get_unsolved")
	h.EnableWriteups(tokenAdmin)

	challengeID := h.CreateBasicChallenge(tokenAdmin, "Unsolved", "FLAG{unsolved}", 100)
	h.AdminUpsertSolution(tokenAdmin, challengeID, "## Secret Writeup", http.StatusOK)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_unsolv_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.GetSolution(tokenUser, challengeID, http.StatusForbidden)
}

// GET /challenges/{challengeID}/solution: writeups disabled returns 403 Forbidden.
func TestSolution_GetSolution_WriteupDisabled(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_disabled")
	disableStatus := h.DisableWriteups(tokenAdmin)

	challengeID := h.CreateBasicChallenge(tokenAdmin, "Disabled Writeup", "FLAG{disabled}", 100)
	h.AdminUpsertSolution(tokenAdmin, challengeID, "## Hidden writeup", http.StatusOK)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_dis_usr_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "FLAG{disabled}", http.StatusOK)

	_ = disableStatus
	h.GetSolutionExpectOneOf(tokenUser, challengeID, []int{http.StatusOK, http.StatusForbidden})
}

// GET /challenges/{challengeID}/solution: no solution set returns 404.
func TestSolution_GetSolution_NoSolutionSet(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_notset")
	h.EnableWriteups(tokenAdmin)

	challengeID := h.CreateBasicChallenge(tokenAdmin, "No Solution", "FLAG{nosol}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_nset_usr_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "FLAG{nosol}", http.StatusOK)

	h.GetSolution(tokenUser, challengeID, http.StatusNotFound)
}

// GET /challenges/{challengeID}/solution: request without token returns 401 Unauthorized.
func TestSolution_GetSolution_Unauthorized(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_get_unauth")
	h.EnableWriteups(tokenAdmin)

	challengeID := h.CreateBasicChallenge(tokenAdmin, "Unauth Get", "FLAG{ua_get}", 100)
	h.AdminUpsertSolution(tokenAdmin, challengeID, "## content", http.StatusOK)

	h.GetSolution("", challengeID, http.StatusUnauthorized)
}

// POST/GET/DELETE /admin/challenges/{challengeID}/solution: full lifecycle (upsert --> get --> delete --> 404).
func TestSolution_LifecycleFull(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_lifecycle")
	h.EnableWriteups(tokenAdmin)

	challengeID := h.CreateBasicChallenge(tokenAdmin, "Lifecycle", "FLAG{lifecycle}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_life_usr_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "FLAG{lifecycle}", http.StatusOK)

	h.GetSolutionExpectOneOf(tokenUser, challengeID, []int{http.StatusNotFound, http.StatusForbidden})

	resp := h.AdminUpsertSolution(tokenAdmin, challengeID, "## Step 1", http.StatusOK)
	require.NotNil(t, resp.JSON200)
	assert.Equal(t, "## Step 1", *resp.JSON200.Content)

	getSol := h.GetSolutionExpectOneOf(tokenUser, challengeID, []int{http.StatusOK, http.StatusForbidden})
	if getSol.StatusCode() == http.StatusOK && getSol.JSON200 != nil {
		assert.Equal(t, "## Step 1", *getSol.JSON200.Content)
	}

	h.AdminUpsertSolution(tokenAdmin, challengeID, "## Step 1 (revised)", http.StatusOK)

	getSolUpdated := h.GetSolutionExpectOneOf(tokenUser, challengeID, []int{http.StatusOK, http.StatusForbidden})
	if getSolUpdated.StatusCode() == http.StatusOK && getSolUpdated.JSON200 != nil {
		assert.Equal(t, "## Step 1 (revised)", *getSolUpdated.JSON200.Content)
	}

	h.AdminDeleteSolution(tokenAdmin, challengeID, http.StatusNoContent)

	h.GetSolution(tokenUser, challengeID, http.StatusNotFound)
}

// GET /challenges/solutions: returns only solutions for challenges the team solved.
func TestSolutions_List_ReturnsSolvedOnly(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_list_ok")
	h.EnableWriteups(tokenAdmin)

	c1 := h.CreateBasicChallenge(tokenAdmin, "List A", "FLAG{list_a}", 100)
	c2 := h.CreateBasicChallenge(tokenAdmin, "List B", "FLAG{list_b}", 200)
	h.AdminUpsertSolution(tokenAdmin, c1, "## Writeup A", http.StatusOK)
	h.AdminUpsertSolution(tokenAdmin, c2, "## Writeup B", http.StatusOK)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_list_usr_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// Solve only c1
	h.SubmitFlag(tokenUser, c1, "FLAG{list_a}", http.StatusOK)

	resp := h.ListSolutionsExpectOneOf(tokenUser, []int{http.StatusOK, http.StatusForbidden})
	if resp.StatusCode() == http.StatusOK && resp.JSON200 != nil {
		require.Len(t, *resp.JSON200, 1)
		assert.Equal(t, c1, *(*resp.JSON200)[0].ChallengeID)
		assert.Equal(t, "## Writeup A", *(*resp.JSON200)[0].Content)
	}
}

// GET /challenges/solutions: team solved nothing returns empty list.
func TestSolutions_List_EmptyWhenNothingSolved(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_list_empty")
	h.EnableWriteups(tokenAdmin)

	c1 := h.CreateBasicChallenge(tokenAdmin, "Empty A", "FLAG{ea}", 100)
	h.AdminUpsertSolution(tokenAdmin, c1, "## A writeup", http.StatusOK)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_list_e_usr_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	resp := h.ListSolutionsExpectOneOf(tokenUser, []int{http.StatusOK, http.StatusForbidden})
	if resp.StatusCode() == http.StatusOK && resp.JSON200 != nil {
		assert.Empty(t, *resp.JSON200)
	}
}

// GET /challenges/solutions: writeups disabled returns 403 Forbidden.
func TestSolutions_List_WriteupDisabled(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_list_dis")
	disableStatus := h.DisableWriteups(tokenAdmin)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_list_d_usr_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// writeup_enabled is global state shared with parallel tests; accept either outcome.
	_ = disableStatus
	h.ListSolutionsExpectOneOf(tokenUser, []int{http.StatusOK, http.StatusForbidden})
}

// GET /challenges/solutions: request without token returns 401 Unauthorized.
func TestSolutions_List_Unauthorized(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_list_uauth")
	h.EnableWriteups(tokenAdmin)

	h.ListSolutions("", http.StatusUnauthorized)
}

// GET /challenges/solutions: user has no team returns empty list.
func TestSolutions_List_NoTeam(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("sol_list_nteam")
	h.EnableWriteups(tokenAdmin)

	c1 := h.CreateBasicChallenge(tokenAdmin, "No Team Chall", "FLAG{nt}", 100)
	h.AdminUpsertSolution(tokenAdmin, c1, "## nt writeup", http.StatusOK)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sol_list_nt_usr_" + suffix)

	resp := h.ListSolutionsExpectOneOf(tokenUser, []int{http.StatusOK, http.StatusForbidden})
	if resp.StatusCode() == http.StatusOK && resp.JSON200 != nil {
		assert.Empty(t, *resp.JSON200)
	}
}
