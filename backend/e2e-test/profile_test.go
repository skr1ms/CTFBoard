package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// GET /auth/me: returns own profile with email, username, team_id.
func TestProfile_GetMe(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	username := "profileuser"
	email, _, token := h.RegisterUserAndLogin(username)
	h.CreateSoloTeam(token, http.StatusCreated)

	resp := h.MeWithClient(context.Background(), h.client, token)
	me := RequireMeOK(t, resp)
	require.Equal(t, email, *me.Email)
	require.Equal(t, username, *me.Username)
	require.NotNil(t, me.TeamID)
}

// GET /users/{ID}: public profile exposes username but not email.
func TestProfile_GetPublicProfile(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	username := "publicuser"
	_, _, token := h.RegisterUserAndLogin(username)

	meResp := h.MeWithClient(context.Background(), h.client, token)
	me := RequireMeOK(t, meResp)
	require.NotNil(t, me.ID)
	userID := *me.ID

	userProfile := h.GetPublicProfile(userID, http.StatusOK)
	require.NotNil(t, userProfile.JSON200)
	require.Equal(t, username, *userProfile.JSON200.Username)
}

// GET /users/{ID}: non-existent user returns 404.
func TestProfile_GetPublicProfileNotFound(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	h.GetPublicProfile("00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}
