package e2e_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func e2eRawGet(t *testing.T, rawURL string, editors ...openapi.RequestEditorFn) (int, []byte) {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, http.NoBody)
	require.NoError(t, err)

	for _, edit := range editors {
		require.NoError(t, edit(context.Background(), req))
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, body
}

func e2eAbsoluteURL(t *testing.T, rawURL string) string {
	t.Helper()

	if strings.HasPrefix(rawURL, "/") {
		return GetTestBaseURL() + rawURL
	}

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)

	if !parsed.IsAbs() {
		return GetTestBaseURL() + "/" + strings.TrimPrefix(rawURL, "/")
	}

	base, err := url.Parse(GetTestBaseURL())
	require.NoError(t, err)

	parsed.Scheme = base.Scheme
	parsed.Host = base.Host

	return parsed.String()
}

func TestE2E_AuthSessionLifecycle(t *testing.T) {
	s := newE2ESuite(t)

	user := s.registerUser("session_user")

	refresh, err := s.client.PostAuthRefreshWithResponse(context.Background())
	require.NoError(t, err)
	requireStatus(t, "refresh session", http.StatusOK, refresh.StatusCode(), refresh.Body)
	require.NotNil(t, refresh.JSON200)
	require.NotNil(t, refresh.JSON200.AccessToken)
	require.NotEqual(t, user.Token, *refresh.JSON200.AccessToken)

	me, err := s.client.GetAuthMeWithResponse(context.Background(), e2eBearer(*refresh.JSON200.AccessToken))
	require.NoError(t, err)
	requireStatus(t, "me with refreshed token", http.StatusOK, me.StatusCode(), me.Body)
	require.NotNil(t, me.JSON200)
	require.Equal(t, user.UserID, *me.JSON200.ID)

	logout, err := s.client.PostAuthLogoutWithResponse(context.Background(), e2eLowercaseBearer(*refresh.JSON200.AccessToken))
	require.NoError(t, err)
	requireStatus(t, "logout session", http.StatusNoContent, logout.StatusCode(), logout.Body)

	afterLogoutRefresh, err := s.client.PostAuthRefreshWithResponse(context.Background())
	require.NoError(t, err)
	requireStatus(t, "refresh after logout", http.StatusUnauthorized, afterLogoutRefresh.StatusCode(), afterLogoutRefresh.Body)

	afterLogoutMe, err := s.client.GetAuthMeWithResponse(context.Background(), e2eBearer(*refresh.JSON200.AccessToken))
	require.NoError(t, err)
	requireStatus(t, "me after logout", http.StatusUnauthorized, afterLogoutMe.StatusCode(), afterLogoutMe.Body)
}

func TestE2E_ProfilePatchRequiresBearerAndPasswordChangeReturnsFreshToken(t *testing.T) {
	s := newE2ESuite(t)

	user := s.registerUser("profile_rotate_user")

	description := "profile mutation regression"
	apiToken, err := s.client.PostUserTokensWithResponse(context.Background(), openapi.PostUserTokensJSONRequestBody{
		Description: &description,
	}, e2eBearer(user.Token))
	require.NoError(t, err)
	requireStatus(t, "create api token", http.StatusCreated, apiToken.StatusCode(), apiToken.Body)
	require.NotNil(t, apiToken.JSON201)
	require.NotEmpty(t, apiToken.JSON201.Token)

	apiTokenUsername := e2eUID("api_token_patch")
	apiTokenPatch, err := s.client.PatchAuthMeWithResponse(context.Background(), openapi.PatchAuthMeJSONRequestBody{
		Username: &apiTokenUsername,
	}, e2eAPIToken(apiToken.JSON201.Token))
	require.NoError(t, err)
	requireStatus(t, "profile patch with api token", http.StatusForbidden, apiTokenPatch.StatusCode(), apiTokenPatch.Body)

	currentPassword := e2ePassword
	newPassword := "ValidPass2"
	passwordPatch, err := s.client.PatchAuthMeWithResponse(context.Background(), openapi.PatchAuthMeJSONRequestBody{
		CurrentPassword: &currentPassword,
		Password:        &newPassword,
	}, e2eBearer(user.Token))
	require.NoError(t, err)
	requireStatus(t, "profile password patch", http.StatusOK, passwordPatch.StatusCode(), passwordPatch.Body)
	require.NotNil(t, passwordPatch.JSON200)
	require.NotNil(t, passwordPatch.JSON200.TokenPair)
	require.NotNil(t, passwordPatch.JSON200.TokenPair.AccessToken)
	require.NotEqual(t, user.Token, *passwordPatch.JSON200.TokenPair.AccessToken)

	oldTokenMe, err := s.client.GetAuthMeWithResponse(context.Background(), e2eBearer(user.Token))
	require.NoError(t, err)
	requireStatus(t, "me with old token after password change", http.StatusUnauthorized, oldTokenMe.StatusCode(), oldTokenMe.Body)

	newTokenMe, err := s.client.GetAuthMeWithResponse(context.Background(), e2eBearer(*passwordPatch.JSON200.TokenPair.AccessToken))
	require.NoError(t, err)
	requireStatus(t, "me with fresh token after password change", http.StatusOK, newTokenMe.StatusCode(), newTokenMe.Body)
	require.NotNil(t, newTokenMe.JSON200)
	require.NotNil(t, newTokenMe.JSON200.ID)
	require.Equal(t, user.UserID, *newTokenMe.JSON200.ID)
}

