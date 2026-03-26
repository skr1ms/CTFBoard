package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// POST /challenges/{id}/hints/{hintId}/unlock: banned user gets 401 (token revoked).
func TestBannedUser_CannotUnlockHint(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("aubh_" + suffix)

	challID := h.CreateBasicChallenge(tokenAdmin, "User Ban Hint "+suffix, "flag{user_ban_hint_"+suffix+"}", 100)
	hintID := h.CreateHint(tokenAdmin, challID, "User ban hint test", 0)

	userName := "user_ban_hint_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	meResp := h.GetMe(tokenUser, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID)
	userID := *meResp.JSON200.ID

	t.Log("=== Before ban: hint unlock succeeds ===")
	h.UnlockHint(tokenUser, challID, hintID, http.StatusOK)

	t.Log("=== Admin bans user ===")
	h.BanUser(tokenAdmin, userID, "Testing user ban hint", http.StatusOK)
	h.InvalidateUserCache(userID)

	hintID2 := h.CreateHint(tokenAdmin, challID, "Second hint for user ban test", 0)

	t.Log("=== After ban: hint unlock blocked (token revoked) ===")
	h.UnlockHint(tokenUser, challID, hintID2, http.StatusUnauthorized)
}

// GET /files/{ID}/download: banned user gets 401 (token revoked).
func TestBannedUser_CannotDownloadFile(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("aubf_" + suffix)

	challID := h.CreateBasicChallenge(tokenAdmin, "User Ban File "+suffix, "flag{user_ban_file_"+suffix+"}", 100)

	uploadResp := h.UploadChallengeFile(tokenAdmin, challID, "secret.txt", "secret content")
	require.NotNil(t, uploadResp.JSON201)
	fileID := uploadResp.JSON201.ID

	userName := "user_ban_file_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	meResp := h.GetMe(tokenUser, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID)
	userID := *meResp.JSON200.ID

	t.Log("=== Before ban: file download URL accessible ===")
	h.GetFilesIDDownloadExpectStatus(tokenUser, fileID, http.StatusOK)

	t.Log("=== Admin bans user ===")
	h.BanUser(tokenAdmin, userID, "Testing user ban file", http.StatusOK)
	h.InvalidateUserCache(userID)

	t.Log("=== After ban: file download blocked (token revoked) ===")
	h.GetFilesIDDownloadExpectStatus(tokenUser, fileID, http.StatusUnauthorized)
}

// Submit, hint unlock, file download: banned user all return 401 (token revoked).
func TestBannedUser_AllActionsBlocked(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("auba_" + suffix)

	challID := h.CreateBasicChallenge(tokenAdmin, "User Ban All "+suffix, "flag{user_ban_all_"+suffix+"}", 100)
	hintID := h.CreateHint(tokenAdmin, challID, "Hint for user ban all test", 0)
	uploadResp := h.UploadChallengeFile(tokenAdmin, challID, "file.txt", "file content")
	require.NotNil(t, uploadResp.JSON201)
	fileID := uploadResp.JSON201.ID

	userName := "user_ban_all_" + suffix
	_, _, tokenUser := h.RegisterUserAndLogin(userName)
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	meResp := h.GetMe(tokenUser, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID)
	userID := *meResp.JSON200.ID

	h.SubmitFlag(tokenUser, challID, "flag{wrong}", http.StatusOK)
	h.UnlockHint(tokenUser, challID, hintID, http.StatusOK)
	h.GetFilesIDDownloadExpectStatus(tokenUser, fileID, http.StatusOK)

	t.Log("=== Ban user ===")
	h.BanUser(tokenAdmin, userID, "All actions user ban test", http.StatusOK)
	h.InvalidateUserCache(userID)

	t.Log("=== All three actions must be blocked (token revoked) ===")
	h.SubmitFlag(tokenUser, challID, "flag{user_ban_all_"+suffix+"}", http.StatusUnauthorized)
	hintID2 := h.CreateHint(tokenAdmin, challID, "Second hint for user ban all", 0)
	h.UnlockHint(tokenUser, challID, hintID2, http.StatusUnauthorized)
	h.GetFilesIDDownloadExpectStatus(tokenUser, fileID, http.StatusUnauthorized)
}

