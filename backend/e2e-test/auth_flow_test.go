package e2e_test

import (
	"net/http"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/entity"
)

func TestAuth_EmailVerification_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup User Data
	username := "verify_user"
	email := "verify@example.com"
	password := "password123"

	// 2. Register Unverified User
	h.Register(username, email, password)

	h.AssertUserVerified(email, false)

	// 3. Obtain User ID and Prepare Verification Token
	userID := h.GetUserIDByEmail(email)
	rawToken := "known_verification_token"

	// 4. Inject Token into DB (Bypassing Mailer)
	h.InjectToken(userID, entity.TokenTypeEmailVerification, rawToken)

	// 5. Verify Email via API
	h.VerifyEmail(rawToken)

	// 6. Assert User is Verified
	h.AssertUserVerified(email, true)
}

func TestAuth_PasswordReset_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup User Data
	username := "reset_user"
	email := "reset@example.com"
	password := "old_password"

	// 2. Register User
	h.Register(username, email, password)

	// 3. Request Password Reset
	h.ForgotPassword(email, http.StatusOK)

	// 4. Inject Reset Token into DB
	userID := h.GetUserIDByEmail(email)
	rawToken := "known_reset_token"

	h.InjectToken(userID, entity.TokenTypePasswordReset, rawToken)

	// 5. Reset Password via API
	newPassword := "new_secure_password"
	h.ResetPassword(rawToken, newPassword)

	// 6. Verify Old Password Fails and New Password Works
	h.Login(email, password, http.StatusUnauthorized)

	h.Login(email, newPassword, http.StatusOK)
}

func TestAuth_RateLimiting_Exists(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	// 1. Setup Target Email
	email := "spam@example.com"

	// 2. Trigger Rate Limit by Repeated Requests
	for i := 0; i < 10; i++ {
		h.ForgotPassword(email, http.StatusOK)
	}

	// 3. Expect 429 Too Many Requests
	h.ForgotPassword(email, http.StatusTooManyRequests)
}
