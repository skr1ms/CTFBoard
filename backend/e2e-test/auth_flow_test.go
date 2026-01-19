package e2e_test

import (
	"net/http"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/entity"
)

func TestAuth_EmailVerification_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

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

func TestAuth_PasswordReset_Flow(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

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

func TestAuth_RateLimiting_Exists(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestDB)

	email := "spam@example.com"

	for i := 0; i < 10; i++ {
		h.ForgotPassword(email, http.StatusOK)
	}

	h.ForgotPassword(email, http.StatusTooManyRequests)
}
