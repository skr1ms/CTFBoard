package v1

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	restapiMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type emailRoutes struct {
	emailUC   *usecase.EmailUseCase
	validator validator.Validator
	logger    logger.Interface
	redis     redis.Client
}

func NewEmailRoutes(
	authRouter chi.Router,
	emailUC *usecase.EmailUseCase,
	validator validator.Validator,
	logger logger.Interface,
	jwtService *jwt.JWTService,
	redisClient redis.Client,
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

// Helper function for rate limiting
func (h *emailRoutes) checkRateLimit(ctx context.Context, key string, limit int, duration time.Duration) bool {
	count, err := h.redis.Incr(ctx, key).Result()
	if err != nil {
		h.logger.Error("emailRoutes - checkRateLimit - Incr", err)
		return true // Fail open to avoid blocking users on redis error
	}

	h.redis.Expire(ctx, key, duration)

	return count <= int64(limit)
}

func getRealIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
		if ip != "" {
			parts := strings.Split(ip, ",")
			ip = strings.TrimSpace(parts[0])
		}
	}
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
		if ip == "" {
			ip = r.RemoteAddr
		}
	}
	return ip
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "token is required"})
		return
	}

	err := h.emailUC.VerifyEmail(r.Context(), token)
	if err != nil {
		h.logger.Error("restapi - v1 - VerifyEmail", err)

		switch err {
		case entityError.ErrTokenNotFound:
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, ErrorResponse{Error: "invalid token"})
		case entityError.ErrTokenExpired:
			render.Status(r, http.StatusGone)
			render.JSON(w, r, ErrorResponse{Error: "token expired"})
		case entityError.ErrTokenAlreadyUsed:
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, ErrorResponse{Error: "token already used"})
		default:
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, ErrorResponse{Error: "internal server error"})
		}
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "email verified successfully"})
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
	// Rate Limit: 10 requests per day per user (forgot password)
	ip := getRealIP(r)
	key := "ratelimit:forgot:" + ip
	if !h.checkRateLimit(r.Context(), key, 10, 24*time.Hour) {
		render.Status(r, http.StatusTooManyRequests)
		render.JSON(w, r, ErrorResponse{Error: "too many requests"})
		return
	}

	var req request.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("restapi - v1 - ForgotPassword - Decode", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.Error("restapi - v1 - ForgotPassword - Validate", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.emailUC.SendPasswordResetEmail(r.Context(), req.Email); err != nil {
		h.logger.Error("restapi - v1 - ForgotPassword - SendPasswordResetEmail", err)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "if an account exists with this email, a password reset link has been sent"})
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
	var req request.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("restapi - v1 - ResetPassword - Decode", err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.Error("restapi - v1 - ResetPassword - Validate", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	err := h.emailUC.ResetPassword(r.Context(), req.Token, req.NewPassword)
	if err != nil {
		h.logger.Error("restapi - v1 - ResetPassword", err)

		switch err {
		case entityError.ErrTokenNotFound:
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, ErrorResponse{Error: "invalid token"})
		case entityError.ErrTokenExpired:
			render.Status(r, http.StatusGone)
			render.JSON(w, r, ErrorResponse{Error: "token expired"})
		case entityError.ErrTokenAlreadyUsed:
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, ErrorResponse{Error: "token already used"})
		default:
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, ErrorResponse{Error: "internal server error"})
		}
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "password reset successfully"})
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
	userId := restapiMiddleware.GetUserID(r.Context())
	if userId == "" {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, ErrorResponse{Error: "unauthorized"})
		return
	}

	// Rate Limit: 10 requests per day per user (resend)
	key := "ratelimit:resend:" + userId
	if !h.checkRateLimit(r.Context(), key, 10, 24*time.Hour) {
		render.Status(r, http.StatusTooManyRequests)
		render.JSON(w, r, ErrorResponse{Error: "too many requests"})
		return
	}

	if err := h.emailUC.ResendVerification(r.Context(), userId); err != nil {
		h.logger.Error("restapi - v1 - ResendVerification", err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, ErrorResponse{Error: "failed to resend verification email"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "verification email sent"})
}
