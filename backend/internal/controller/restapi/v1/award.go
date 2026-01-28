package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
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

type awardRoutes struct {
	awardUC   *usecase.AwardUseCase
	validator validator.Validator
	logger    logger.Logger
}

func NewAwardRoutes(
	router chi.Router,
	awardUC *usecase.AwardUseCase,
	validator validator.Validator,
	logger logger.Logger,
	jwtService *jwt.JWTService,
) {
	routes := awardRoutes{
		awardUC:   awardUC,
		validator: validator,
		logger:    logger,
	}

	router.Route("/admin/awards", func(r chi.Router) {
		r.Use(restapiMiddleware.Auth(jwtService))
		r.Use(restapiMiddleware.Admin)
		r.Post("/", routes.Create)
		r.Get("/team/{teamId}", routes.GetByTeamID)
	})
}

// @Summary      Create award
// @Description  Creates a new award (bonus or penalty) for a team. Admin only.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      request.CreateAwardRequest true "Award data"
// @Success      201     {object}  response.AwardResponse
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      403     {object}  ErrorResponse
// @Router       /admin/awards [post]
func (h *awardRoutes) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.CreateAwardRequest](
		w, r, h.validator, h.logger, "CreateAward",
	)
	if !ok {
		return
	}

	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	teamUUID, err := uuid.Parse(req.TeamId)
	if err != nil {
		httputil.RenderError(w, r, http.StatusBadRequest, "invalid team_id")
		return
	}

	award, err := h.awardUC.Create(r.Context(), teamUUID, req.Value, req.Description, user.Id)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Award - Create")
		handleError(w, r, err)
		return
	}

	res := response.FromAward(award)

	httputil.RenderCreated(w, r, res)
}

// @Summary      Get awards by team
// @Description  Returns list of awards for a team. Admin only.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        teamId path      string  true  "Team ID"
// @Success      200    {array}   response.AwardResponse
// @Failure      400    {object}  ErrorResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      403    {object}  ErrorResponse
// @Router       /admin/awards/team/{teamId} [get]
func (h *awardRoutes) GetByTeamID(w http.ResponseWriter, r *http.Request) {
	teamUUID, ok := httputil.ParseUUIDParam(w, r, "teamId")
	if !ok {
		return
	}

	awards, err := h.awardUC.GetByTeamID(r.Context(), teamUUID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Award - GetByTeamID")
		handleError(w, r, err)
		return
	}

	res := response.FromAwardList(awards)

	httputil.RenderOK(w, r, res)
}
