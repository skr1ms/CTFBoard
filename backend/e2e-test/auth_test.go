package e2e_test

import (
	"net/http"
	"testing"
)

func TestAuth_RegisterAndLogin(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	username := "testuser1"
	email := "testuser1@example.com"
	password := "password123"

	h.Register(username, email, password)

	loginResp := h.Login(email, password, http.StatusOK)
	token := "Bearer " + loginResp.Value("access_token").String().Raw()

	meResp := e.GET("/api/v1/auth/me").
		WithHeader("Authorization", token).
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	meResp.Value("email").String().IsEqual(email)
	meResp.Value("username").String().IsEqual(username)
	meResp.Value("id").NotNull()
}

func TestAuth_RegisterDuplicateUsername(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	h.Register("duplicateuser", "original@example.com", "password123")

	e.POST("/api/v1/auth/register").
		WithJSON(map[string]string{
			"username": "duplicateuser",
			"email":    "different@example.com",
			"password": "password123",
		}).
		Expect().
		Status(http.StatusConflict).
		JSON().Object().
		Value("error").String().NotEmpty()
}

func TestAuth_RegisterDuplicateEmail(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	email := "user1@example.com"
	h.Register("user1", email, "password123")

	e.POST("/api/v1/auth/register").
		WithJSON(map[string]string{
			"username": "user2",
			"email":    email,
			"password": "password123",
		}).
		Expect().
		Status(http.StatusConflict).
		JSON().Object().
		Value("error").String().NotEmpty()
}

func TestAuth_LoginInvalidPassword(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	email := "testuser2@example.com"
	h.Register("testuser2", email, "password123")

	h.Login(email, "wrongpassword", http.StatusUnauthorized).
		Value("error").String().NotEmpty()
}

func TestAuth_LoginInvalidEmail(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	h.Login("nonexistent@example.com", "password123", http.StatusUnauthorized).
		Value("error").String().NotEmpty()
}

func TestAuth_MeWithoutToken(t *testing.T) {
	e := setupE2E(t)
	e.GET("/api/v1/auth/me").
		Expect().
		Status(http.StatusUnauthorized).
		JSON().Object().
		Value("error").String().NotEmpty()
}
