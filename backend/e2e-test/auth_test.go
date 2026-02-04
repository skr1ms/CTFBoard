package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// POST /auth/register + POST /auth/login + GET /auth/me via generated OpenAPI client.
func TestAuth_RegisterAndLogin_WithClient(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())
	ctx := context.Background()

	username := "clientuser1"
	email := "clientuser1@example.com"
	password := "password123"

	me := h.RegisterLoginAndGetMe(ctx, username, email, password)
	assert.Equal(t, email, *me.Email)
	assert.Equal(t, username, *me.Username)
	assert.NotNil(t, me.ID)
}

// POST /auth/register + POST /auth/login + GET /auth/me: successful registration, login and profile by JWT.
func TestAuth_RegisterAndLogin(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())
	ctx := context.Background()

	username := "testuser1"
	email := "testuser1@example.com"
	password := "password123"

	h.Register(username, email, password)
	token := helper.RequireLoginOK(t, h.Login(email, password, http.StatusOK))
	me := helper.RequireMeOK(t, h.MeWithClient(ctx, h.Client(), token))

	assert.Equal(t, email, *me.Email)
	assert.Equal(t, username, *me.Username)
	assert.NotNil(t, me.ID)
}

// POST /auth/register: duplicate username returns 409 Conflict.
func TestAuth_RegisterDuplicateUsername(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	h.Register("duplicateuser", "original@example.com", "password123")
	helper.RequireConflict(t, h.RegisterExpectStatus("duplicateuser", "different@example.com", "password123", http.StatusConflict), "register")
}

// POST /auth/register: duplicate email returns 409 Conflict.
func TestAuth_RegisterDuplicateEmail(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	email := "user1@example.com"
	h.Register("user1", email, "password123")
	helper.RequireConflict(t, h.RegisterExpectStatus("user2", email, "password123", http.StatusConflict), "register")
}

// POST /auth/login: wrong password returns 401 Unauthorized.
func TestAuth_LoginInvalidPassword(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	email := "testuser2@example.com"
	h.Register("testuser2", email, "password123")
	helper.RequireUnauthorized(t, h.Login(email, "wrongpassword", http.StatusUnauthorized), "login")
}

// POST /auth/login: non-existent email returns 401 Unauthorized.
func TestAuth_LoginInvalidEmail(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	helper.RequireUnauthorized(t, h.Login("nonexistent@example.com", "password123", http.StatusUnauthorized), "login")
}

// GET /auth/me: request without token returns 401 (Auth middleware on protected routes).
func TestAuth_MeWithoutToken(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	resp, err := h.Client().GetAuthMeWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireMeUnauthorized(t, resp)
}

// POST /auth/resend-verification: request without token returns 401.
func TestAuth_ResendVerification_WithoutToken(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	resp, err := h.Client().PostAuthResendVerificationWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "resend without token")
}

// GET /auth/verify-email: verify email by token; user becomes verified after call.
func TestAuth_EmailVerification_Flow(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	username := "verify_user"
	email := "verify@example.com"
	password := "password123"

	h.Register(username, email, password)
	h.AssertUserVerified(email, false)

	userID := h.GetUserIDByEmail(email)
	rawToken := "known_verification_token"
	h.InjectToken(userID, entity.TokenTypeEmailVerification, rawToken)

	h.VerifyEmail(rawToken)
	h.AssertUserVerified(email, true)
}

// POST /auth/forgot-password + POST /auth/reset-password: reset password by token; old password stops working.
func TestAuth_PasswordReset_Flow(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	username := "reset_user"
	email := "reset@example.com"
	password := "old_password"

	h.Register(username, email, password)
	h.ForgotPassword(email, http.StatusOK)

	userID := h.GetUserIDByEmail(email)
	rawToken := "known_reset_token"
	h.InjectToken(userID, entity.TokenTypePasswordReset, rawToken)

	newPassword := "new_secure_password"
	h.ResetPassword(rawToken, newPassword)

	h.Login(email, password, http.StatusUnauthorized)
	h.Login(email, newPassword, http.StatusOK)
}

// POST /auth/forgot-password: after N requests rate limit returns 429 Too Many Requests.
func TestAuth_RateLimiting_Exists(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	email := "spam@example.com"

	for i := 0; i < 10; i++ {
		h.ForgotPassword(email, http.StatusOK)
	}

	h.ForgotPassword(email, http.StatusTooManyRequests)
}

// POST /auth/resend-verification: authenticated user requests new verification email; returns 200.
func TestAuth_ResendVerification(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("resend_verify_user")
	h.ResendVerification(token, http.StatusOK)
}

// GET /auth/verify-email: invalid or expired token returns 404.
func TestAuth_VerifyEmail_InvalidToken(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	h.VerifyEmailExpectStatus("invalid-token", http.StatusNotFound)
}

// POST /auth/reset-password: invalid or expired token returns 404.
func TestAuth_ResetPassword_InvalidToken(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	h.ResetPasswordExpectStatus("invalid-token", "newpass123", http.StatusNotFound)
}
