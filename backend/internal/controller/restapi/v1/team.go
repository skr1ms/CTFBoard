package v1

import (
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
	"github.com/skr1ms/CTFBoard/pkg/httputil"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type teamRoutes struct {
	teamUC    *usecase.TeamUseCase
	userUC    *usecase.UserUseCase
	validator validator.Validator
	logger    logger.Logger
}

func NewTeamRoutes(router chi.Router,
	teamUC *usecase.TeamUseCase,
	userUC *usecase.UserUseCase,
	validator validator.Validator,
	logger logger.Logger,
	jwtService *jwt.JWTService,
) {
	routes := teamRoutes{
		teamUC:    teamUC,
		userUC:    userUC,
		validator: validator,
		logger:    logger,
	}

	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.InjectUser(userUC)).Post("/teams", routes.Create)
	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.InjectUser(userUC), httprate.LimitByIP(10, 1*time.Minute)).Post("/teams/join", routes.Join)
	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.InjectUser(userUC)).Post("/teams/leave", routes.Leave)
	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.InjectUser(userUC), httprate.LimitByIP(10, 1*time.Hour)).Post("/teams/transfer-captain", routes.TransferCaptain)
	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.InjectUser(userUC)).Get("/teams/my", routes.GetMyTeam)
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
	req, ok := httputil.DecodeAndValidate[request.CreateTeamRequest](
		w, r, h.validator, h.logger, "Create",
	)
	if !ok {
		return
	}

	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	team, err := h.teamUC.Create(r.Context(), req.Name, user.Id)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Create - Create")
		handleError(w, r, err)
		return
	}

	res := response.FromTeam(team)

	httputil.RenderCreated(w, r, res)
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
	req, ok := httputil.DecodeAndValidate[request.JoinTeamRequest](
		w, r, h.validator, h.logger, "Join",
	)
	if !ok {
		return
	}

	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	inviteTokenUUID, err := uuid.Parse(req.InviteToken)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{Error: "invalid invite token format"})
		return
	}

	team, err := h.teamUC.Join(r.Context(), inviteTokenUUID, user.Id)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Join - Join")
		handleError(w, r, err)
		return
	}

	res := response.FromTeam(team)

	httputil.RenderOK(w, r, res)
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
	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	team, members, err := h.teamUC.GetMyTeam(r.Context(), user.Id)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetMyTeam - GetMyTeam")
		handleError(w, r, err)
		return
	}

	res := response.FromTeamWithMembers(team, members)

	httputil.RenderOK(w, r, res)
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
	teamUUID, ok := httputil.ParseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	team, err := h.teamUC.GetByID(r.Context(), teamUUID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetByID - GetByID")
		handleError(w, r, err)
		return
	}

	res := response.FromTeamWithoutToken(team)

	httputil.RenderOK(w, r, res)
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
	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	if err := h.teamUC.Leave(r.Context(), user.Id); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Leave - Leave")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "Successfully left the team"})
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
	req, ok := httputil.DecodeAndValidate[request.TransferCaptainRequest](
		w, r, h.validator, h.logger, "TransferCaptain",
	)
	if !ok {
		return
	}

	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	newCaptainUUID, err := uuid.Parse(req.NewCaptainId)
	if err != nil {
		httputil.RenderError(w, r, http.StatusBadRequest, "invalid new captain ID format")
		return
	}

	if err := h.teamUC.TransferCaptain(r.Context(), user.Id, newCaptainUUID); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - TransferCaptain")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "Captainship transferred successfully"})
}
