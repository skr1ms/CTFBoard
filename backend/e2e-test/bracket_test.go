package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /brackets: returns created brackets.
func TestBracket_GetBrackets_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_brackets_list")

	suffix := uuid.New().String()[:8]
	name := "bracket_" + suffix

	createResp := h.CreateBracket(tokenAdmin, name, "desc", false, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	listResp := h.GetBrackets(http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	found := false
	for _, b := range *listResp.JSON200 {
		if b.Name != nil && *b.Name == name {
			found = true
			break
		}
	}
	require.True(t, found, "created bracket must be in /brackets list")
}

// POST /admin/brackets: non-admin gets 403.
func TestBracket_Create_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("bracket_user_" + suffix)

	h.CreateBracket(tokenUser, "x_"+suffix, "desc", false, http.StatusForbidden)
}

// PUT /admin/brackets/{id}: admin updates bracket.
func TestBracket_Update_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_bracket_upd")
	suffix := uuid.New().String()[:8]
	createResp := h.CreateBracket(tokenAdmin, "bracket_"+suffix, "desc", false, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)

	h.UpdateBracket(tokenAdmin, *createResp.JSON201.ID, "bracket_updated_"+suffix, "new desc", true, http.StatusOK)
	listResp := h.GetBrackets(http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	found := false
	for _, b := range *listResp.JSON200 {
		if b.Name != nil && *b.Name == "bracket_updated_"+suffix {
			found = true
			break
		}
	}
	require.True(t, found)
}

// DELETE /admin/brackets/{id}: admin deletes bracket.
func TestBracket_Delete_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_bracket_del")
	suffix := uuid.New().String()[:8]
	createResp := h.CreateBracket(tokenAdmin, "bracket_del_"+suffix, "desc", false, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)

	h.DeleteBracket(tokenAdmin, *createResp.JSON201.ID, http.StatusNoContent)
}

// PATCH /admin/teams/{id}/bracket: non-admin gets 403.
func TestBracket_SetTeamBracket_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_bracket_team")
	suffix := uuid.New().String()[:8]
	bracketResp := h.CreateBracket(tokenAdmin, "br_"+suffix, "d", false, http.StatusCreated)
	require.NotNil(t, bracketResp.JSON201)
	_, _, tokenUser := h.RegisterUserAndLogin("bracket_team_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)

	h.SetTeamBracket(tokenUser, *team.JSON200.ID, *bracketResp.JSON201.ID, http.StatusForbidden)
}
