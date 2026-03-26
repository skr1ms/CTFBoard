package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// GET /healthcheck: returns 200 OK.
func TestStatic_Healthcheck(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().GetHealthcheckWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "healthcheck")
}

// GET /robots.txt: returns 200 with body.
func TestStatic_RobotsTxt(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().GetRobotsTxtWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "robots.txt")
	require.NotEmpty(t, resp.Body)
}

// GET /tos: returns 200.
func TestStatic_Tos(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().GetTosWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "tos")
}

// GET /privacy: returns 200.
func TestStatic_Privacy(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().GetPrivacyWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "privacy")
}

// GET /debug: admin-only; returns 200 when DEBUG_ENABLED=true, otherwise 404. Without auth returns 401.
func TestStatic_Debug(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("debug_test")
	resp, err := h.Client().GetDebugWithResponse(context.Background(), helper.WithBearerToken(tokenAdmin))
	require.NoError(t, err)

	if resp.StatusCode() == http.StatusNotFound {
		t.Skip("debug endpoint not enabled (DEBUG_ENABLED != true)")
	}

	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "debug")
}

// GET /challenges/types: returns list of challenge types (public endpoint).
func TestStatic_ChallengesTypes(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().GetChallengesTypesWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusOK, resp.StatusCode(), resp.Body, "challenges types")
}
