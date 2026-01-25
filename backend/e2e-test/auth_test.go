package e2e_test

import (
	"net/http"
	"testing"
)

func TestAuth_RegisterAndLogin(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup User Data
	username := "testuser1"
	email := "testuser1@example.com"
	password := "password123"

	// 2. Register New User
	h.Register(username, email, password)

	// 3. Login with Credentials
	loginResp := h.Login(email, password, http.StatusOK)
	token := "Bearer " + loginResp.Value("access_token").String().Raw()

	// 4. Verify Identity via /me Endpoint
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

	// 1. Register Initial User
	h.Register("duplicateuser", "original@example.com", "password123")

	// 2. Attempt to Register Another User with Same Username (Expect Conflict)
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

	// 1. Register Initial User
	email := "user1@example.com"
	h.Register("user1", email, "password123")

	// 2. Attempt to Register Another User with Same Email (Expect Conflict)
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

	// 1. Register User
	email := "testuser2@example.com"
	h.Register("testuser2", email, "password123")

	// 2. Attempt Login with Wrong Password (Expect Unauthorized)
	h.Login(email, "wrongpassword", http.StatusUnauthorized).
		Value("error").String().NotEmpty()
}

func TestAuth_LoginInvalidEmail(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Attempt Login with Non-existent Email (Expect Unauthorized)
	h.Login("nonexistent@example.com", "password123", http.StatusUnauthorized).
		Value("error").String().NotEmpty()
}

func TestAuth_MeWithoutToken(t *testing.T) {
	e := setupE2E(t)

	// 1. Attempt Access to Protected /me Endpoint without Token (Expect Unauthorized)
	e.GET("/api/v1/auth/me").
		Expect().
		Status(http.StatusUnauthorized).
		JSON().Object().
		Value("error").String().NotEmpty()
}
