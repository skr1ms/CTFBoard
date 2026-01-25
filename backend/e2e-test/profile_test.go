package e2e_test

import (
	"net/http"
	"testing"
)

func TestProfile_GetMe(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup User
	username := "profileuser"
	email, _, token := h.RegisterUserAndLogin(username)

	// 2. Fetch Own Profile
	resp := h.GetMe(token)

	// 3. Verify Personal Info is Visible
	resp.Value("email").String().IsEqual(email)
	resp.Value("username").String().IsEqual(username)
	resp.Value("team_id").NotNull()
}

func TestProfile_GetPublicProfile(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup User
	username := "publicuser"
	_, _, token := h.RegisterUserAndLogin(username)

	// 2. Get User ID from Own Profile
	meResp := h.GetMe(token)
	userID := meResp.Value("id").String().Raw()

	// 3. Access Public Profile Endpoint
	userProfile := h.GetPublicProfile(userID, http.StatusOK)

	// 4. Verify Sensitive Data (Email) is Hidden
	userProfile.Value("username").String().IsEqual(username)
	userProfile.NotContainsKey("email")
}

func TestProfile_GetPublicProfileNotFound(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Attempt Accessing Profile of Non-existent User (Expect 404)
	h.GetPublicProfile("00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}
