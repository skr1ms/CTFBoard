package e2e_test

import (
	"net/http"
	"testing"
)

// GET /auth/me: returns own profile with email, username, team_id.
func TestProfile_GetMe(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	username := "profileuser"
	email, _, token := h.RegisterUserAndLogin(username)
	h.CreateSoloTeam(token, http.StatusCreated)

	resp := h.GetMe(token)

	resp.Value("email").String().IsEqual(email)
	resp.Value("username").String().IsEqual(username)
	resp.Value("team_id").NotNull()
}

// GET /users/{ID}: public profile exposes username but not email.
func TestProfile_GetPublicProfile(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	username := "publicuser"
	_, _, token := h.RegisterUserAndLogin(username)

	meResp := h.GetMe(token)
	userID := meResp.Value("id").String().Raw()

	userProfile := h.GetPublicProfile(userID, http.StatusOK)

	userProfile.Value("username").String().IsEqual(username)
	userProfile.NotContainsKey("email")
}

// GET /users/{ID}: non-existent user returns 404.
func TestProfile_GetPublicProfileNotFound(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	h.GetPublicProfile("00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}
