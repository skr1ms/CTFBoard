package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Verify email
// (GET /auth/verify-email)
func (h *Server) GetAuthVerifyEmail(w http.ResponseWriter, r *http.Request, params openapi.GetAuthVerifyEmailParams) {
	err := h.user.EmailUC.VerifyEmail(r.Context(), params.Token)
	if h.OnError(w, r, err, "GetAuthVerifyEmail", "VerifyEmail") {
		return
	}

	helper.RenderOK(w, r, response.Message("email verified successfully"))
}

// Request password reset
// (POST /auth/forgot-password)
func (h *Server) PostAuthForgotPassword(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.ForgotPasswordRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAuthForgotPassword",
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
		h.OnError(w, r, helper.ErrTooManyRequests, "PostAuthForgotPassword", "RateLimit")
		return
	}
	if err := h.user.EmailUC.SendPasswordResetEmail(r.Context(), email); err != nil {
		h.infra.Logger.WithError(err).Warn("restapi - v1 - PostAuthForgotPassword - SendPasswordResetEmail")
		// Do not return error to client (avoid email enumeration)
	}

	helper.RenderOK(w, r, response.Message("if an account exists with this email, a password reset link has been sent"))
}

// Reset password
// (POST /auth/reset-password)
func (h *Server) PostAuthResetPassword(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.ResetPasswordRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAuthResetPassword",
	)
	if !ok {
		return
	}

	token, newPassword := request.ResetPasswordRequestToParams(&req)
	err := h.user.EmailUC.ResetPassword(r.Context(), token, newPassword)
	if h.OnError(w, r, err, "PostAuthResetPassword", "ResetPassword") {
		return
	}

	helper.RenderOK(w, r, response.Message("password reset successfully"))
}

// Resend verification email
// (POST /auth/resend-verification)
func (h *Server) PostAuthResendVerification(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	// Rate Limit: 10 requests per day per user (resend)
	allowed, err := h.infra.ResendVerificationRateLimiter.Check(r.Context(), userID.String())
	if h.OnError(w, r, err, "PostAuthResendVerification", "CheckRateLimit") {
		return
	}
	if !allowed {
		h.OnError(w, r, helper.ErrTooManyRequests, "PostAuthResendVerification", "RateLimit")
		return
	}

	if h.OnError(w, r, h.user.EmailUC.ResendVerification(r.Context(), userID), "PostAuthResendVerification", "ResendVerification") {
		return
	}

	helper.RenderOK(w, r, response.Message("verification email sent"))
}
