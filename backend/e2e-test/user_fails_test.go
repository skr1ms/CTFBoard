package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// GET /users/{ID}/fails: admin can get another user's failed submissions.
func TestUserFails_GetByID_Admin(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("ufails_admin_" + suffix)
	challID := h.CreateBasicChallenge(tokenAdmin, "UserFails Chall "+suffix, "flag{ufails_"+suffix+"}", 100)

	_, _, tokenUser := h.RegisterUserAndLogin("ufails_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challID, "flag{wrong}", http.StatusOK)
	h.SubmitFlag(tokenUser, challID, "flag{also_wrong}", http.StatusOK)

	meResp := h.GetMe(tokenUser, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID)
	userID := *meResp.JSON200.ID

	page, perPage := 1, 20
	resp, err := h.Client().GetUsersIDFailsWithResponse(context.Background(), userID,
		&openapi.GetUsersIDFailsParams{Page: &page, PerPage: &perPage},
		helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "get user fails as admin")
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Data)
	assert.GreaterOrEqual(t, len(*resp.JSON200.Data), 2, "should have at least 2 failed submissions")
}

// GET /users/{ID}/fails: non-admin cannot access another user's fails (403).
func TestUserFails_GetByID_NonAdmin_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _ = h.SetupCompetition("ufails_forbidden_" + suffix)

	_, _, tokenUser1 := h.RegisterUserAndLogin("ufails_u1_" + suffix)
	h.CreateSoloTeam(tokenUser1, http.StatusCreated)

	_, _, tokenUser2 := h.RegisterUserAndLogin("ufails_u2_" + suffix)
	h.CreateSoloTeam(tokenUser2, http.StatusCreated)

	meResp := h.GetMe(tokenUser1, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID)
	userID := *meResp.JSON200.ID

	page, perPage := 1, 20
	resp, err := h.Client().GetUsersIDFailsWithResponse(context.Background(), userID,
		&openapi.GetUsersIDFailsParams{Page: &page, PerPage: &perPage},
		helper.WithBearerToken(tokenUser2))
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode(), "non-admin should get 403: %s", string(resp.Body))
}
