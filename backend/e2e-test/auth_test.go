package e2e

import (
	"testing"
)

func TestAuth_RegisterAndLogin(t *testing.T) {
	e := setupE2E(t)

	email, password := registerUser(e, "testuser1")
	token := login(e, email, password)

	obj := e.GET("/api/v1/auth/me").
		WithHeader("Authorization", token).
		Expect().
		Status(200).
		JSON().
		Object()

	obj.Value("email").String().IsEqual(email)
	obj.Value("username").String().IsEqual("testuser1")
	obj.Value("id").NotNull()
}

func TestAuth_RegisterDuplicateUsername(t *testing.T) {
	e := setupE2E(t)

	registerUser(e, "duplicateuser")

	e.POST("/api/v1/auth/register").
		WithJSON(map[string]string{
			"username": "duplicateuser",
			"email":    "different@example.com",
			"password": "password123",
		}).
		Expect().
		Status(409).
		JSON().
		Object().
		Value("error").
		String().
		NotEmpty()
}

func TestAuth_RegisterDuplicateEmail(t *testing.T) {
	e := setupE2E(t)

	email, _ := registerUser(e, "user1")

	e.POST("/api/v1/auth/register").
		WithJSON(map[string]string{
			"username": "user2",
			"email":    email,
			"password": "password123",
		}).
		Expect().
		Status(409).
		JSON().
		Object().
		Value("error").
		String().
		NotEmpty()
}

func TestAuth_LoginInvalidPassword(t *testing.T) {
	e := setupE2E(t)

	email, _ := registerUser(e, "testuser2")

	e.POST("/api/v1/auth/login").
		WithJSON(map[string]string{
			"email":    email,
			"password": "wrongpassword",
		}).
		Expect().
		Status(401).
		JSON().
		Object().
		Value("error").
		String().
		NotEmpty()
}

func TestAuth_LoginInvalidEmail(t *testing.T) {
	e := setupE2E(t)

	e.POST("/api/v1/auth/login").
		WithJSON(map[string]string{
			"email":    "nonexistent@example.com",
			"password": "password123",
		}).
		Expect().
		Status(401).
		JSON().
		Object().
		Value("error").
		String().
		NotEmpty()
}

func TestAuth_MeWithoutToken(t *testing.T) {
	e := setupE2E(t)

	e.GET("/api/v1/auth/me").
		Expect().
		Status(401).
		JSON().
		Object().
		Value("error").
		String().
		NotEmpty()
}
