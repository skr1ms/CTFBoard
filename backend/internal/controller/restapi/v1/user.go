package v1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	restapiMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type userRoutes struct {
	userUC    *usecase.UserUseCase
	emailUC   *usecase.EmailUseCase
	validator validator.Validator
	logger    logger.Interface
}

func NewUserRoutes(router chi.Router,
	authRouter chi.Router,
	userUC *usecase.UserUseCase,
	emailUC *usecase.EmailUseCase,
	validator validator.Validator,
	logger logger.Interface,
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
	authRouter.With(restapiMiddleware.Auth(jwtService)).Get("/me", routes.Me)

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
	var req request.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("restapi - v1 - Register - Decode", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.Error("restapi - v1 - Register - Validate", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	user, err := h.userUC.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		h.logger.Error("restapi - v1 - Register - Register", err)
		render.Status(r, http.StatusConflict)
		handleError(w, r, err)
		return
	}

	if err := h.emailUC.SendVerificationEmail(r.Context(), user); err != nil {
		h.logger.Error("restapi - v1 - Register - SendVerificationEmail", err)
	}

	res := response.RegisterResponse{
		Id:       user.Id.String(),
		Username: user.Username,
		Email:    user.Email,
		CreateAt: user.CreatedAt,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, res)
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
	var req request.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("restapi - v1 - Login - Decode", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.Error("restapi - v1 - Login - Validate", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	tokenPair, err := h.userUC.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.Error("restapi - v1 - Login - Login", err)
		render.Status(r, http.StatusUnauthorized)
		handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, tokenPair)
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
	userId := restapiMiddleware.GetUserID(r.Context())
	if userId == "" {
		render.Status(r, http.StatusUnauthorized)
		handleError(w, r, nil)
		return
	}

	userUUID, err := uuid.Parse(userId)
	if err != nil {
		RenderInvalidID(w, r)
		return
	}

	user, err := h.userUC.GetByID(r.Context(), userUUID)
	if err != nil {
		h.logger.Error("restapi - v1 - Me - GetByID", err)
		handleError(w, r, err)
		return
	}

	var teamIdStr *string
	if user.TeamId != nil {
		s := user.TeamId.String()
		teamIdStr = &s
	}

	res := response.MeResponse{
		Id:       user.Id.String(),
		Username: user.Username,
		Email:    user.Email,
		TeamId:   teamIdStr,
		CreateAt: user.CreatedAt,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
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
	userId := chi.URLParam(r, "id")
	if userId == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, nil)
		return
	}

	userUUID, err := uuid.Parse(userId)
	if err != nil {
		RenderInvalidID(w, r)
		return
	}

	profile, err := h.userUC.GetProfile(r.Context(), userUUID)
	if err != nil {
		h.logger.Error("restapi - v1 - GetProfile - GetProfile", err)
		handleError(w, r, err)
		return
	}

	var solves []response.SolveResponse
	for _, solve := range profile.Solves {
		solves = append(solves, response.SolveResponse{
			Id:          solve.Id.String(),
			ChallengeId: solve.ChallengeId.String(),
			SolvedAt:    solve.SolvedAt,
		})
	}

	var teamIdStr *string
	if profile.User.TeamId != nil {
		s := profile.User.TeamId.String()
		teamIdStr = &s
	}

	res := response.UserProfileResponse{
		Id:       profile.User.Id.String(),
		Username: profile.User.Username,
		TeamId:   teamIdStr,
		CreateAt: profile.User.CreatedAt,
		Solves:   solves,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}
