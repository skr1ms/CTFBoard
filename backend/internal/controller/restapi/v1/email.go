package v1

import (
	"errors"
	"net/http"
	"time"

	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

// Verify email
// (GET /auth/verify-email)
func (h *Server) GetAuthVerifyEmail(w http.ResponseWriter, r *http.Request, params openapi.GetAuthVerifyEmailParams) {
	if params.Token == "" {
		httputil.RenderError(w, r, http.StatusBadRequest, "token is required")
		return
	}

	err := h.emailUC.VerifyEmail(r.Context(), params.Token)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAuthVerifyEmail")

		if errors.Is(err, entityError.ErrTokenNotFound) {
			httputil.RenderError(w, r, http.StatusNotFound, "invalid token")
		} else if errors.Is(err, entityError.ErrTokenExpired) {
			httputil.RenderError(w, r, http.StatusGone, "token expired")
		} else if errors.Is(err, entityError.ErrTokenAlreadyUsed) {
			httputil.RenderError(w, r, http.StatusConflict, "token already used")
		} else {
			httputil.RenderError(w, r, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "email verified successfully"})
}

// Request password reset
// (POST /auth/forgot-password)
func (h *Server) PostAuthForgotPassword(w http.ResponseWriter, r *http.Request) {
	// Rate Limit: 10 requests per day per IP (forgot password)
	ip := httputil.GetClientIP(r)
	allowed, err := restapimiddleware.CheckRateLimit(r.Context(), h.redisClient, "forgot", ip, 10, 24*time.Hour)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthForgotPassword - CheckRateLimit")
		httputil.RenderError(w, r, http.StatusInternalServerError, "rate limit check failed")
		return
	}
	if !allowed {
		httputil.RenderError(w, r, http.StatusTooManyRequests, "too many requests")
		return
	}

	req, ok := httputil.DecodeAndValidate[request.ForgotPasswordRequest](
		w, r, h.validator, h.logger, "PostAuthForgotPassword",
	)
	if !ok {
		return
	}

	if err := h.emailUC.SendPasswordResetEmail(r.Context(), req.Email); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthForgotPassword - SendPasswordResetEmail")
	}

	httputil.RenderOK(w, r, map[string]string{"message": "if an account exists with this email, a password reset link has been sent"})
}

// Reset password
// (POST /auth/reset-password)
func (h *Server) PostAuthResetPassword(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.ResetPasswordRequest](
		w, r, h.validator, h.logger, "PostAuthResetPassword",
	)
	if !ok {
		return
	}

	err := h.emailUC.ResetPassword(r.Context(), req.Token, req.NewPassword)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthResetPassword")

		if errors.Is(err, entityError.ErrTokenNotFound) {
			httputil.RenderError(w, r, http.StatusNotFound, "invalid token")
		} else if errors.Is(err, entityError.ErrTokenExpired) {
			httputil.RenderError(w, r, http.StatusGone, "token expired")
		} else if errors.Is(err, entityError.ErrTokenAlreadyUsed) {
			httputil.RenderError(w, r, http.StatusConflict, "token already used")
		} else {
			httputil.RenderError(w, r, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "password reset successfully"})
}

// Resend verification email
// (POST /auth/resend-verification)
func (h *Server) PostAuthResendVerification(w http.ResponseWriter, r *http.Request) {
	useruuid, ok := httputil.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	// Rate Limit: 10 requests per day per user (resend)
	allowed, err := restapimiddleware.CheckRateLimit(r.Context(), h.redisClient, "resend", useruuid.String(), 10, 24*time.Hour)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthResendVerification - CheckRateLimit")
		httputil.RenderError(w, r, http.StatusInternalServerError, "rate limit check failed")
		return
	}
	if !allowed {
		httputil.RenderError(w, r, http.StatusTooManyRequests, "too many requests")
		return
	}

	if err := h.emailUC.ResendVerification(r.Context(), useruuid); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAuthResendVerification")
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to resend verification email")
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "verification email sent"})
}
