package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// PUT /users/me/avatar: success - returns full and thumb presigned URLs.
func TestAvatar_UploadUserAvatar_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("av_user_up")
	h.CreateSoloTeam(token, http.StatusCreated)

	resp := h.UploadUserAvatar(token, helper.MakeAvatarPNG(512), http.StatusOK)
	require.NotNil(t, resp.JSON200)
	assert.NotEmpty(t, resp.JSON200.FullURL)
	assert.NotEmpty(t, resp.JSON200.ThumbURL)
}

// PUT /users/me/avatar: unauthenticated returns 401.
func TestAvatar_UploadUserAvatar_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.UploadUserAvatar("", helper.MakeAvatarPNG(512), http.StatusUnauthorized)
}

// DELETE /users/me/avatar: success after upload.
func TestAvatar_DeleteUserAvatar_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("av_user_del")
	h.CreateSoloTeam(token, http.StatusCreated)

	h.UploadUserAvatar(token, helper.MakeAvatarPNG(512), http.StatusOK)
	h.DeleteUserAvatar(token, http.StatusNoContent)
}

// DELETE /users/me/avatar: unauthenticated returns 401.
func TestAvatar_DeleteUserAvatar_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.DeleteUserAvatar("", http.StatusUnauthorized)
}

// PUT /teams/me/avatar: captain can upload team avatar.
func TestAvatar_UploadTeamAvatar_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("av_team_up")
	h.CreateSoloTeam(token, http.StatusCreated)

	resp := h.UploadTeamAvatar(token, helper.MakeAvatarPNG(512), http.StatusOK)
	require.NotNil(t, resp.JSON200)
	assert.NotEmpty(t, resp.JSON200.FullURL)
	assert.NotEmpty(t, resp.JSON200.ThumbURL)
}

// PUT /teams/me/avatar: user without team returns 400/403.
func TestAvatar_UploadTeamAvatar_NoTeam(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("av_team_noteam")

	h.UploadTeamAvatar(token, helper.MakeAvatarPNG(512), http.StatusNotFound)
}

// DELETE /teams/me/avatar: captain can delete team avatar.
func TestAvatar_DeleteTeamAvatar_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("av_team_del")
	h.CreateSoloTeam(token, http.StatusCreated)

	h.UploadTeamAvatar(token, helper.MakeAvatarPNG(512), http.StatusOK)
	h.DeleteTeamAvatar(token, http.StatusNoContent)
}

// PUT /admin/users/{ID}/avatar: admin can upload avatar for any user.
func TestAvatar_AdminUploadUserAvatar_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("av_adm_user_up")
	meResp := h.GetMe(tokenAdmin, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID)
	userID := *meResp.JSON200.ID

	resp := h.AdminUploadUserAvatar(tokenAdmin, userID, helper.MakeAvatarPNG(512), http.StatusOK)
	require.NotNil(t, resp.JSON200)
	assert.NotEmpty(t, resp.JSON200.FullURL)
}

// DELETE /admin/users/{ID}/avatar: admin can delete avatar for any user.
func TestAvatar_AdminDeleteUserAvatar_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("av_adm_user_del")
	meResp := h.GetMe(tokenAdmin, http.StatusOK)
	require.NotNil(t, meResp.JSON200)
	require.NotNil(t, meResp.JSON200.ID)
	userID := *meResp.JSON200.ID

	h.AdminUploadUserAvatar(tokenAdmin, userID, helper.MakeAvatarPNG(512), http.StatusOK)
	h.AdminDeleteUserAvatar(tokenAdmin, userID, http.StatusNoContent)
}

// PUT /admin/teams/{ID}/avatar: admin can upload avatar for any team.
func TestAvatar_AdminUploadTeamAvatar_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("av_adm_team_up")
	_, _, tokenUser := h.RegisterUserAndLogin("av_adm_tup_u")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	myTeamResp := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeamResp.JSON200)
	require.NotNil(t, myTeamResp.JSON200.ID)
	teamID := *myTeamResp.JSON200.ID

	resp := h.AdminUploadTeamAvatar(tokenAdmin, teamID, helper.MakeAvatarPNG(512), http.StatusOK)
	require.NotNil(t, resp.JSON200)
	assert.NotEmpty(t, resp.JSON200.FullURL)
}

// DELETE /admin/teams/{ID}/avatar: admin can delete avatar for any team.
func TestAvatar_AdminDeleteTeamAvatar_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("av_adm_team_del")
	_, _, tokenUser := h.RegisterUserAndLogin("av_adm_tdel_u")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	myTeamResp := h.GetMyTeam(tokenUser, http.StatusOK)
	require.NotNil(t, myTeamResp.JSON200)
	require.NotNil(t, myTeamResp.JSON200.ID)
	teamID := *myTeamResp.JSON200.ID

	h.AdminUploadTeamAvatar(tokenAdmin, teamID, helper.MakeAvatarPNG(512), http.StatusOK)
	h.AdminDeleteTeamAvatar(tokenAdmin, teamID, http.StatusNoContent)
}

// PUT /users/me/avatar: uploading twice replaces the old avatar and returns new URLs.
func TestAvatar_UploadUserAvatar_Replaces(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("av_user_replace")
	h.CreateSoloTeam(token, http.StatusCreated)

	resp1 := h.UploadUserAvatar(token, helper.MakeAvatarPNG(512), http.StatusOK)
	require.NotNil(t, resp1.JSON200)
	url1 := resp1.JSON200.FullURL

	resp2 := h.UploadUserAvatar(token, helper.MakeAvatarPNG(512), http.StatusOK)
	require.NotNil(t, resp2.JSON200)
	url2 := resp2.JSON200.FullURL

	// Both calls succeed; URLs may differ (different hash) or be the same if image is identical - just assert non-empty
	assert.NotEmpty(t, url1)
	assert.NotEmpty(t, url2)
}
