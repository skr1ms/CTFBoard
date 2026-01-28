package v1

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	restapiMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type emailRoutes struct {
	emailUC   *usecase.EmailUseCase
	validator validator.Validator
	logger    logger.Logger
	redis     *redis.Client
}

func NewEmailRoutes(
	authRouter chi.Router,
	emailUC *usecase.EmailUseCase,
	validator validator.Validator,
	logger logger.Logger,
	jwtService *jwt.JWTService,
	redisClient *redis.Client,
) {
	routes := emailRoutes{
		emailUC:   emailUC,
		validator: validator,
		logger:    logger,
		redis:     redisClient,
	}

	authRouter.Get("/verify-email", routes.VerifyEmail)
	authRouter.Post("/forgot-password", routes.ForgotPassword)
	authRouter.Post("/reset-password", routes.ResetPassword)
	authRouter.With(restapiMiddleware.Auth(jwtService)).Post("/resend-verification", routes.ResendVerification)
}

// @Summary      Verify email
// @Description  Verifies user email using token from query parameter
// @Tags         Authentication
// @Produce      json
// @Param        token query string true "Verification token"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      410  {object}  ErrorResponse
// @Router       /auth/verify-email [get]
func (h *emailRoutes) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		httputil.RenderError(w, r, http.StatusBadRequest, "token is required")
		return
	}

	err := h.emailUC.VerifyEmail(r.Context(), token)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - VerifyEmail")

		switch err {
		case entityError.ErrTokenNotFound:
			httputil.RenderError(w, r, http.StatusNotFound, "invalid token")
		case entityError.ErrTokenExpired:
			httputil.RenderError(w, r, http.StatusGone, "token expired")
		case entityError.ErrTokenAlreadyUsed:
			httputil.RenderError(w, r, http.StatusConflict, "token already used")
		default:
			httputil.RenderError(w, r, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "email verified successfully"})
}

// @Summary      Request password reset
// @Description  Sends password reset email to specified address
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body request.ForgotPasswordRequest true "Email address"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      429  {object}  ErrorResponse
// @Router       /auth/forgot-password [post]
func (h *emailRoutes) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	// Rate Limit: 10 requests per day per IP (forgot password)
	ip := httputil.GetClientIP(r)
	allowed, err := restapiMiddleware.CheckRateLimit(r.Context(), h.redis, "forgot", ip, 10, 24*time.Hour)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - ForgotPassword - CheckRateLimit")
		httputil.RenderError(w, r, http.StatusInternalServerError, "rate limit check failed")
		return
	}
	if !allowed {
		httputil.RenderError(w, r, http.StatusTooManyRequests, "too many requests")
		return
	}

	req, ok := httputil.DecodeAndValidate[request.ForgotPasswordRequest](
		w, r, h.validator, h.logger, "ForgotPassword",
	)
	if !ok {
		return
	}

	if err := h.emailUC.SendPasswordResetEmail(r.Context(), req.Email); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - ForgotPassword - SendPasswordResetEmail")
	}

	httputil.RenderOK(w, r, map[string]string{"message": "if an account exists with this email, a password reset link has been sent"})
}

// @Summary      Reset password
// @Description  Resets password using token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body request.ResetPasswordRequest true "Token and new password"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      410  {object}  ErrorResponse
// @Router       /auth/reset-password [post]
func (h *emailRoutes) ResetPassword(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.ResetPasswordRequest](
		w, r, h.validator, h.logger, "ResetPassword",
	)
	if !ok {
		return
	}

	err := h.emailUC.ResetPassword(r.Context(), req.Token, req.NewPassword)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - ResetPassword")

		switch err {
		case entityError.ErrTokenNotFound:
			httputil.RenderError(w, r, http.StatusNotFound, "invalid token")
		case entityError.ErrTokenExpired:
			httputil.RenderError(w, r, http.StatusGone, "token expired")
		case entityError.ErrTokenAlreadyUsed:
			httputil.RenderError(w, r, http.StatusConflict, "token already used")
		default:
			httputil.RenderError(w, r, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "password reset successfully"})
}

// @Summary      Resend verification email
// @Description  Resends verification email to authenticated user
// @Tags         Authentication
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  ErrorResponse
// @Router       /auth/resend-verification [post]
func (h *emailRoutes) ResendVerification(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := httputil.ParseAuthUserID(w, r)
	if !ok {
		return
	}

	// Rate Limit: 10 requests per day per user (resend)
	allowed, err := restapiMiddleware.CheckRateLimit(r.Context(), h.redis, "resend", userUUID.String(), 10, 24*time.Hour)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - ResendVerification - CheckRateLimit")
		httputil.RenderError(w, r, http.StatusInternalServerError, "rate limit check failed")
		return
	}
	if !allowed {
		httputil.RenderError(w, r, http.StatusTooManyRequests, "too many requests")
		return
	}

	if err := h.emailUC.ResendVerification(r.Context(), userUUID); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - ResendVerification")
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to resend verification email")
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "verification email sent"})
}
