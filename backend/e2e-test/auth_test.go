package e2e_test

import (
	"net/http"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/entity"
)

// POST /auth/register + POST /auth/login + GET /auth/me: successful registration, login and profile by JWT.
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

// POST /auth/register: duplicate username returns 409 Conflict.
func TestAuth_RegisterDuplicateUsername(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	h.Register("duplicateuser", "original@example.com", "password123")
	h.RegisterExpectStatus("duplicateuser", "different@example.com", "password123", http.StatusConflict).
		Value("error").String().NotEmpty()
}

// POST /auth/register: duplicate email returns 409 Conflict.
func TestAuth_RegisterDuplicateEmail(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	email := "user1@example.com"
	h.Register("user1", email, "password123")
	h.RegisterExpectStatus("user2", email, "password123", http.StatusConflict).
		Value("error").String().NotEmpty()
}

// POST /auth/login: wrong password returns 401 Unauthorized.
func TestAuth_LoginInvalidPassword(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	email := "testuser2@example.com"
	h.Register("testuser2", email, "password123")
	h.Login(email, "wrongpassword", http.StatusUnauthorized).
		Value("error").String().NotEmpty()
}

// POST /auth/login: non-existent email returns 401 Unauthorized.
func TestAuth_LoginInvalidEmail(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	h.Login("nonexistent@example.com", "password123", http.StatusUnauthorized).
		Value("error").String().NotEmpty()
}

// GET /auth/me: request without token returns 401 (Auth middleware on protected routes).
func TestAuth_MeWithoutToken(t *testing.T) {
	e := setupE2E(t)

	e.GET("/api/v1/auth/me").
		Expect().
		Status(http.StatusUnauthorized).
		JSON().Object().
		Value("error").String().NotEmpty()
}

// GET /auth/verify-email: verify email by token; user becomes verified after call.
func TestAuth_EmailVerification_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	username := "verify_user"
	email := "verify@example.com"
	password := "password123"

	h.Register(username, email, password)
	h.AssertUserVerified(email, false)

	userID := h.GetuserIDByEmail(email)
	rawToken := "known_verification_token"
	h.InjectToken(userID, entity.TokenTypeEmailVerification, rawToken)

	h.VerifyEmail(rawToken)
	h.AssertUserVerified(email, true)
}

// POST /auth/forgot-password + POST /auth/reset-password: reset password by token; old password stops working.
func TestAuth_PasswordReset_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	username := "reset_user"
	email := "reset@example.com"
	password := "old_password"

	h.Register(username, email, password)
	h.ForgotPassword(email, http.StatusOK)

	userID := h.GetuserIDByEmail(email)
	rawToken := "known_reset_token"
	h.InjectToken(userID, entity.TokenTypePasswordReset, rawToken)

	newPassword := "new_secure_password"
	h.ResetPassword(rawToken, newPassword)

	h.Login(email, password, http.StatusUnauthorized)
	h.Login(email, newPassword, http.StatusOK)
}

// POST /auth/forgot-password: after N requests rate limit returns 429 Too Many Requests.
func TestAuth_RateLimiting_Exists(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	email := "spam@example.com"

	for i := 0; i < 10; i++ {
		h.ForgotPassword(email, http.StatusOK)
	}

	h.ForgotPassword(email, http.StatusTooManyRequests)
}

// POST /auth/resend-verification: authenticated user requests new verification email; returns 200.
func TestAuth_ResendVerification(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, _, token := h.RegisterUserAndLogin("resend_verify_user")
	h.ResendVerification(token, http.StatusOK)
}
