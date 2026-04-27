package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

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

// PostAuthForgotPassword handles password-reset requests. It applies a per-email
// daily rate limit on top of the IP-level rate limit applied at the router layer,
// which together prevent targeted account enumeration. Errors from SendPasswordResetEmail
// are deliberately swallowed and a generic success message is always returned so
// the response gives no hint about whether an account exists for the given email.
// (POST /auth/forgot-password).
func (h *Server) PostAuthForgotPassword(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.ForgotPasswordRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	email := request.ForgotPasswordRequestToParams(&req)

	// Rate Limit: per-email daily limit (prevents targeting a specific account)
	allowed, err := h.infra.ForgotPasswordRateLimiter.Check(r.Context(), email)
	if h.OnError(w, r, err, "PostAuthForgotPassword", "CheckRateLimit") {
		return
	}

	if !allowed {
		h.OnError(w, r, apperr.ErrTooManyRequests, "PostAuthForgotPassword", "RateLimit")

		return
	}

	if err := h.user.EmailUC.SendPasswordResetEmail(r.Context(), email); err != nil {
		h.infra.Logger.WithError(err).Warn("restapi - v1 - PostAuthForgotPassword - SendPasswordResetEmail")
		// Do not return error to client (avoid email enumeration)
	}

	httputil.RenderOK(w, r, response.Message("if an account exists with this email, a password reset link has been sent"))
}

// PostAuthResendVerificationByEmail handles public resend-verification requests by
// email address. It applies a per-email daily rate limit on top of the IP-level
// rate limit applied at the router layer. Errors from ResendVerificationByEmail
// are deliberately swallowed so the response gives no hint about whether an
// account exists for the given email.
// (POST /auth/resend-verification-by-email).
func (h *Server) PostAuthResendVerificationByEmail(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.ResendVerificationByEmailRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	email := request.ResendVerificationByEmailRequestToParams(&req)

	// Rate Limit: per-email daily limit (prevents targeting a specific account)
	allowed, err := h.infra.ResendVerificationRateLimiter.Check(r.Context(), email)
	if h.OnError(w, r, err, "PostAuthResendVerificationByEmail", "CheckRateLimit") {
		return
	}

	if !allowed {
		h.OnError(w, r, apperr.ErrTooManyRequests, "PostAuthResendVerificationByEmail", "RateLimit")

		return
	}

	if err := h.user.EmailUC.ResendVerificationByEmail(r.Context(), email); err != nil {
		h.infra.Logger.WithError(err).Warn("restapi - v1 - PostAuthResendVerificationByEmail - ResendVerificationByEmail")
		// Do not return error to client (avoid email enumeration)
	}

	httputil.RenderOK(w, r, response.Message("if the account exists and is unverified, a verification email has been sent"))
}

// PostAuthResetPassword applies a per-token rate limit keyed by the SHA-256 hash
// of the reset token to prevent brute-force guessing of short-lived tokens.
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
		h.OnError(w, r, apperr.ErrTooManyRequests, "PostAuthResetPassword", "ResetPasswordTokenRateLimit")

		return
	}

	err = h.user.EmailUC.ResetPassword(r.Context(), token, newPassword)
	if h.OnError(w, r, err, "PostAuthResetPassword", "ResetPassword") {
		return
	}

	httputil.RenderOK(w, r, response.Message("password reset successfully"))
}

// (POST /auth/resend-verification).
func (h *Server) PostAuthResendVerification(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	// Rate Limit: 10 requests per day per user (resend)
	allowed, err := h.infra.ResendVerificationRateLimiter.Check(r.Context(), user.ID.String())
	if h.OnError(w, r, err, "PostAuthResendVerification", "CheckRateLimit") {
		return
	}

	if !allowed {
		h.OnError(w, r, apperr.ErrTooManyRequests, "PostAuthResendVerification", "RateLimit")

		return
	}

	if h.OnError(w, r, h.user.EmailUC.ResendVerification(r.Context(), user.ID), "PostAuthResendVerification", "ResendVerification") {
		return
	}

	httputil.RenderOK(w, r, response.Message("verification email sent"))
}
