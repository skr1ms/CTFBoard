package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
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

type teamRoutes struct {
	teamUC    *usecase.TeamUseCase
	validator validator.Validator
	logger    logger.Logger
}

func NewTeamRoutes(router chi.Router,
	teamUC *usecase.TeamUseCase,
	validator validator.Validator,
	logger logger.Logger,
	jwtService *jwt.JWTService,
) {
	routes := teamRoutes{
		teamUC:    teamUC,
		validator: validator,
		logger:    logger,
	}

	router.With(restapiMiddleware.Auth(jwtService)).Post("/teams", routes.Create)
	router.With(restapiMiddleware.Auth(jwtService), httprate.LimitByIP(10, 1*time.Minute)).Post("/teams/join", routes.Join)
	router.With(restapiMiddleware.Auth(jwtService)).Post("/teams/leave", routes.Leave)
	router.With(restapiMiddleware.Auth(jwtService), httprate.LimitByIP(10, 1*time.Hour)).Post("/teams/transfer-captain", routes.TransferCaptain)
	router.With(restapiMiddleware.Auth(jwtService)).Get("/teams/my", routes.GetMyTeam)
	router.With(restapiMiddleware.Auth(jwtService)).Get("/teams/{id}", routes.GetByID)
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
		h.logger.WithError(err).Error("restapi - v1 - Create - Decode")
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Create - Validate")
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

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

	team, err := h.teamUC.Create(r.Context(), req.Name, userUUID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Create - Create")
		handleError(w, r, err)
		return
	}

	res := response.TeamResponse{
		Id:          team.Id.String(),
		Name:        team.Name,
		InviteToken: team.InviteToken.String(),
		CaptainId:   team.CaptainId.String(),
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
		h.logger.WithError(err).Error("restapi - v1 - Join - Decode")
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Join - Validate")
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

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

	inviteTokenUUID, err := uuid.Parse(req.InviteToken)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "invalid invite token format"})
		return
	}

	team, err := h.teamUC.Join(r.Context(), inviteTokenUUID, userUUID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Join - Join")
		handleError(w, r, err)
		return
	}

	res := response.TeamResponse{
		Id:          team.Id.String(),
		Name:        team.Name,
		InviteToken: team.InviteToken.String(),
		CaptainId:   team.CaptainId.String(),
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

	team, members, err := h.teamUC.GetMyTeam(r.Context(), userUUID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetMyTeam - GetMyTeam")
		handleError(w, r, err)
		return
	}

	var memberResponses []response.UserResponse
	for _, member := range members {
		var teamIdStr *string
		if member.TeamId != nil {
			s := member.TeamId.String()
			teamIdStr = &s
		}
		memberResponses = append(memberResponses, response.UserResponse{
			Id:       member.Id.String(),
			Username: member.Username,
			TeamId:   teamIdStr,
			Role:     member.Role,
		})
	}

	res := response.TeamWithMembersResponse{
		Id:          team.Id.String(),
		Name:        team.Name,
		InviteToken: team.InviteToken.String(),
		CaptainId:   team.CaptainId.String(),
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

	teamUUID, err := uuid.Parse(teamId)
	if err != nil {
		RenderInvalidID(w, r)
		return
	}

	team, err := h.teamUC.GetByID(r.Context(), teamUUID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetByID - GetByID")
		handleError(w, r, err)
		return
	}

	// Don't expose invite token when getting team by ID (security)
	res := response.TeamResponse{
		Id:        team.Id.String(),
		Name:      team.Name,
		CaptainId: team.CaptainId.String(),
		CreatedAt: team.CreatedAt,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

// @Summary      Leave team
// @Description  Leave current team. Captain cannot leave, must transfer captainship first
// @Tags         Teams
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      409  {object}  ErrorResponse
// @Router       /teams/leave [post]
func (h *teamRoutes) Leave(w http.ResponseWriter, r *http.Request) {
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

	if err := h.teamUC.Leave(r.Context(), userUUID); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Leave - Leave")
		handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "Successfully left the team"})
}

// @Summary      Transfer captainship
// @Description  Transfer team captain role to another team member
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      request.TransferCaptainRequest true "New captain ID"
// @Success      200     {object}  map[string]string
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      403     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Router       /teams/transfer-captain [post]
func (h *teamRoutes) TransferCaptain(w http.ResponseWriter, r *http.Request) {
	var req request.TransferCaptainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - TransferCaptain - Decode")
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - TransferCaptain - Validate")
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

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

	newCaptainUUID, err := uuid.Parse(req.NewCaptainId)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "invalid new captain ID format"})
		return
	}

	if err := h.teamUC.TransferCaptain(r.Context(), userUUID, newCaptainUUID); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - TransferCaptain - TransferCaptain")
		handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "Captainship transferred successfully"})
}
