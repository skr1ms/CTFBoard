package v1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	restapiMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type competitionRoutes struct {
	competitionUC *usecase.CompetitionUseCase
	validator     validator.Validator
	logger        logger.Interface
}

func NewCompetitionRoutes(router chi.Router, competitionUC *usecase.CompetitionUseCase, validator validator.Validator, logger logger.Interface, jwtService *jwt.JWTService) {
	routes := competitionRoutes{
		competitionUC: competitionUC,
		validator:     validator,
		logger:        logger,
	}

	router.Get("/competition/status", routes.GetStatus)
	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.Admin).Get("/admin/competition", routes.Get)
	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.Admin).Put("/admin/competition", routes.Update)
}

func (h *competitionRoutes) GetStatus(w http.ResponseWriter, r *http.Request) {
	comp, err := h.competitionUC.Get(r.Context())
	if err != nil {
		h.logger.Error("restapi - v1 - GetStatus - Get", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	var startTime, endTime *string
	if comp.StartTime != nil {
		s := comp.StartTime.Format("2006-01-02T15:04:05Z07:00")
		startTime = &s
	}
	if comp.EndTime != nil {
		s := comp.EndTime.Format("2006-01-02T15:04:05Z07:00")
		endTime = &s
	}

	res := response.CompetitionStatusResponse{
		Status:            string(comp.GetStatus()),
		Name:              comp.Name,
		StartTime:         startTime,
		EndTime:           endTime,
		SubmissionAllowed: comp.IsSubmissionAllowed(),
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

func (h *competitionRoutes) Get(w http.ResponseWriter, r *http.Request) {
	comp, err := h.competitionUC.Get(r.Context())
	if err != nil {
		h.logger.Error("restapi - v1 - Get - Get", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	var startTime, endTime, freezeTime *string
	if comp.StartTime != nil {
		s := comp.StartTime.Format("2006-01-02T15:04:05Z07:00")
		startTime = &s
	}
	if comp.EndTime != nil {
		s := comp.EndTime.Format("2006-01-02T15:04:05Z07:00")
		endTime = &s
	}
	if comp.FreezeTime != nil {
		s := comp.FreezeTime.Format("2006-01-02T15:04:05Z07:00")
		freezeTime = &s
	}

	res := response.CompetitionResponse{
		Id:         comp.Id,
		Name:       comp.Name,
		StartTime:  startTime,
		EndTime:    endTime,
		FreezeTime: freezeTime,
		IsPaused:   comp.IsPaused,
		IsPublic:   comp.IsPublic,
		Status:     string(comp.GetStatus()),
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

func (h *competitionRoutes) Update(w http.ResponseWriter, r *http.Request) {
	var req request.UpdateCompetitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("restapi - v1 - Update - Decode", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.Error("restapi - v1 - Update - Validate", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	// Validate time constraints
	if req.EndTime != nil && req.StartTime != nil && req.EndTime.Before(*req.StartTime) {
		h.logger.Error("restapi - v1 - Update - TimeValidation", nil)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "end_time must be after start_time"})
		return
	}

	if req.FreezeTime != nil && req.EndTime != nil && req.FreezeTime.After(*req.EndTime) {
		h.logger.Error("restapi - v1 - Update - TimeValidation", nil)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "freeze_time must be before end_time"})
		return
	}

	if req.FreezeTime != nil && req.StartTime != nil && req.FreezeTime.Before(*req.StartTime) {
		h.logger.Error("restapi - v1 - Update - TimeValidation", nil)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "freeze_time must be after start_time"})
		return
	}

	comp := &entity.Competition{
		Id:         1,
		Name:       req.Name,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		FreezeTime: req.FreezeTime,
		IsPaused:   req.IsPaused,
		IsPublic:   req.IsPublic,
	}

	if err := h.competitionUC.Update(r.Context(), comp); err != nil {
		h.logger.Error("restapi - v1 - Update - Update", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "competition updated"})
}