func TestE2E_APITokenLifecycle(t *testing.T) {
	s := newE2ESuite(t)

	user := s.registerUser("api_token_lifecycle_user")

	description := "e2e api token lifecycle"
	created, err := s.client.PostUserTokensWithResponse(context.Background(), openapi.PostUserTokensJSONRequestBody{
		Description: &description,
	}, e2eBearer(user.Token))
	require.NoError(t, err)
	requireStatus(t, "create api token", http.StatusCreated, created.StatusCode(), created.Body)
	require.NotNil(t, created.JSON201)
	require.NotNil(t, created.JSON201.ID)
	require.NotEmpty(t, created.JSON201.Token)

	list, err := s.client.GetUserTokensWithResponse(context.Background(), e2eBearer(user.Token))
	require.NoError(t, err)
	requireStatus(t, "list api tokens", http.StatusOK, list.StatusCode(), list.Body)
	require.NotNil(t, list.JSON200)

	found := false

	for _, token := range *list.JSON200 {
		if token.ID != nil && *token.ID == *created.JSON201.ID {
			require.NotNil(t, token.Description)
			require.Equal(t, description, *token.Description)

			found = true
		}
	}

	require.True(t, found, "created API token should be visible in token list")

	solves, err := s.client.GetUsersMeSolvesWithResponse(context.Background(), e2eAPIToken(created.JSON201.Token))
	require.NoError(t, err)
	requireStatus(t, "use api token on read endpoint", http.StatusOK, solves.StatusCode(), solves.Body)

	deleted, err := s.client.DeleteUserTokensIDWithResponse(context.Background(), *created.JSON201.ID, e2eBearer(user.Token))
	require.NoError(t, err)
	requireStatus(t, "delete api token", http.StatusNoContent, deleted.StatusCode(), deleted.Body)

	revoked, err := s.client.GetUsersMeSolvesWithResponse(context.Background(), e2eAPIToken(created.JSON201.Token))
	require.NoError(t, err)
	requireStatus(t, "use deleted api token", http.StatusUnauthorized, revoked.StatusCode(), revoked.Body)
}

