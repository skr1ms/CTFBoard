package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestE2E_PublicUserStatsFollowScoreVisibility(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("public_user_stats_admin")
	player := s.registerUser("public_user_stats_player")
	s.createTeam(&player, e2eUID("public_user_stats_team"))
	challengeID := s.createChallenge(admin, e2eUID("public_user_stats_chal"), "flag{public_user_stats}", 100)

	s.submitFlag(player, challengeID, "flag{public_user_stats}", true, http.StatusOK)
	s.submitFlag(player, challengeID, "wrong", false, http.StatusOK)
	s.createAward(admin, player.TeamID, 25)

	s.setScoreVisibility(admin, "public")

	solves, err := s.client.GetUsersIDSolvesWithResponse(context.Background(), player.UserID)
	require.NoError(t, err)
	requireStatus(t, "public user solves", http.StatusOK, solves.StatusCode(), solves.Body)
	require.NotNil(t, solves.JSON200)
	require.NotEmpty(t, *solves.JSON200)

	fails, err := s.client.GetUsersIDFailsWithResponse(context.Background(), player.UserID, &openapi.GetUsersIDFailsParams{})
	require.NoError(t, err)
	requireStatus(t, "public user fails", http.StatusOK, fails.StatusCode(), fails.Body)
	require.NotNil(t, fails.JSON200)
	require.NotNil(t, fails.JSON200.Data)
	require.NotEmpty(t, *fails.JSON200.Data)

	awards, err := s.client.GetUsersIDAwardsWithResponse(context.Background(), player.UserID)
	require.NoError(t, err)
	requireStatus(t, "public user awards", http.StatusOK, awards.StatusCode(), awards.Body)
	require.NotNil(t, awards.JSON200)
	require.NotEmpty(t, *awards.JSON200)

	s.setScoreVisibility(admin, "private")

	privateSolves, err := s.client.GetUsersIDSolvesWithResponse(context.Background(), player.UserID)
	require.NoError(t, err)
	requireStatus(t, "private user solves", http.StatusUnauthorized, privateSolves.StatusCode(), privateSolves.Body)
}

func TestE2E_BlockedUserHiddenFromPublicAccountSurfaces(t *testing.T) {
	s := newE2ESuite(t)

	admin := s.registerAdmin("blocked_public_user_admin")
	viewer := s.registerUser("blocked_public_user_viewer")
	player := s.registerUser("blocked_public_user_player")
	s.createTeam(&player, e2eUID("blocked_public_user_team"))
	challengeID := s.createChallenge(admin, e2eUID("blocked_public_user_chal"), "flag{blocked_public_user}", 100)

	s.submitFlag(player, challengeID, "flag{blocked_public_user}", true, http.StatusOK)
	s.setScoreVisibility(admin, "public")

	_, err := TestPool.Exec(context.Background(), "UPDATE users SET was_in_banned_team = true WHERE id = $1", player.UserID)
	require.NoError(t, err)

	profile, err := s.client.GetUsersIDWithResponse(context.Background(), player.UserID, e2eBearer(viewer.Token))
	require.NoError(t, err)
	requireStatus(t, "blocked user profile for regular viewer", http.StatusNotFound, profile.StatusCode(), profile.Body)

	solves, err := s.client.GetUsersIDSolvesWithResponse(context.Background(), player.UserID)
	require.NoError(t, err)
	requireStatus(t, "blocked user solves for public viewer", http.StatusNotFound, solves.StatusCode(), solves.Body)

	adminProfile, err := s.client.GetUsersIDWithResponse(context.Background(), player.UserID, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "blocked user profile for admin", http.StatusOK, adminProfile.StatusCode(), adminProfile.Body)

	adminSolves, err := s.client.GetUsersIDSolvesWithResponse(context.Background(), player.UserID, e2eBearer(admin.Token))
	require.NoError(t, err)
	requireStatus(t, "blocked user solves for admin", http.StatusOK, adminSolves.StatusCode(), adminSolves.Body)
}
