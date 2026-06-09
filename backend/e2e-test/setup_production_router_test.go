package e2e_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_ProductionRouterSetupFlow(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, truncateE2EDB(ctx, t))
	_, err := TestPool.Exec(ctx, `INSERT INTO configs (key, value, value_type, description, category)
		VALUES ('setup_complete', 'false', 'bool', 'initial setup wizard completion flag', 'general')`)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, truncateE2EDB(context.Background(), t))
		resetCompetitionToActive()
		invalidateScoreboardCache(context.Background())
	})

	deps, err := initTestDeps()
	require.NoError(t, err)

	useCases, tempStorageDir, fileStorage, err := initTestUseCases(deps)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tempStorageDir) })

	router := setupProductionTestRouter(ctx, deps.logger, useCases, deps.validator, deps.jwt, fileStorage)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client, err := openapi.NewClientWithResponses(server.URL+"/api/v1", openapi.WithHTTPClient(&http.Client{Jar: jar}))
	require.NoError(t, err)

	status, err := client.GetSetupStatusWithResponse(ctx)
	require.NoError(t, err)
	requireStatus(t, "setup status before setup", http.StatusOK, status.StatusCode(), status.Body)
	require.NotNil(t, status.JSON200)
	require.False(t, status.JSON200.Complete)

	blocked, err := client.GetCompetitionStatusWithResponse(ctx)
	require.NoError(t, err)
	requireStatus(t, "competition status before setup", http.StatusServiceUnavailable, blocked.StatusCode(), blocked.Body)

	setupReq := productionSetupRequest()
	wrongToken, err := client.PostSetupWithResponse(ctx, &openapi.PostSetupParams{XSetupToken: "wrong-token"}, setupReq)
	require.NoError(t, err)
	requireStatus(t, "setup with wrong token", http.StatusForbidden, wrongToken.StatusCode(), wrongToken.Body)

	setupResp, err := client.PostSetupWithResponse(ctx, &openapi.PostSetupParams{XSetupToken: testProductionSetupToken}, setupReq)
	require.NoError(t, err)
	requireStatus(t, "complete setup", http.StatusOK, setupResp.StatusCode(), setupResp.Body)
	require.NotNil(t, setupResp.JSON200)
	require.NotEmpty(t, setupResp.JSON200.Token)

	afterSetupStatus, err := client.GetSetupStatusWithResponse(ctx)
	require.NoError(t, err)
	requireStatus(t, "setup status after setup", http.StatusOK, afterSetupStatus.StatusCode(), afterSetupStatus.Body)
	require.NotNil(t, afterSetupStatus.JSON200)
	require.True(t, afterSetupStatus.JSON200.Complete)

	unblocked, err := client.GetCompetitionStatusWithResponse(ctx)
	require.NoError(t, err)
	requireStatus(t, "competition status after setup", http.StatusOK, unblocked.StatusCode(), unblocked.Body)

	me, err := client.GetAuthMeWithResponse(ctx, e2eBearer(setupResp.JSON200.Token))
	require.NoError(t, err)
	requireStatus(t, "setup admin /auth/me", http.StatusOK, me.StatusCode(), me.Body)

	secondSetup, err := client.PostSetupWithResponse(ctx, &openapi.PostSetupParams{XSetupToken: testProductionSetupToken}, setupReq)
	require.NoError(t, err)
	requireStatus(t, "setup after completion", http.StatusConflict, secondSetup.StatusCode(), secondSetup.Body)
}

func productionSetupRequest() openapi.PostSetupJSONRequestBody {
	maxTeamSize := 5
	verifyEmails := false
	start := time.Now().UTC().Add(-time.Hour)
	end := start.Add(24 * time.Hour)
	timezone := "UTC"

	return openapi.PostSetupJSONRequestBody{
		AccountVisibility:         openapi.SetupRequestAccountVisibilityPublic,
		AdminEmail:                e2eUID("setup_admin") + "@example.com",
		AdminPassword:             "SetupPass123!",
		AdminUsername:             e2eUID("setup_admin"),
		ChallengeVisibility:       openapi.SetupRequestChallengeVisibilityPublic,
		CtfDescription:            nil,
		CtfName:                   "Production Router E2E",
		EmailVerificationRequired: &verifyEmails,
		EndTime:                   &end,
		FreezeTime:                nil,
		MaxTeamSize:               &maxTeamSize,
		Mode:                      openapi.SetupRequestModeTeamsOnly,
		RegistrationVisibility:    openapi.SetupRequestRegistrationVisibilityPublic,
		ScoreVisibility:           openapi.SetupRequestScoreVisibilityPublic,
		StartTime:                 &start,
		Timezone:                  &timezone,
	}
}
