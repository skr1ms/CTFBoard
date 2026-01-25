package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	restapiMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type userRoutes struct {
	userUC    *usecase.UserUseCase
	emailUC   *usecase.EmailUseCase
	validator validator.Validator
	logger    logger.Logger
}

func NewUserRoutes(router chi.Router,
	authRouter chi.Router,
	userUC *usecase.UserUseCase,
	emailUC *usecase.EmailUseCase,
	validator validator.Validator,
	logger logger.Logger,
	jwtService *jwt.JWTService,
) {
	routes := userRoutes{
		userUC:    userUC,
		emailUC:   emailUC,
		validator: validator,
		logger:    logger,
	}

	authRouter.Post("/register", routes.Register)
	authRouter.Post("/login", routes.Login)
	authRouter.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.InjectUser(userUC)).Get("/me", routes.Me)

	router.Get("/users/{id}", routes.GetProfile)
}

// @Summary      Register new user
// @Description  Creates a new user in the system
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body request.RegisterRequest true "Registration data"
// @Success      201  {object}  response.RegisterResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      409  {object}  ErrorResponse
// @Router       /auth/register [post]
func (h *userRoutes) Register(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.RegisterRequest](
		w, r, h.validator, h.logger, "Register",
	)
	if !ok {
		return
	}

	user, err := h.userUC.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Register - Register")
		handleError(w, r, err)
		return
	}

	if err := h.emailUC.SendVerificationEmail(r.Context(), user); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Register - SendVerificationEmail")
	}

	res := response.FromUserForRegister(user)

	httputil.RenderCreated(w, r, res)
}

// @Summary      User login
// @Description  Authenticates user and returns JWT tokens
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body request.LoginRequest true "Credentials"
// @Success      200  {object}  jwt.TokenPair
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /auth/login [post]
func (h *userRoutes) Login(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.LoginRequest](
		w, r, h.validator, h.logger, "Login",
	)
	if !ok {
		return
	}

	tokenPair, err := h.userUC.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Login - Login")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, tokenPair)
}

// @Summary      Get current user info
// @Description  Returns information about authenticated user
// @Tags         Authentication
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.MeResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /auth/me [get]
func (h *userRoutes) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "user not found in context")
		return
	}

	res := response.FromUserForMe(user)

	httputil.RenderOK(w, r, res)
}

// @Summary      Get user profile
// @Description  Returns public information about user and their solved challenges
// @Tags         Users
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  response.UserProfileResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /users/{id} [get]
func (h *userRoutes) GetProfile(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := httputil.ParseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	profile, err := h.userUC.GetProfile(r.Context(), userUUID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetProfile - GetProfile")
		handleError(w, r, err)
		return
	}

	res := response.FromUserProfile(profile)

	httputil.RenderOK(w, r, res)
}
