package e2e_test

import (
	"net/http"
	"testing"
)

func TestProfile_GetMe(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

	username := "profileuser"

	email, _, token := h.RegisterUserAndLogin(username)

	resp := h.GetMe(token)

	resp.Value("email").String().IsEqual(email)
	resp.Value("username").String().IsEqual(username)
	resp.Value("team_id").NotNull()
}

func TestProfile_GetPublicProfile(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

	username := "publicuser"
	_, _, token := h.RegisterUserAndLogin(username)

	meResp := h.GetMe(token)
	userID := meResp.Value("id").String().Raw()

	userProfile := h.GetPublicProfile(userID, http.StatusOK)

	userProfile.Value("username").String().IsEqual(username)
	userProfile.NotContainsKey("email")
}

func TestProfile_GetPublicProfileNotFound(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

	h.GetPublicProfile("00000000-0000-0000-0000-000000000000", http.StatusNotFound)
}
