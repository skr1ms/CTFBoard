package v1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	httpMiddleware "github.com/skr1ms/CTFBoard/internal/controller/http/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/http/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/http/v1/response"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type teamRoutes struct {
	teamUC    *usecase.TeamUseCase
	validator validator.Validator
	logger    logger.Interface
}

func NewTeamRoutes(router chi.Router, teamUC *usecase.TeamUseCase, validator validator.Validator, logger logger.Interface, jwtService *jwt.JWTService) {
	routes := teamRoutes{
		teamUC:    teamUC,
		validator: validator,
		logger:    logger,
	}

	router.With(httpMiddleware.Auth(jwtService)).Post("/teams", routes.Create)
	router.With(httpMiddleware.Auth(jwtService)).Post("/teams/join", routes.Join)
	router.With(httpMiddleware.Auth(jwtService)).Get("/teams/my", routes.GetMyTeam)
	router.With(httpMiddleware.Auth(jwtService)).Get("/teams/{id}", routes.GetByID)
}

// @Summary      Create team
// @Description  Creates new team. User becomes captain
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      request.CreateTeamRequest true "Team data"
// @Success      201     {object}  response.TeamResponse
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      409     {object}  ErrorResponse
// @Router       /teams [post]
func (h *teamRoutes) Create(w http.ResponseWriter, r *http.Request) {
	var req request.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("http - v1 - Create - Decode", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.Error("http - v1 - Create - Validate", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	userId := httpMiddleware.GetUserID(r.Context())
	if userId == "" {
		render.Status(r, http.StatusUnauthorized)
		handleError(w, r, nil)
		return
	}

	team, err := h.teamUC.Create(r.Context(), req.Name, userId)
	if err != nil {
		h.logger.Error("http - v1 - Create - Create", err)
		handleError(w, r, err)
		return
	}

	res := response.TeamResponse{
		Id:          team.Id,
		Name:        team.Name,
		InviteToken: team.InviteToken,
		CaptainId:   team.CaptainId,
		CreatedAt:   team.CreatedAt,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, res)
}

// @Summary      Join team
// @Description  Joins team by invite token
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      request.JoinTeamRequest true "Invite token"
// @Success      200     {object}  response.TeamResponse
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      409     {object}  ErrorResponse
// @Router       /teams/join [post]
func (h *teamRoutes) Join(w http.ResponseWriter, r *http.Request) {
	var req request.JoinTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("http - v1 - Join - Decode", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.Error("http - v1 - Join - Validate", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	userId := httpMiddleware.GetUserID(r.Context())
	if userId == "" {
		render.Status(r, http.StatusUnauthorized)
		handleError(w, r, nil)
		return
	}

	team, err := h.teamUC.Join(r.Context(), req.InviteToken, userId)
	if err != nil {
		h.logger.Error("http - v1 - Join - Join", err)
		handleError(w, r, err)
		return
	}

	res := response.TeamResponse{
		Id:          team.Id,
		Name:        team.Name,
		InviteToken: team.InviteToken,
		CaptainId:   team.CaptainId,
		CreatedAt:   team.CreatedAt,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

// @Summary      Get my team
// @Description  Returns information about current user's team with members list
// @Tags         Teams
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.TeamWithMembersResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /teams/my [get]
func (h *teamRoutes) GetMyTeam(w http.ResponseWriter, r *http.Request) {
	userId := httpMiddleware.GetUserID(r.Context())
	if userId == "" {
		render.Status(r, http.StatusUnauthorized)
		handleError(w, r, nil)
		return
	}

	team, members, err := h.teamUC.GetMyTeam(r.Context(), userId)
	if err != nil {
		h.logger.Error("http - v1 - GetMyTeam - GetMyTeam", err)
		handleError(w, r, err)
		return
	}

	var memberResponses []response.UserResponse
	for _, member := range members {
		memberResponses = append(memberResponses, response.UserResponse{
			Id:       member.Id,
			Username: member.Username,
			TeamId:   member.TeamId,
			Role:     member.Role,
		})
	}

	res := response.TeamWithMembersResponse{
		Id:          team.Id,
		Name:        team.Name,
		InviteToken: team.InviteToken,
		CaptainId:   team.CaptainId,
		CreatedAt:   team.CreatedAt,
		Members:     memberResponses,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

// @Summary      Get team by ID
// @Description  Returns team information by ID
// @Tags         Teams
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Team ID"
// @Success      200  {object}  response.TeamResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /teams/{id} [get]
func (h *teamRoutes) GetByID(w http.ResponseWriter, r *http.Request) {
	teamId := chi.URLParam(r, "id")
	if teamId == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, nil)
		return
	}

	team, err := h.teamUC.GetByID(r.Context(), teamId)
	if err != nil {
		h.logger.Error("http - v1 - GetByID - GetByID", err)
		handleError(w, r, err)
		return
	}

	res := response.TeamResponse{
		Id:          team.Id,
		Name:        team.Name,
		InviteToken: team.InviteToken,
		CaptainId:   team.CaptainId,
		CreatedAt:   team.CreatedAt,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}
