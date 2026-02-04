package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /challenges + POST /challenges/{ID}/submit: create challenge, submit correct flag, verify solved state; duplicate submit returns 409.
func TestChallenge_Lifecycle(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

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

// Dynamic scoring: first solver gets initial points, second gets decayed points (min_value).
func TestChallenge_DynamicScoring(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

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
func TestChallenge_CreateHIDden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

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
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

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

// POST /challenges/{ID}/submit: wrong flag returns 400 Bad Request.
func TestChallenge_SubmitInvalidFlag(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

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

	h.SubmitFlag(tokenUser, challengeID, "FLAG{wrong}", http.StatusBadRequest)
}

// DELETE /admin/challenges/{ID}: challenge is removed; GET /challenges no longer returns it.
func TestChallenge_Delete(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

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
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	resp := h.GetChallengesExpectStatus("", http.StatusUnauthorized)
	require.NotNil(t, resp.JSON401)
}

// PUT /admin/challenges/{ID}: non-existent challenge returns 404.
func TestChallenge_Update_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_up_404")
	h.UpdateChallengeExpectStatus(tokenAdmin, "00000000-0000-0000-0000-000000000000", map[string]any{
		"title": "X", "description": "Y", "points": 10, "category": "misc",
	}, http.StatusNotFound)
}

// DELETE /admin/challenges/{ID}: non-existent challenge returns 404.
func TestChallenge_Delete_NotFound(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_del_404")
	h.DeleteChallengeExpectStatus(tokenAdmin, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}
