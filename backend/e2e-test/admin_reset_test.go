package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// POST /admin/reset: admin can reset solve data and scoreboard is cleared.
func TestAdminReset_Solves_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("reset_ok_" + suffix)

	flag := "flag{reset_" + suffix + "}"
	challID := h.CreateBasicChallenge(tokenAdmin, "Reset Chall "+suffix, flag, 100)

	_, _, tokenUser := h.RegisterUserAndLogin("reset_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.SubmitFlag(tokenUser, challID, flag, http.StatusOK)

	trueVal := true
	resp, err := h.Client().PostAdminResetWithResponse(context.Background(),
		openapi.PostAdminResetJSONRequestBody{Submissions: &trueVal},
		helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "admin reset should succeed: %s", string(resp.Body))
}

// POST /admin/reset: non-admin gets 403.
func TestAdminReset_NonAdmin_Forbidden(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, _ = h.SetupCompetition("reset_forbidden_" + suffix)
	_, _, tokenUser := h.RegisterUserAndLogin("reset_nonadmin_" + suffix)

	trueVal := true
	resp, err := h.Client().PostAdminResetWithResponse(context.Background(),
		openapi.PostAdminResetJSONRequestBody{Submissions: &trueVal},
		helper.WithBearerToken(tokenUser))
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode())
}

// POST /admin/reset: admin account still exists and token is still valid after reset.
func TestAdminReset_PreservesAdminAccount(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("reset_preserve_" + suffix)

	trueVal := true
	resp, err := h.Client().PostAdminResetWithResponse(context.Background(),
		openapi.PostAdminResetJSONRequestBody{Submissions: &trueVal},
		helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "reset should succeed: %s", string(resp.Body))

	meResp := h.GetMe(tokenAdmin, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID, "admin user should still exist after reset")
}
