package v1

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	restapiMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
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
	redisClient *redis.Client,
) {
	routes := teamRoutes{
		teamUC:    teamUC,
		userUC:    userUC,
		validator: validator,
		logger:    logger,
	}

	joinLimit := restapiMiddleware.RateLimit(redisClient, "team:join", 10, 1*time.Minute, func(r *http.Request) (string, error) {
		return httputil.GetClientIP(r), nil
	})

	transferLimit := restapiMiddleware.RateLimit(redisClient, "team:transfer", 10, 1*time.Hour, func(r *http.Request) (string, error) {
		return httputil.GetClientIP(r), nil
	})

	router.Post("/teams", routes.Create)
	router.Post("/teams/solo", routes.CreateSolo)
	router.With(joinLimit).Post("/teams/join", routes.Join)
	router.Post("/teams/leave", routes.Leave)
	router.With(transferLimit).Post("/teams/transfer-captain", routes.TransferCaptain)
	router.Delete("/teams/me", routes.Disband)
	router.Delete("/teams/members/{id}", routes.Kick)

	router.Get("/teams/my", routes.GetMyTeam)
	router.Get("/teams/{id}", routes.GetByID)
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

	team, err := h.teamUC.Create(r.Context(), req.Name, user.Id, false, req.ConfirmReset)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Create - Create")
		handleError(w, r, err)
		return
	}

	res := response.FromTeam(team)

	httputil.RenderCreated(w, r, res)
}

// @Summary      Create solo team
// @Description  Creates a solo team for the user (Select Solo Mode)
// @Tags         Teams
// @Produce      json
// @Security     BearerAuth
// @Param        request body      request.CreateTeamRequest false "Confirm reset"
// @Success      200     {object}  response.TeamResponse
// @Failure      400     {object}  ErrorResponse
// @Router       /teams/solo [post]
func (h *teamRoutes) CreateSolo(w http.ResponseWriter, r *http.Request) {
	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	// Optional body for confirmation
	var req request.CreateTeamRequest
	// We don't strictly validate here because name is not required for solo
	_ = httputil.DecodeJSON(r, &req)

	team, err := h.teamUC.CreateSoloTeam(r.Context(), user.Id, req.ConfirmReset)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - CreateSolo")
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
		render.JSON(w, r, map[string]string{"error": "invalid invite token format"})
		return
	}

	team, err := h.teamUC.Join(r.Context(), inviteTokenUUID, user.Id, req.ConfirmReset)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Join - Join")
		handleError(w, r, err)
		return
	}

	res := response.FromTeam(team)

	httputil.RenderOK(w, r, res)
}

// @Summary      Get my team
// @Description  Returns information about current user's team. If no team, returns 404.
// @Tags         Teams
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.TeamWithMembersResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse "User is not in a team"
// @Router       /teams/my [get]
func (h *teamRoutes) GetMyTeam(w http.ResponseWriter, r *http.Request) {
	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	team, members, err := h.teamUC.GetMyTeam(r.Context(), user.Id)
	if err != nil {
		if errors.Is(err, entityError.ErrTeamNotFound) {
			httputil.RenderError(w, r, http.StatusNotFound, "user is not in a team")
			return
		}
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
// @Description  Leave current team
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
// @Description  Transfer team captain role
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

// @Summary      Disband team
// @Description  Disbands the current team (Captain only)
// @Tags         Teams
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Router       /teams/me [delete]
func (h *teamRoutes) Disband(w http.ResponseWriter, r *http.Request) {
	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	if err := h.teamUC.DisbandTeam(r.Context(), user.Id); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Disband")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "Team disbanded successfully"})
}

// @Summary      Kick member
// @Description  Kicks a member from the team (Captain only)
// @Tags         Teams
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Member User ID"
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /teams/members/{id} [delete]
func (h *teamRoutes) Kick(w http.ResponseWriter, r *http.Request) {
	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	targetUserID, ok := httputil.ParseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if err := h.teamUC.KickMember(r.Context(), user.Id, targetUserID); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Kick")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "Member kicked successfully"})
}
