package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

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

	logout, err := s.client.PostAuthLogoutWithResponse(context.Background(), e2eBearer(*refresh.JSON200.AccessToken))
	require.NoError(t, err)
	requireStatus(t, "logout session", http.StatusNoContent, logout.StatusCode(), logout.Body)

	afterLogoutRefresh, err := s.client.PostAuthRefreshWithResponse(context.Background())
	require.NoError(t, err)
	requireStatus(t, "refresh after logout", http.StatusUnauthorized, afterLogoutRefresh.StatusCode(), afterLogoutRefresh.Body)

	afterLogoutMe, err := s.client.GetAuthMeWithResponse(context.Background(), e2eBearer(*refresh.JSON200.AccessToken))
	require.NoError(t, err)
	requireStatus(t, "me after logout", http.StatusUnauthorized, afterLogoutMe.StatusCode(), afterLogoutMe.Body)
}
