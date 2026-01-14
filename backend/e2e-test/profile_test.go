package e2e

import (
	"testing"
)

func TestProfile_GetMe(t *testing.T) {
	e := setupE2E(t)

	email, password := registerUser(e, "profileuser")
	token := login(e, email, password)

	resp := e.GET("/api/v1/auth/me").
		WithHeader("Authorization", token).
		Expect().
		Status(200).
		JSON().
		Object()

	resp.Value("email").String().IsEqual(email)
	resp.Value("username").String().IsEqual("profileuser")

	resp.Value("team_id").NotNull()
}

func TestProfile_GetPublicProfile(t *testing.T) {
	e := setupE2E(t)

	email, password := registerUser(e, "publicuser")
	token := login(e, email, password)

	meResp := e.GET("/api/v1/auth/me").
		WithHeader("Authorization", token).
		Expect().
		Status(200).
		JSON().
		Object()

	userId := meResp.Value("id").String().Raw()

	userProfile := e.GET("/api/v1/users/{id}", userId).
		Expect().
		Status(200).
		JSON().
		Object()

	userProfile.Value("username").String().IsEqual("publicuser")

	userProfile.NotContainsKey("email")
}

func TestProfile_GetPublicProfileNotFound(t *testing.T) {
	e := setupE2E(t)

	e.GET("/api/v1/users/00000000-0000-0000-0000-000000000000").
		Expect().
		Status(404)
}
