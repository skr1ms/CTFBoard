package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// POST /auth/register + POST /auth/login + GET /auth/me via generated OpenAPI client.
func TestAuth_RegisterAndLogin_WithClient(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	username := "user_" + uid
	email := "user_" + uid + "@example.com"
	password := "ValidPass1"

	me := h.RegisterLoginAndGetMe(ctx, username, email, password)
	assert.Equal(t, email, *me.Email)
	assert.Equal(t, username, *me.Username)
	assert.NotNil(t, me.ID)
}

// POST /auth/register + POST /auth/login + GET /auth/me: successful registration, login and profile by JWT.
func TestAuth_RegisterAndLogin(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	username := "user_" + uid
	email := "user_" + uid + "@example.com"
	password := "ValidPass1"

	h.Register(username, email, password)
	token := helper.RequireLoginOK(t, h.Login(email, password, http.StatusOK))
	me := helper.RequireMeOK(t, h.MeWithClient(ctx, h.Client(), token))

	assert.Equal(t, email, *me.Email)
	assert.Equal(t, username, *me.Username)
	assert.NotNil(t, me.ID)
}

// POST /auth/register: duplicate username returns 409 Conflict.
func TestAuth_RegisterDuplicateUsername(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	uid := helper.UID()
	username := "dup_" + uid
	h.Register(username, "orig_"+uid+"@example.com", "ValidPass1")
	helper.RequireConflict(t, h.RegisterExpectStatus(username, "other_"+helper.UID()+"@example.com", "ValidPass1", http.StatusConflict), "register")
}

// POST /auth/register: duplicate email returns 409 Conflict.
func TestAuth_RegisterDuplicateEmail(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	uid := helper.UID()
	email := "user_" + uid + "@example.com"
	h.Register("u1_"+uid, email, "ValidPass1")
	helper.RequireConflict(t, h.RegisterExpectStatus("u2_"+helper.UID(), email, "ValidPass1", http.StatusConflict), "register")
}

// POST /auth/login: wrong password returns 401 Unauthorized.
func TestAuth_LoginInvalidPassword(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	uid := helper.UID()
	email := "user_" + uid + "@example.com"
	h.Register("user_"+uid, email, "ValidPass1")
	helper.RequireUnauthorized(t, h.Login(email, "wrongpassword", http.StatusUnauthorized), "login")
}

// POST /auth/login: non-existent email returns 401 Unauthorized.
func TestAuth_LoginInvalidEmail(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	helper.RequireUnauthorized(t, h.Login("nonexistent_"+helper.UID()+"@example.com", "ValidPass1", http.StatusUnauthorized), "login")
}

// GET /auth/me: request without token returns 401 (Auth middleware on protected routes).
func TestAuth_MeWithoutToken(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().GetAuthMeWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireMeUnauthorized(t, resp)
}

// POST /auth/resend-verification: request without token returns 401.
func TestAuth_ResendVerification_WithoutToken(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().PostAuthResendVerificationWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "resend without token")
}

// GET /auth/verify-email: verify email by token; user becomes verified after call.
func TestAuth_EmailVerification_Flow(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	uid := helper.UID()
	username := "verify_" + uid
	email := "verify_" + uid + "@example.com"
	password := "ValidPass1"

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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	uid := helper.UID()
	username := "reset_" + uid
	email := "reset_" + uid + "@example.com"
	password := "ValidPass1"

	h.Register(username, email, password)
	h.ForgotPassword(email, http.StatusOK)

	userID := h.GetUserIDByEmail(email)
	rawToken := "known_reset_token"
	h.InjectToken(userID, entity.TokenTypePasswordReset, rawToken)

	newPassword := "NewValid1"
	h.ResetPassword(rawToken, newPassword)

	h.Login(email, password, http.StatusUnauthorized)
	h.Login(email, newPassword, http.StatusOK)
}

// POST /auth/forgot-password: after 3 requests (limit) rate limit returns 429 Too Many Requests.
func TestAuth_RateLimiting_Exists(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	email := "spam_" + helper.UID() + "@example.com"

	for i := 0; i < 3; i++ {
		h.ForgotPassword(email, http.StatusOK)
	}

	h.ForgotPassword(email, http.StatusTooManyRequests)
}

// POST /auth/resend-verification: authenticated user requests new verification email; returns 200.
func TestAuth_ResendVerification(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, token := h.RegisterUserAndLogin("resend_" + helper.UID())
	h.ResendVerification(token, http.StatusOK)
}

// GET /auth/verify-email: invalid or expired token returns 404.
func TestAuth_VerifyEmail_InvalidToken(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.VerifyEmailExpectStatus("invalid-token", http.StatusNotFound)
}

// POST /auth/reset-password: invalid or expired token returns 404.
func TestAuth_ResetPassword_InvalidToken(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.ResetPasswordExpectStatus("invalid-token", "NewValid1", http.StatusNotFound)
}

// POST /auth/refresh: valid refresh token returns new token pair.
func TestAuth_Refresh_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	username := "refresh_" + helper.UID()
	_, _, accessToken := h.RegisterUserAndLogin(username)
	loginResp := h.Login(username+"@example.com", "ValidPass1", http.StatusOK)
	require.NotNil(t, loginResp.JSON200)
	require.NotNil(t, loginResp.JSON200.RefreshToken)
	refreshToken := *loginResp.JSON200.RefreshToken

	refreshResp := h.Refresh(refreshToken, http.StatusOK)
	newAccess, newRefresh := helper.RequireRefreshOK(t, refreshResp)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)
	assert.NotEqual(t, accessToken, newAccess)

	me := helper.RequireMeOK(t, h.MeWithClient(ctx, h.Client(), newAccess))
	assert.Equal(t, username, *me.Username)
}

// POST /auth/refresh: missing or invalid token returns 401.
func TestAuth_Refresh_InvalidToken(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().PostAuthRefreshWithResponse(context.Background(), nil)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "refresh without token")

	h.Refresh("invalid-refresh-token", http.StatusUnauthorized)
}

// POST /auth/logout: valid refresh token returns 204 NoContent.
func TestAuth_Logout_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	username := "logout_" + helper.UID()
	email, password := h.RegisterUser(username)
	loginResp := h.Login(email, password, http.StatusOK)
	require.NotNil(t, loginResp.JSON200)
	require.NotNil(t, loginResp.JSON200.RefreshToken)
	refreshToken := *loginResp.JSON200.RefreshToken

	authHeader := "Bearer " + refreshToken
	resp, err := h.Client().PostAuthLogoutWithResponse(context.Background(), &openapi.PostAuthLogoutParams{Authorization: &authHeader}, openapi.PostAuthLogoutJSONRequestBody{})
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNoContent, resp.StatusCode(), resp.Body, "logout")
}

// POST /auth/logout: no Authorization header returns 401 (token required).
func TestAuth_Logout_NoAuth(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().PostAuthLogoutWithResponse(context.Background(), nil, openapi.PostAuthLogoutJSONRequestBody{})
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "logout no auth")
}
