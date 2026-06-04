package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
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
	h.InjectToken(userID, domain.TokenTypeEmailVerification, rawToken)

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
	h.InjectToken(userID, domain.TokenTypePasswordReset, rawToken)

	newPassword := "NewValid1"
	h.ResetPassword(rawToken, newPassword)

	h.Login(email, password, http.StatusUnauthorized)
	h.Login(email, newPassword, http.StatusOK)
}

// POST /auth/forgot-password: over limit returns 429 (per-email limit 20 in e2e).
func TestAuth_RateLimiting_Exists(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	email := "spam_" + helper.UID() + "@example.com"

	const limit = 20
	for range limit {
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

// POST /auth/refresh: valid refresh cookie returns new access token.
func TestAuth_Refresh_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	username := "refresh_" + helper.UID()
	_, _, accessToken := h.RegisterUserAndLogin(username)
	h.Login(username+"@example.com", "ValidPass1", http.StatusOK) // sets cookie in jar

	refreshResp := h.Refresh(http.StatusOK)
	newAccess := helper.RequireRefreshOK(t, refreshResp)
	assert.NotEmpty(t, newAccess)
	assert.NotEqual(t, accessToken, newAccess)

	me := helper.RequireMeOK(t, h.MeWithClient(ctx, h.Client(), newAccess))
	assert.Equal(t, username, *me.Username)
}

// POST /auth/refresh: missing or invalid token returns 401.
func TestAuth_Refresh_InvalidToken(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().PostAuthRefreshWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "refresh without token")

	h.Refresh(http.StatusUnauthorized, helper.WithRefreshCookie("invalid-refresh-token"))
}

// POST /auth/refresh: admin with WasInBannedTeam can refresh tokens (policy allows admins).
func TestAuth_Refresh_AdminWasInBannedTeam_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	email, password, tokenAdmin := h.RegisterAdmin("admin_refresh_ban_" + helper.UID())
	me := h.GetMe(tokenAdmin, http.StatusOK)
	require.NotNil(t, me.JSON200)
	require.NotNil(t, me.JSON200.ID)
	adminID := *me.JSON200.ID

	_, err := h.Pool().Exec(context.Background(), "UPDATE users SET was_in_banned_team = true WHERE id = $1", adminID)
	require.NoError(t, err)

	if h.Redis() != nil {
		var cursor uint64

		for {
			keys, next, err := h.Redis().Scan(context.Background(), cursor, "user:*", 100).Result()
			require.NoError(t, err)

			if len(keys) > 0 {
				require.NoError(t, h.Redis().Del(context.Background(), keys...).Err())
			}

			cursor = next
			if cursor == 0 {
				break
			}
		}
	}

	h.Login(email, password, http.StatusOK) // sets cookie in jar

	refreshResp := h.Refresh(http.StatusOK)
	helper.RequireRefreshOK(t, refreshResp)
}

// POST /auth/register: invalid custom_fields key (non-UUID) returns 400.
func TestAuth_Register_InvalidCustomFieldKey_Returns400(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	uid := helper.UID()
	username := "reg_inv_cf_" + uid
	email := username + "@example.com"
	password := "ValidPass1"
	invalidKey := "not-a-uuid"
	req := openapi.PostAuthRegisterJSONRequestBody{
		Username:     username,
		Email:        email,
		Password:     password,
		CustomFields: &map[string]string{invalidKey: "value"},
	}
	resp, err := h.Client().PostAuthRegisterWithResponse(context.Background(), req)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusBadRequest, resp.StatusCode(), resp.Body, "register invalid custom field key")
}

// POST /auth/register: dynamic required custom_fields use field UUID keys and persist values.
func TestAuth_Register_WithRequiredCustomField_Success(t *testing.T) {
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_reg_cf_success")
	suffix := helper.UID()
	createResp := h.CreateField(tokenAdmin, "reg_cf_"+suffix, "text", "user", true, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	fieldID := *createResp.JSON201.ID
	defer h.DeleteField(tokenAdmin, fieldID, http.StatusNoContent)

	username := "reg_cf_" + suffix
	email := username + "@example.com"
	password := "ValidPass1"
	value := "Astro University"

	req := openapi.PostAuthRegisterJSONRequestBody{
		Username:     username,
		Email:        email,
		Password:     password,
		CustomFields: &map[string]string{fieldID: " " + value + " "},
	}

	resp, err := h.Client().PostAuthRegisterWithResponse(context.Background(), req)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusCreated, resp.StatusCode(), resp.Body, "register with required custom field")
	require.NotNil(t, resp.JSON201)
	require.NotNil(t, resp.JSON201.ID)

	var storedValue string

	err = h.Pool().QueryRow(
		context.Background(),
		"SELECT value FROM field_values WHERE entity_id = $1 AND field_id = $2",
		*resp.JSON201.ID,
		fieldID,
	).Scan(&storedValue)
	require.NoError(t, err)
	assert.Equal(t, value, storedValue)
}

// POST /auth/register: required dynamic custom_fields are enforced by the usecase validator.
func TestAuth_Register_RequiredCustomFieldMissing_Returns400(t *testing.T) {
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_reg_cf_missing")
	suffix := helper.UID()
	createResp := h.CreateField(tokenAdmin, "reg_cf_missing_"+suffix, "text", "user", true, http.StatusCreated)
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.ID)

	fieldID := *createResp.JSON201.ID
	defer h.DeleteField(tokenAdmin, fieldID, http.StatusNoContent)

	req := openapi.PostAuthRegisterJSONRequestBody{
		Username: "reg_cf_missing_" + suffix,
		Email:    "reg_cf_missing_" + suffix + "@example.com",
		Password: "ValidPass1",
	}
	resp, err := h.Client().PostAuthRegisterWithResponse(context.Background(), req)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusBadRequest, resp.StatusCode(), resp.Body, "register missing required custom field")
}

// POST /auth/logout: valid refresh token returns 204 NoContent.
func TestAuth_Logout_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	username := "logout_" + helper.UID()
	email, password := h.RegisterUser(username)
	h.Login(email, password, http.StatusOK) // sets cookie in jar

	resp, err := h.Client().PostAuthLogoutWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNoContent, resp.StatusCode(), resp.Body, "logout")
}

// POST /auth/logout: no refresh cookie returns 401.
func TestAuth_Logout_NoAuth(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	resp, err := h.Client().PostAuthLogoutWithResponse(context.Background())
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusUnauthorized, resp.StatusCode(), resp.Body, "logout no auth")
}

// POST /auth/resend-verification-by-email: registered unverified user gets 200.
func TestAuth_ResendVerificationByEmail_Success(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	suffix := helper.UID()
	email := "resend_byemail_" + suffix + "@example.com"
	h.Register("resend_byemail_"+suffix, email, "ValidPass1!")

	h.ResendVerificationByEmail(email, http.StatusOK)
}

// POST /auth/resend-verification-by-email: non-existent email returns 200 (no enumeration).
func TestAuth_ResendVerificationByEmail_NonExistentEmail(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.ResendVerificationByEmail("nosuchuser@example.com", http.StatusOK)
}

// POST /auth/resend-verification-by-email: malformed email returns 400.
func TestAuth_ResendVerificationByEmail_InvalidEmail(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	h.ResendVerificationByEmail("not-an-email", http.StatusBadRequest)
}
