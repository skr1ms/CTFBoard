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

type competitionRoutes struct {
	competitionUC *usecase.CompetitionUseCase
	userUC        *usecase.UserUseCase
	validator     validator.Validator
	logger        logger.Logger
}

func NewCompetitionRoutes(
	publicRouter chi.Router,
	protectedRouter chi.Router,
	competitionUC *usecase.CompetitionUseCase,
	userUC *usecase.UserUseCase,
	validator validator.Validator,
	logger logger.Logger,
	jwtService *jwt.JWTService,
) {
	routes := competitionRoutes{
		competitionUC: competitionUC,
		userUC:        userUC,
		validator:     validator,
		logger:        logger,
	}

	publicRouter.Get("/competition/status", routes.GetStatus)

	protectedRouter.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.InjectUser(userUC), restapiMiddleware.Admin).Get("/admin/competition", routes.Get)
	protectedRouter.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.InjectUser(userUC), restapiMiddleware.Admin).Put("/admin/competition", routes.Update)
}

func (h *competitionRoutes) GetStatus(w http.ResponseWriter, r *http.Request) {
	comp, err := h.competitionUC.Get(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetStatus - Get")
		handleError(w, r, err)
		return
	}

	res := response.FromCompetitionStatus(comp)

	httputil.RenderOK(w, r, res)
}

func (h *competitionRoutes) Get(w http.ResponseWriter, r *http.Request) {
	comp, err := h.competitionUC.Get(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Get - Get")
		handleError(w, r, err)
		return
	}

	res := response.FromCompetition(comp)

	httputil.RenderOK(w, r, res)
}

func (h *competitionRoutes) Update(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.UpdateCompetitionRequest](
		w, r, h.validator, h.logger, "UpdateCompetition",
	)
	if !ok {
		return
	}

	if req.EndTime != nil && req.StartTime != nil && req.EndTime.Before(*req.StartTime) {
		h.logger.Error("restapi - v1 - Update - TimeValidation", nil)
		httputil.RenderError(w, r, http.StatusBadRequest, "end_time must be after start_time")
		return
	}

	if req.FreezeTime != nil && req.EndTime != nil && req.FreezeTime.After(*req.EndTime) {
		h.logger.Error("restapi - v1 - Update - TimeValidation", nil)
		httputil.RenderError(w, r, http.StatusBadRequest, "freeze_time must be before end_time")
		return
	}

	if req.FreezeTime != nil && req.StartTime != nil && req.FreezeTime.Before(*req.StartTime) {
		h.logger.Error("restapi - v1 - Update - TimeValidation", nil)
		httputil.RenderError(w, r, http.StatusBadRequest, "freeze_time must be after start_time")
		return
	}

	comp := req.ToCompetition(1)

	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	clientIP := httputil.GetClientIP(r)

	if err := h.competitionUC.Update(r.Context(), comp, user.Id, clientIP); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Update - Update")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "competition updated"})
}
