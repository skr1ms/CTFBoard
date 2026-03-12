package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// GET /auth/me: returns own profile with email, username, team_id.
func TestProfile_GetMe(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	username := "profile_" + helper.UID()
	email, _, token := h.RegisterUserAndLogin(username)
	h.CreateSoloTeam(token, http.StatusCreated)

	resp := h.MeWithClient(context.Background(), h.Client(), token)
	me := helper.RequireMeOK(t, resp)
	require.Equal(t, email, *me.Email)
	require.Equal(t, username, *me.Username)
	require.NotNil(t, me.TeamID)
}

// GET /users/{ID}: profile requires auth, exposes username but not email.
func TestProfile_GetPublicProfile(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	username := "public_" + helper.UID()
	_, _, token := h.RegisterUserAndLogin(username)

	meResp := h.MeWithClient(context.Background(), h.Client(), token)
	me := helper.RequireMeOK(t, meResp)
	require.NotNil(t, me.ID)
	userID := *me.ID

	userProfile := h.GetProfileWithAuth(token, userID, http.StatusOK)
	require.NotNil(t, userProfile.JSON200)
	require.Equal(t, username, *userProfile.JSON200.Username)
}

// GET /users/{ID}: without auth returns 401.
func TestProfile_GetUsersID_Unauthorized(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("some_" + helper.UID())
	meResp := h.MeWithClient(context.Background(), h.Client(), token)
	me := helper.RequireMeOK(t, meResp)
	require.NotNil(t, me.ID)

	h.GetPublicProfile(*me.ID, http.StatusUnauthorized)
}

// GET /users/{ID}: non-existent user returns 404.
func TestProfile_GetPublicProfileNotFound(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("profile404_" + helper.UID())
	h.GetProfileWithAuth(token, "00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}
