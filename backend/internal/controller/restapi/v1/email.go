package v1

import (
	"errors"
	"net/http"
	"time"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Verify email
// (GET /auth/verify-email)
func (h *Server) GetAuthVerifyEmail(w http.ResponseWriter, r *http.Request, params openapi.GetAuthVerifyEmailParams) {
	if params.Token == "" {
		RenderError(w, r, http.StatusBadRequest, "token is required")
		return
	}

	err := h.emailUC.VerifyEmail(r.Context(), params.Token)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAuthVerifyEmail")

		if errors.Is(err, entityError.ErrTokenNotFound) {
			RenderError(w, r, http.StatusNotFound, "invalid token")
		} else if errors.Is(err, entityError.ErrTokenExpired) {
			RenderError(w, r, http.StatusGone, "token expired")
		} else if errors.Is(err, entityError.ErrTokenAlreadyUsed) {
			RenderError(w, r, http.StatusConflict, "token already used")
		} else {
			RenderError(w, r, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	RenderOK(w, r, map[string]string{"message": "email verified successfully"})
}

// Request password reset
// (POST /auth/forgot-password)
func (h *Server) PostAuthForgotPassword(w http.ResponseWriter, r *http.Request) {
	// Rate Limit: 10 requests per day per IP (forgot password)
	ip := GetClientIP(r)
	allowed, err := middleware.CheckRateLimit(r.Context(), h.redisClient, "forgot", ip, 10, 24*time.Hour)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthForgotPassword - CheckRateLimit")
		RenderError(w, r, http.StatusInternalServerError, "rate limit check failed")
		return
	}
	if !allowed {
		RenderError(w, r, http.StatusTooManyRequests, "too many requests")
		return
	}

	req, ok := DecodeAndValidate[openapi.RequestForgotPasswordRequest](
		w, r, h.validator, h.logger, "PostAuthForgotPassword",
	)
	if !ok {
		return
	}

	email := request.ForgotPasswordRequestEmail(&req)
	if err := h.emailUC.SendPasswordResetEmail(r.Context(), email); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthForgotPassword - SendPasswordResetEmail")
	}

	RenderOK(w, r, map[string]string{"message": "if an account exists with this email, a password reset link has been sent"})
}

// Reset password
// (POST /auth/reset-password)
func (h *Server) PostAuthResetPassword(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestResetPasswordRequest](
		w, r, h.validator, h.logger, "PostAuthResetPassword",
	)
	if !ok {
		return
	}

	token, newPassword := request.ResetPasswordRequestParams(&req)
	err := h.emailUC.ResetPassword(r.Context(), token, newPassword)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthResetPassword")

		if errors.Is(err, entityError.ErrTokenNotFound) {
			RenderError(w, r, http.StatusNotFound, "invalid token")
		} else if errors.Is(err, entityError.ErrTokenExpired) {
			RenderError(w, r, http.StatusGone, "token expired")
		} else if errors.Is(err, entityError.ErrTokenAlreadyUsed) {
			RenderError(w, r, http.StatusConflict, "token already used")
		} else {
			RenderError(w, r, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	RenderOK(w, r, map[string]string{"message": "password reset successfully"})
}

// Resend verification email
// (POST /auth/resend-verification)
func (h *Server) PostAuthResendVerification(w http.ResponseWriter, r *http.Request) {
	useruuid, ok := ParseAuthUserID(w, r)
	if !ok {
		return
	}

	// Rate Limit: 10 requests per day per user (resend)
	allowed, err := middleware.CheckRateLimit(r.Context(), h.redisClient, "resend", useruuid.String(), 10, 24*time.Hour)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthResendVerification - CheckRateLimit")
		RenderError(w, r, http.StatusInternalServerError, "rate limit check failed")
		return
	}
	if !allowed {
		RenderError(w, r, http.StatusTooManyRequests, "too many requests")
		return
	}

	if err := h.emailUC.ResendVerification(r.Context(), useruuid); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthResendVerification")
		RenderError(w, r, http.StatusInternalServerError, "failed to resend verification email")
		return
	}

	RenderOK(w, r, map[string]string{"message": "verification email sent"})
}
