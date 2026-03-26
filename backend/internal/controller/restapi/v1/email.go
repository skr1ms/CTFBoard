package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

// Verify email
// (POST /auth/verify-email).
func (h *Server) PostAuthVerifyEmail(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.VerifyEmailRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	err := h.user.EmailUC.VerifyEmail(r.Context(), req.Token)
	if h.OnError(w, r, err, "PostAuthVerifyEmail", "VerifyEmail") {
		return
	}

	httputil.RenderOK(w, r, response.Message("email verified successfully"))
}

// Request password reset
// (POST /auth/forgot-password).
func (h *Server) PostAuthForgotPassword(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.ForgotPasswordRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	email := request.ForgotPasswordRequestToParams(&req)

	// Rate Limit: per-email daily limit (prevents targeting a specific account).
	allowed, err := h.infra.ForgotPasswordRateLimiter.Check(r.Context(), email)
	if h.OnError(w, r, err, "PostAuthForgotPassword", "CheckRateLimit") {
		return
	}

	if !allowed {
		h.OnError(w, r, httperr.ErrTooManyRequests(), "PostAuthForgotPassword", "RateLimit")

		return
	}

	if err := h.user.EmailUC.SendPasswordResetEmail(r.Context(), email); err != nil {
		h.infra.Logger.WithError(err).Warn("restapi - v1 - PostAuthForgotPassword - SendPasswordResetEmail")
		// Do not return error to client (avoid email enumeration)
	}

	httputil.RenderOK(w, r, response.Message("if an account exists with this email, a password reset link has been sent"))
}

// Reset password
// (POST /auth/reset-password).
func (h *Server) PostAuthResetPassword(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.ResetPasswordRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	token, newPassword := request.ResetPasswordRequestToParams(&req)
	tokenHash := crypto.SHA256Hex(token)

	allowed, err := h.infra.ResetPasswordTokenRateLimiter.Check(r.Context(), tokenHash)
	if h.OnError(w, r, err, "PostAuthResetPassword", "ResetPasswordTokenRateLimit") {
		return
	}

	if !allowed {
		h.OnError(w, r, httperr.ErrTooManyRequests(), "PostAuthResetPassword", "ResetPasswordTokenRateLimit")

		return
	}

	err = h.user.EmailUC.ResetPassword(r.Context(), token, newPassword)
	if h.OnError(w, r, err, "PostAuthResetPassword", "ResetPassword") {
		return
	}

	httputil.RenderOK(w, r, response.Message("password reset successfully"))
}

// Resend verification email
// (POST /auth/resend-verification).
func (h *Server) PostAuthResendVerification(w http.ResponseWriter, r *http.Request) {
	userID, ok := httputil.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	// Rate Limit: 10 requests per day per user (resend)
	allowed, err := h.infra.ResendVerificationRateLimiter.Check(r.Context(), userID.String())
	if h.OnError(w, r, err, "PostAuthResendVerification", "CheckRateLimit") {
		return
	}

	if !allowed {
		h.OnError(w, r, httperr.ErrTooManyRequests(), "PostAuthResendVerification", "RateLimit")

		return
	}

	if h.OnError(w, r, h.user.EmailUC.ResendVerification(r.Context(), userID), "PostAuthResendVerification", "ResendVerification") {
		return
	}

	httputil.RenderOK(w, r, response.Message("verification email sent"))
}