// POST /teams/leave: banned user gets 403.
func TestBannedUser_LeaveRejected(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("aubleave_" + suffix)

	_, _, tokenCap := h.RegisterUserAndLogin("cap_ban_leave_" + suffix)
	h.CreateTeam(tokenCap, "BanLeaveTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, team.JSON200)
	inviteToken := *team.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin("member_ban_leave_" + suffix)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	meResp := h.GetMe(tokenMember, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID)
	memberID := *meResp.JSON200.ID

	h.BanUser(tokenAdmin, memberID, "Testing leave after ban", http.StatusOK)
	h.InvalidateUserCache(memberID)

	h.LeaveTeam(tokenMember, http.StatusUnauthorized)
}

// Ban former team member: team score drops (solves removed from team).
func TestBannedUser_FormerMemberSolvesRemovedFromTeam(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("aubformer_" + suffix)

	challID := h.CreateBasicChallenge(tokenAdmin, "Former Ban "+suffix, "flag{former_ban_"+suffix+"}", 100)

	_, _, tokenCap := h.RegisterUserAndLogin("cap_former_" + suffix)
	h.CreateTeam(tokenCap, "FormerTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, team.JSON200)
	teamName := *team.JSON200.Name
	inviteToken := *team.JSON200.InviteToken

	_, _, tokenMember := h.RegisterUserAndLogin("member_former_" + suffix)
	h.JoinTeam(tokenMember, inviteToken, false, http.StatusOK)

	h.SubmitFlag(tokenMember, challID, "flag{former_ban_"+suffix+"}", http.StatusOK)
	h.AssertTeamScore(tokenCap, teamName, 100)

	h.LeaveTeam(tokenMember, http.StatusNoContent)

	meResp := h.GetMe(tokenMember, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID)
	memberID := *meResp.JSON200.ID
	h.BanUser(tokenAdmin, memberID, "Testing former member ban", http.StatusOK)
	h.InvalidateUserCache(memberID)
	invalidateScoreboardCache(context.Background())

	h.AssertTeamScore(tokenCap, teamName, 0)
}

// DELETE /admin/teams/{ID}/ban + DELETE /admin/users/{ID}/ban: unban team then user; user can log in again.
func TestUnbanTeam_ThenUnbanUser_UserCanLogin(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	_, tokenAdmin := h.SetupCompetition("unban_flow_" + suffix)

	_, passwordCap, tokenCap := h.RegisterUserAndLogin("cap_unban_" + suffix)
	meCap := h.GetMe(tokenCap, http.StatusOK)
	require.NotNil(t, meCap.JSON200)
	require.NotNil(t, meCap.JSON200.ID)
	capID := *meCap.JSON200.ID

	h.CreateTeam(tokenCap, "UnbanFlowTeam_"+suffix, http.StatusCreated)
	team := h.GetMyTeam(tokenCap, http.StatusOK)
	require.NotNil(t, team.JSON200)
	require.NotNil(t, team.JSON200.ID)
	teamID := *team.JSON200.ID

	h.BanTeamWithOptions(tokenAdmin, teamID, "test", true, http.StatusOK)
	h.UnbanTeam(tokenAdmin, teamID, http.StatusNoContent)
	h.UnbanUser(tokenAdmin, capID, http.StatusNoContent)

	if h.Redis() != nil {
		var cursor uint64

		for {
			keys, next, err := h.Redis().Scan(context.Background(), cursor, "user:*", 100).Result()
			require.NoError(t, err)

			if len(keys) > 0 {
				_ = h.Redis().Del(context.Background(), keys...)
			}

			cursor = next
			if cursor == 0 {
				break
			}
		}
	}

	emailCap := "cap_unban_" + suffix + "@example.com"
	h.Login(emailCap, passwordCap, http.StatusOK)
}
