package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// TestBanFlow_UserBanBlocksSubmit verifies the full ban cycle: admin bans user -> submit returns 401
// (token revoked) -> admin unbans -> user re-logs in -> submit succeeds.
func TestBanFlow_UserBanBlocksSubmit(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("banflow_" + suffix)

	flag := "flag{ban_flow_" + suffix + "}"
	challID := h.CreateBasicChallenge(tokenAdmin, "Ban Flow Challenge "+suffix, flag, 100)

	_, _, tokenUser := h.RegisterUserAndLogin("banflow_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// Confirm submit works before ban
	h.SubmitFlag(tokenUser, challID, flag, http.StatusOK)

	// Get user ID
	me := h.GetMe(tokenUser, http.StatusOK)
	require.NotNil(t, me.JSON200)
	require.NotNil(t, me.JSON200.ID)
	userID := *me.JSON200.ID

	// Admin bans the user - RevokeAllForUser stores revokedAt=now() in Redis
	h.BanUser(tokenAdmin, userID, "ban flow test", http.StatusOK)
	h.InvalidateUserCache(userID)

	// After ban, existing token is revoked - submit returns 401
	challID2 := h.CreateBasicChallenge(tokenAdmin, "Ban Flow Challenge2 "+suffix, "flag{second_"+suffix+"}", 100)
	require.NotEmpty(t, challID2)
	h.SubmitFlag(tokenUser, challID2, "flag{second_"+suffix+"}", http.StatusUnauthorized)

	// Admin unbans the user
	h.UnbanUser(tokenAdmin, userID, http.StatusNoContent)

	// Wait 1s: jwtkit IsUserRevoked checks iat<=revokedAt; a token issued in the same
	// second as the ban would be falsely rejected, so we ensure iat > revokedAt.
	time.Sleep(time.Second)

	// User logs in again to get a fresh token after ban
	email := "banflow_user_" + suffix + "@example.com"
	loginResp := h.Login(email, "ValidPass1", http.StatusOK)
	require.NotNil(t, loginResp.JSON200)
	require.NotEmpty(t, loginResp.JSON200.AccessToken)
	freshToken := "Bearer " + *loginResp.JSON200.AccessToken

	// After unban with a fresh token, submit succeeds
	h.SubmitFlag(freshToken, challID2, "flag{second_"+suffix+"}", http.StatusOK)
}

// TestBanFlow_TeamBanBlocksSubmitAndUnban verifies: ban team -> member submit blocked ->
// unban -> member submit passes.
func TestBanFlow_TeamBanBlocksSubmitAndUnban(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("tbanflow_" + suffix)

	flag := "flag{team_ban_" + suffix + "}"
	challID := h.CreateBasicChallenge(tokenAdmin, "Team Ban Flow "+suffix, flag, 100)

	// Captain creates team, member joins
	_, _, tokenCap := h.RegisterUserAndLogin("tbanflow_cap_" + suffix)
	h.CreateTeam(tokenCap, "BanFlowTeam_"+suffix, http.StatusCreated)
	myTeam := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, myTeam.JSON200)
	teamID := *myTeam.JSON200.ID
	inviteToken := *myTeam.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin("tbanflow_mem_" + suffix)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	// Confirm submit works before ban
	h.SubmitFlag(tokenCap, challID, flag, http.StatusOK)

	// Admin bans the team
	h.BanTeam(tokenAdmin, teamID, "team ban flow test", http.StatusOK)

	// Create second challenge after ban to avoid "already solved" responses
	challID2 := h.CreateBasicChallenge(tokenAdmin, "Team Ban Flow2 "+suffix, "flag{tb2_"+suffix+"}", 200)

	// After ban, captain and member submits are blocked
	h.SubmitFlag(tokenCap, challID2, "flag{tb2_"+suffix+"}", http.StatusForbidden)
	h.SubmitFlag(tokenMember, challID2, "flag{tb2_"+suffix+"}", http.StatusForbidden)

	// Admin unbans - team ban state is cached; wait for cache propagation
	h.UnbanTeam(tokenAdmin, teamID, http.StatusNoContent)

	require.Eventually(t, func() bool {
		resp, err := h.Client().PostChallengesChallengeIDSubmitWithResponse(
			context.Background(), challID2,
			openapi.PostChallengesChallengeIDSubmitJSONRequestBody{Flag: "flag{tb2_" + suffix + "}"},
			helper.WithBearerToken(tokenCap))

		return err == nil && resp != nil && resp.StatusCode() == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond, "captain submit should succeed after team unban")
}

// TestBanFlow_BannedUserCannotViewProtectedResources verifies that a banned user
// cannot access protected resources (hint unlock).
func TestBanFlow_BannedUserCannotViewProtectedResources(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("banprot_" + suffix)

	challID := h.CreateBasicChallenge(tokenAdmin, "Ban Prot Challenge "+suffix, "flag{bp_"+suffix+"}", 100)
	hintID := h.CreateHint(tokenAdmin, challID, "pre-ban hint", 0)

	_, _, tokenUser := h.RegisterUserAndLogin("banprot_user_" + suffix)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	// Can unlock hint before ban
	h.UnlockHint(tokenUser, challID, hintID, http.StatusOK)

	me := h.GetMe(tokenUser, http.StatusOK)
	require.NotNil(t, me.JSON200)
	userID := *me.JSON200.ID

	h.BanUser(tokenAdmin, userID, "protection test", http.StatusOK)
	h.InvalidateUserCache(userID)

	// After ban token is revoked - hint unlock returns 401
	hintID2 := h.CreateHint(tokenAdmin, challID, "post-ban hint", 0)
	h.UnlockHint(tokenUser, challID, hintID2, http.StatusUnauthorized)
}
