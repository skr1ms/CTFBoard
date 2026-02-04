package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /admin/submissions/challenge/{challengeID}: submissions exist after a wrong flag submit.
func TestSubmission_AdminListByChallenge_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_ok")

	challengeID := h.CreateBasicChallenge(tokenAdmin, "Sub Challenge", "FLAG{sub}", 100)

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sub_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	h.SubmitFlag(tokenUser, challengeID, "FLAG{wrong}", http.StatusBadRequest)

	listResp := h.GetAdminSubmissionsByChallenge(tokenAdmin, challengeID, 1, 50, http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	require.NotNil(t, listResp.JSON200.Items)
	require.GreaterOrEqual(t, len(*listResp.JSON200.Items), 1)

	statsResp := h.GetAdminSubmissionStatsByChallenge(tokenAdmin, challengeID, http.StatusOK)
	require.NotNil(t, statsResp.JSON200)
	require.NotNil(t, statsResp.JSON200.Total)
	require.GreaterOrEqual(t, *statsResp.JSON200.Total, 1)
}

// GET /admin/submissions: non-admin gets 403.
func TestSubmission_AdminList_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sub_forbid_" + suffix)

	h.GetAdminSubmissions(tokenUser, 1, 50, http.StatusForbidden)
}

// GET /admin/submissions: admin gets list (may be empty).
func TestSubmission_AdminList_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_list")
	h.GetAdminSubmissions(tokenAdmin, 1, 50, http.StatusOK)
}

// GET /admin/submissions/challenge/{id}: non-admin gets 403.
func TestSubmission_AdminListByChallenge_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_ch")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "Ch", "FLAG{x}", 100)
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sub_ch_forbid_" + suffix)

	h.GetAdminSubmissionsByChallenge(tokenUser, challengeID, 1, 50, http.StatusForbidden)
}

// GET /admin/submissions/user/{id}: admin gets list by user.
func TestSubmission_AdminListByUser_Success(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_user")
	challengeID := h.CreateBasicChallenge(tokenAdmin, "ChUser", "FLAG{user}", 100)
	suffix := uuid.New().String()[:8]
	email, _, tokenUser := h.RegisterUserAndLogin("sub_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challengeID, "FLAG{wrong}", http.StatusBadRequest)
	userID := h.GetUserIDByEmail(email)

	listResp := h.GetAdminSubmissionsByUser(tokenAdmin, userID, 1, 50, http.StatusOK)
	require.NotNil(t, listResp.JSON200)
	require.NotNil(t, listResp.JSON200.Items)
	require.GreaterOrEqual(t, len(*listResp.JSON200.Items), 1)
}

// GET /admin/submissions/team/{id}: non-admin gets 403.
func TestSubmission_AdminListByTeam_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_subs_team")
	_ = h.CreateBasicChallenge(tokenAdmin, "ChTeam", "FLAG{team}", 100)
	suffix := uuid.New().String()[:8]
	_, _, tokenUser := h.RegisterUserAndLogin("sub_team_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	team := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamID := *team.JSON200.ID

	h.GetAdminSubmissionsByTeam(tokenUser, teamID, 1, 50, http.StatusForbidden)
}