func TestE2E_TeamInviteRequiresVerifiedCaptain(t *testing.T) {
	s := newE2ESuite(t)

	defer resetAppSettingsFull()

	admin := s.registerAdmin("invite_verify_settings_admin")
	captain := s.registerUser("invite_unverified_captain")
	s.createTeam(&captain, e2eUID("invite_unverified_team"))

	verifyEmails := true
	settingsResp, err := s.client.PutAdminSettingsWithResponse(context.Background(), openapi.PutAdminSettingsJSONRequestBody{
		VerifyEmails: &verifyEmails,
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "enable email verification", http.StatusOK, settingsResp.StatusCode(), settingsResp.Body)

	_, err = TestPool.Exec(context.Background(), "UPDATE users SET is_verified = FALSE, verified_at = NULL WHERE id = $1", captain.UserID)
	require.NoError(t, err)

	if TestRedis != nil {
		_ = TestRedis.Del(context.Background(), "user:"+captain.UserID).Err()
	}

	invite, err := s.client.GetTeamsMeInviteWithResponse(context.Background(), e2eBearer(captain.Token))
	require.NoError(t, err)
	requireStatus(t, "get invite as unverified captain", http.StatusForbidden, invite.StatusCode(), invite.Body)
}

func TestE2E_OAuthExchangeRejectsEmptyCode(t *testing.T) {
	s := newE2ESuite(t)

	code := "   "
	resp, err := s.client.PostAuthOauthExchangeWithResponse(context.Background(), openapi.PostAuthOauthExchangeJSONRequestBody{
		Code: code,
	})
	require.NoError(t, err)
	requireStatus(t, "oauth exchange empty code", http.StatusBadRequest, resp.StatusCode(), resp.Body)
}

func TestE2E_AuthRefreshUsesCurrentRole(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("refresh_role_admin")

	_, err := TestPool.Exec(context.Background(), "UPDATE users SET role = 'user' WHERE id = $1", admin.UserID)
	require.NoError(t, err)

	if TestRedis != nil {
		_ = TestRedis.Del(context.Background(), "user:"+admin.UserID).Err()
	}

	staleAdminSettings, err := s.client.GetAdminSettingsWithResponse(context.Background(), e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "admin settings with stale demoted token", http.StatusForbidden, staleAdminSettings.StatusCode(), staleAdminSettings.Body)

	refresh, err := s.client.PostAuthRefreshWithResponse(context.Background())
	require.NoError(t, err)
	requireStatus(t, "refresh after role change", http.StatusOK, refresh.StatusCode(), refresh.Body)
	require.NotNil(t, refresh.JSON200)
	require.NotNil(t, refresh.JSON200.AccessToken)

	adminSettings, err := s.client.GetAdminSettingsWithResponse(context.Background(), e2eBearer(*refresh.JSON200.AccessToken))
	require.NoError(t, err)
	requireStatus(t, "admin settings with refreshed non-admin token", http.StatusForbidden, adminSettings.StatusCode(), adminSettings.Body)
}

func TestE2E_AuthRefreshReturnsBanStatusAndBlocksCTFActions(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("refresh_ban_admin")
	player := s.registerUser("refresh_ban_player")
	s.createTeam(&player, e2eUID("refresh_ban_team"))
	challengeID := s.createChallenge(admin, e2eUID("refresh_ban_chal"), "flag{refresh_ban}", 100)

	banMembers := true
	ban, err := s.client.PostAdminTeamsIDBanWithResponse(context.Background(), player.TeamID, openapi.PostAdminTeamsIDBanJSONRequestBody{
		BanMembers: &banMembers,
		Reason:     "e2e banned team",
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "ban team with members", http.StatusOK, ban.StatusCode(), ban.Body)

	if TestRedis != nil {
		_ = TestRedis.Del(context.Background(), "user:"+player.UserID).Err()
	}

	staleMe, err := s.client.GetAuthMeWithResponse(context.Background(), e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "me with stale team-ban token", http.StatusOK, staleMe.StatusCode(), staleMe.Body)
	require.NotNil(t, staleMe.JSON200)
	require.NotNil(t, staleMe.JSON200.BanStatus)
	require.NotNil(t, staleMe.JSON200.BanStatus.IsBanned)
	require.NotNil(t, staleMe.JSON200.BanStatus.Source)
	require.True(t, *staleMe.JSON200.BanStatus.IsBanned)
	require.Equal(t, "direct", string(*staleMe.JSON200.BanStatus.Source))

	s.submitFlag(e2eActor{Token: player.Token}, challengeID, "flag{refresh_ban}", false, http.StatusForbidden)

	staleLeave, err := s.client.PostTeamsLeaveWithResponse(context.Background(), e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "leave team with stale team-ban token", http.StatusForbidden, staleLeave.StatusCode(), staleLeave.Body)

	refresh, err := s.client.PostAuthRefreshWithResponse(context.Background())
	require.NoError(t, err)
	requireStatus(t, "refresh after team ban", http.StatusOK, refresh.StatusCode(), refresh.Body)
	require.NotNil(t, refresh.JSON200)
	require.NotNil(t, refresh.JSON200.AccessToken)

	me, err := s.client.GetAuthMeWithResponse(context.Background(), e2eBearer(*refresh.JSON200.AccessToken))
	require.NoError(t, err)
	requireStatus(t, "me after team ban", http.StatusOK, me.StatusCode(), me.Body)
	require.NotNil(t, me.JSON200)
	require.NotNil(t, me.JSON200.BanStatus)
	require.NotNil(t, me.JSON200.BanStatus.IsBanned)
	require.NotNil(t, me.JSON200.BanStatus.Source)
	require.True(t, *me.JSON200.BanStatus.IsBanned)
	require.Equal(t, "direct", string(*me.JSON200.BanStatus.Source))

	s.submitFlag(e2eActor{Token: *refresh.JSON200.AccessToken}, challengeID, "flag{refresh_ban}", false, http.StatusForbidden)
}

func TestE2E_StaleAccessTokenCannotActForBannedTeam(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("stale_team_ban_admin")
	player := s.registerUser("stale_team_ban_player")
	s.createTeam(&player, e2eUID("stale_team_ban_team"))
	challengeID := s.createChallenge(admin, e2eUID("stale_team_ban_chal"), "flag{stale_team_ban}", 100)
	hintID := s.createHint(admin, challengeID, "stale team ban hint", 10)
	s.createAward(admin, player.TeamID, 50)

	banMembers := false
	ban, err := s.client.PostAdminTeamsIDBanWithResponse(context.Background(), player.TeamID, openapi.PostAdminTeamsIDBanJSONRequestBody{
		BanMembers: &banMembers,
		Reason:     "e2e team ban",
	}, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "ban team without members", http.StatusOK, ban.StatusCode(), ban.Body)

	s.submitFlag(e2eActor{Token: player.Token}, challengeID, "flag{stale_team_ban}", false, http.StatusForbidden)

	hint, err := s.client.PostChallengesChallengeIDHintsHintIDUnlockWithResponse(context.Background(), challengeID, hintID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "unlock hint with stale team-ban token", http.StatusForbidden, hint.StatusCode(), hint.Body)

	review := "stale team ban rating"
	rating, err := s.client.PutChallengesChallengeIDRatingWithResponse(context.Background(), challengeID, openapi.PutChallengesChallengeIDRatingJSONRequestBody{
		Review: &review,
		Value:  5,
	}, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "rate challenge with stale team-ban token", http.StatusForbidden, rating.StatusCode(), rating.Body)

	leave, err := s.client.PostTeamsLeaveWithResponse(context.Background(), e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "leave team with stale team-ban token", http.StatusForbidden, leave.StatusCode(), leave.Body)
}

func TestE2E_WasInBannedTeamProtectedWritesBlocked(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("was_team_ban_admin")
	player := s.registerUser("was_team_ban_player")
	s.createTeam(&player, e2eUID("was_team_ban_team"))
	challengeID := s.createChallenge(admin, e2eUID("was_team_ban_chal"), "flag{was_team_ban}", 100)
	s.submitFlag(player, challengeID, "flag{was_team_ban}", true, http.StatusOK)

	_, err := TestPool.Exec(context.Background(), "UPDATE users SET was_in_banned_team = true WHERE id = $1", player.UserID)
	require.NoError(t, err)

	if TestRedis != nil {
		_ = TestRedis.Del(context.Background(), "user:"+player.UserID).Err()
	}

	me, err := s.client.GetAuthMeWithResponse(context.Background(), e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "auth me with was-in-banned-team state", http.StatusOK, me.StatusCode(), me.Body)
	require.NotNil(t, me.JSON200)
	require.NotNil(t, me.JSON200.BanStatus)
	require.NotNil(t, me.JSON200.BanStatus.IsBanned)
	require.NotNil(t, me.JSON200.BanStatus.Source)
	require.NotNil(t, me.JSON200.BanStatus.CanAppeal)
	require.True(t, *me.JSON200.BanStatus.IsBanned)
	require.Equal(t, "team_inherited", string(*me.JSON200.BanStatus.Source))
	require.True(t, *me.JSON200.BanStatus.CanAppeal)

	fresh := s.login(player.Email, player.Password)
	appeal, err := s.client.PostAppealsWithResponse(context.Background(), openapi.PostAppealsJSONRequestBody{
		Message: "team inherited appeal",
	}, e2eBearer(fresh.Token))
	require.NoError(t, err)
	requireStatus(t, "appeal with was-in-banned-team state", http.StatusCreated, appeal.StatusCode(), appeal.Body)

	newUsername := e2eUID("blocked_profile")
	profile, err := s.client.PatchAuthMeWithResponse(context.Background(), openapi.PatchAuthMeJSONRequestBody{
		Username: &newUsername,
	}, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "patch profile with was-in-banned-team state", http.StatusForbidden, profile.StatusCode(), profile.Body)

	tokenDescription := "blocked token"
	apiToken, err := s.client.PostUserTokensWithResponse(context.Background(), openapi.PostUserTokensJSONRequestBody{
		Description: &tokenDescription,
	}, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "create api token with was-in-banned-team state", http.StatusForbidden, apiToken.StatusCode(), apiToken.Body)

	comment, err := s.client.PostChallengesChallengeIDCommentsWithResponse(context.Background(), challengeID, openapi.PostChallengesChallengeIDCommentsJSONRequestBody{
		Content: "blocked comment",
	}, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "create comment with was-in-banned-team state", http.StatusForbidden, comment.StatusCode(), comment.Body)

	review := "blocked rating"
	rating, err := s.client.PutChallengesChallengeIDRatingWithResponse(context.Background(), challengeID, openapi.PutChallengesChallengeIDRatingJSONRequestBody{
		Review: &review,
		Value:  5,
	}, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "rate challenge with was-in-banned-team state", http.StatusForbidden, rating.StatusCode(), rating.Body)

	duplicateAppeal, err := s.client.PostAppealsWithResponse(context.Background(), openapi.PostAppealsJSONRequestBody{
		Message: "duplicate team inherited appeal",
	}, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "duplicate appeal with was-in-banned-team state", http.StatusTooManyRequests, duplicateAppeal.StatusCode(), duplicateAppeal.Body)
}

func TestE2E_OptionalAuthIgnoresQueryJWTAndAuthenticatedSignedDownloadWorks(t *testing.T) {
	s := newE2ESuite(t)

	defer resetAppSettingsFull()

	admin := s.registerAdmin("optional_query_admin")
	player := s.registerUser("optional_query_player")
	s.createTeam(&player, e2eUID("optional_query_team"))
	challengeID := s.createChallenge(admin, e2eUID("optional_query_chal"), "flag{optional_query}", 100)

	fileContent := "signed download token content"
	fileID := s.uploadChallengeFile(admin, challengeID, "optional-query-token.txt", fileContent)

	s.setScoreVisibility(admin, "private")

	headerScoreboard, err := s.client.GetScoreboardWithResponse(context.Background(), &openapi.GetScoreboardParams{}, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "private scoreboard with bearer header", http.StatusOK, headerScoreboard.StatusCode(), headerScoreboard.Body)

	queryStatus, queryBody := e2eRawGet(t, GetTestBaseURL()+"/api/v1/scoreboard?token="+url.QueryEscape(player.Token))
	requireStatus(t, "private scoreboard with query jwt", http.StatusUnauthorized, queryStatus, queryBody)

	downloadURL, err := s.client.GetFilesIDDownloadWithResponse(context.Background(), fileID, e2eBearer(player.Token))
	require.NoError(t, err)
	requireStatus(t, "get signed download url", http.StatusOK, downloadURL.StatusCode(), downloadURL.Body)
	require.NotNil(t, downloadURL.JSON200)

	noBearerStatus, noBearerBody := e2eRawGet(t, e2eAbsoluteURL(t, downloadURL.JSON200.URL))
	requireStatus(t, "download file without bearer", http.StatusUnauthorized, noBearerStatus, noBearerBody)

	downloadStatus, downloadBody := e2eRawGet(t, e2eAbsoluteURL(t, downloadURL.JSON200.URL), e2eBearer(player.Token))
	requireStatus(t, "download file with bearer and signed token", http.StatusOK, downloadStatus, downloadBody)
	require.Equal(t, fileContent, string(downloadBody))
}
