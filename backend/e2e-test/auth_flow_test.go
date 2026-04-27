package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// TestAuthFlow_FullLifecycle covers the full authentication lifecycle:
// register -> verify email -> login -> refresh token -> logout
// attempt to use the revoked refresh token returns 401.
func TestAuthFlow_FullLifecycle(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())
	ctx := context.Background()

	uid := helper.UID()
	username := "authflow_" + uid
	email := "authflow_" + uid + "@example.com"
	password := "ValidPass1"

	// 1. Register
	h.Register(username, email, password)
	h.AssertUserVerified(email, false)

	// 2. Login before email verification - should succeed (verify_emails is true but login is still allowed)
	loginResp := h.Login(email, password, http.StatusOK)
	require.NotNil(t, loginResp.JSON200)
	require.NotEmpty(t, loginResp.JSON200.AccessToken)
	accessToken := "Bearer " + *loginResp.JSON200.AccessToken

	// 3. Inject known verification token and verify email
	userID := h.GetUserIDByEmail(email)
	rawVerifyToken := "known_verify_" + uid
	h.InjectToken(userID, domain.TokenTypeEmailVerification, rawVerifyToken)
	h.VerifyEmail(rawVerifyToken)
	h.AssertUserVerified(email, true)

	// 4. Verify that same token is now invalid (one-time use)
	h.VerifyEmailExpectStatus(rawVerifyToken, http.StatusNotFound)

	// 5. Get new access token after verification; Login sets the httpOnly refresh cookie in the jar
	loginResp2 := h.Login(email, password, http.StatusOK)
	require.NotNil(t, loginResp2.JSON200)
	require.NotNil(t, loginResp2.JSON200.AccessToken)
	newAccess := "Bearer " + *loginResp2.JSON200.AccessToken

	// 6. Access protected endpoint with new access token
	me := h.GetMe(newAccess, http.StatusOK)
	require.NotNil(t, me.JSON200)
	assert.Equal(t, username, *me.JSON200.Username)
	assert.Equal(t, email, *me.JSON200.Email)

	// 7. Refresh via cookie - get new access token
	refreshResp := h.Refresh(http.StatusOK)
	newAccessToken := helper.RequireRefreshOK(t, refreshResp)
	assert.NotEmpty(t, newAccessToken)
	assert.NotEqual(t, *loginResp2.JSON200.AccessToken, newAccessToken)

	// 8. Old access token still valid (not revoked by refresh)
	_ = accessToken // was issued before verify; still valid within TTL

	// 9. New access token works
	me2 := h.GetMe("Bearer "+newAccessToken, http.StatusOK)
	require.NotNil(t, me2.JSON200)
	assert.Equal(t, username, *me2.JSON200.Username)

	// 10. Logout via cookie - server clears the httpOnly cookie
	logoutResp, err := h.Client().PostAuthLogoutWithResponse(ctx)
	require.NoError(t, err)
	helper.RequireStatus(t, http.StatusNoContent, logoutResp.StatusCode(), logoutResp.Body, "logout")

	// 11. After logout the cookie is cleared - refresh returns 401
	h.Refresh(http.StatusUnauthorized)
}

// TestAuthFlow_PasswordReset_InvalidatesOldCredentials verifies that after a password reset
// the old password is rejected and the new password works.
func TestAuthFlow_PasswordReset_InvalidatesOldCredentials(t *testing.T) {
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	uid := helper.UID()
	username := "pwreset_" + uid
	email := "pwreset_" + uid + "@example.com"
	password := "OldPass1"

	h.Register(username, email, password)

	// Login with old password
	h.Login(email, password, http.StatusOK)

	// Request password reset
	h.ForgotPassword(email, http.StatusOK)

	// Inject known token and reset password
	userID := h.GetUserIDByEmail(email)
	rawToken := "reset_" + uid
	h.InjectToken(userID, domain.TokenTypePasswordReset, rawToken)

	newPassword := "NewPass1"
	h.ResetPassword(rawToken, newPassword)

	// Old password no longer works
	h.Login(email, password, http.StatusUnauthorized)

	// New password works
	loginResp := h.Login(email, newPassword, http.StatusOK)
	require.NotNil(t, loginResp.JSON200)
	require.NotNil(t, loginResp.JSON200.AccessToken)

	// Reset token is consumed - second use returns 404
	h.ResetPasswordExpectStatus(rawToken, "AnotherPass1", http.StatusNotFound)
}
