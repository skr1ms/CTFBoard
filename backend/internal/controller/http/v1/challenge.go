package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/go-chi/render"
	httpMiddleware "github.com/skr1ms/CTFBoard/internal/controller/http/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/http/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/http/v1/response"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type challengeRoutes struct {
	challengeUC   *usecase.ChallengeUseCase
	solveUC       *usecase.SolveUseCase
	userUC        *usecase.UserUseCase
	competitionUC *usecase.CompetitionUseCase
	validator     validator.Validator
	logger        logger.Interface
}

func NewChallengeRoutes(router chi.Router,
	challengeUC *usecase.ChallengeUseCase,
	solveUC *usecase.SolveUseCase,
	userUC *usecase.UserUseCase,
	competitionUC *usecase.CompetitionUseCase,
	validator validator.Validator,
	logger logger.Interface,
	jwtService *jwt.JWTService,
	submitLimit int,
	durationLimit time.Duration,
) {
	routes := challengeRoutes{
		challengeUC:   challengeUC,
		solveUC:       solveUC,
		userUC:        userUC,
		competitionUC: competitionUC,
		validator:     validator,
		logger:        logger,
	}

	router.With(httpMiddleware.Auth(jwtService)).Get("/challenges", routes.GetAll)
	router.With(httpMiddleware.Auth(jwtService), httpMiddleware.CompetitionActive(competitionUC), httprate.LimitByIP(submitLimit, durationLimit)).Post("/challenges/{id}/submit", routes.SubmitFlag)
	router.With(httpMiddleware.Auth(jwtService), httpMiddleware.Admin).Post("/admin/challenges", routes.Create)
	router.With(httpMiddleware.Auth(jwtService), httpMiddleware.Admin).Put("/admin/challenges/{id}", routes.Update)
	router.With(httpMiddleware.Auth(jwtService), httpMiddleware.Admin).Delete("/admin/challenges/{id}", routes.Delete)
}

// @Summary      Get challenges list
// @Description  Returns list of all challenges with solved status for user's team
// @Tags         Challenges
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   response.ChallengeResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /challenges [get]
func (h *challengeRoutes) GetAll(w http.ResponseWriter, r *http.Request) {
	userId := httpMiddleware.GetUserID(r.Context())
	if userId == "" {
		render.Status(r, http.StatusUnauthorized)
		handleError(w, r, nil)
		return
	}

	user, err := h.userUC.GetByID(r.Context(), userId)
	if err != nil {
		h.logger.Error("http - v1 - GetAll - GetByID", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	challenges, err := h.challengeUC.GetAll(r.Context(), user.TeamId)
	if err != nil {
		h.logger.Error("http - v1 - GetAll - GetAll", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	res := make([]response.ChallengeResponse, 0)
	for _, cws := range challenges {
		res = append(res, response.ChallengeResponse{
			Id:          cws.Challenge.Id,
			Title:       cws.Challenge.Title,
			Description: cws.Challenge.Description,
			Category:    cws.Challenge.Category,
			Points:      cws.Challenge.Points,
			SolveCount:  cws.Challenge.SolveCount,
			IsHidden:    cws.Challenge.IsHidden,
			Solved:      cws.Solved,
		})
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

// @Summary      Submit flag
// @Description  Verifies flag for challenge. Rate limit: 5 attempts per minute
// @Tags         Challenges
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string                  true  "Challenge ID"
// @Param        request body      request.SubmitFlagRequest true "Flag to verify"
// @Success      200     {object}  map[string]string
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      403     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      409     {object}  ErrorResponse
// @Failure      429     {object}  ErrorResponse
// @Router       /challenges/{id}/submit [post]
func (h *challengeRoutes) SubmitFlag(w http.ResponseWriter, r *http.Request) {
	challengeId := chi.URLParam(r, "id")
	if challengeId == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, nil)
		return
	}

	var req request.SubmitFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("http - v1 - SubmitFlag - Decode", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.Error("http - v1 - SubmitFlag - Validate", err)
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

	user, err := h.userUC.GetByID(r.Context(), userId)
	if err != nil {
		h.logger.Error("http - v1 - SubmitFlag - GetByID", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	valid, err := h.challengeUC.SubmitFlag(r.Context(), challengeId, req.Flag, userId, user.TeamId)
	if err != nil {
		h.logger.Error("http - v1 - SubmitFlag - SubmitFlag", err)
		handleError(w, r, err)
		return
	}

	if !valid {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid flag"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "flag accepted"})
}

// @Summary      Create challenge
// @Description  Creates new challenge. Admin only
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      request.CreateChallengeRequest true "Challenge data"
// @Success      201     {object}  response.ChallengeResponse
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      403     {object}  ErrorResponse
// @Router       /admin/challenges [post]
func (h *challengeRoutes) Create(w http.ResponseWriter, r *http.Request) {
	var req request.CreateChallengeRequest
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

	challenge, err := h.challengeUC.Create(r.Context(), req.Title, req.Description, req.Category, req.Points, req.InitialValue, req.MinValue, req.Decay, req.Flag, req.IsHidden)
	if err != nil {
		h.logger.Error("http - v1 - Create - Create", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	res := response.ChallengeResponse{
		Id:          challenge.Id,
		Title:       challenge.Title,
		Description: challenge.Description,
		Category:    challenge.Category,
		Points:      challenge.Points,
		SolveCount:  challenge.SolveCount,
		IsHidden:    challenge.IsHidden,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, res)
}

// @Summary      Delete challenge
// @Description  Deletes challenge. Admin only
// @Tags         Admin
// @Security     BearerAuth
// @Param        id   path      string  true  "Challenge ID"
// @Success      204  "No Content"
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /admin/challenges/{id} [delete]
func (h *challengeRoutes) Delete(w http.ResponseWriter, r *http.Request) {
	challengeId := chi.URLParam(r, "id")
	if challengeId == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, nil)
		return
	}

	err := h.challengeUC.Delete(r.Context(), challengeId)
	if err != nil {
		h.logger.Error("http - v1 - Delete - Delete", err)
		render.Status(r, http.StatusNotFound)
		handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusNoContent)
}

// @Summary      Update challenge
// @Description  Updates challenge data. Admin only
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string                    true  "Challenge ID"
// @Param        request body      request.UpdateChallengeRequest true "Challenge data"
// @Success      200     {object}  response.ChallengeResponse
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      403     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Router       /admin/challenges/{id} [put]
func (h *challengeRoutes) Update(w http.ResponseWriter, r *http.Request) {
	challengeId := chi.URLParam(r, "id")
	if challengeId == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, nil)
		return
	}

	var req request.UpdateChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("http - v1 - Update - Decode", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.logger.Error("http - v1 - Update - Validate", err)
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	challenge, err := h.challengeUC.Update(r.Context(), challengeId, req.Title, req.Description, req.Category, req.Points, req.InitialValue, req.MinValue, req.Decay, req.Flag, req.IsHidden)
	if err != nil {
		h.logger.Error("http - v1 - Update - Update", err)
		handleError(w, r, err)
		return
	}

	res := response.ChallengeResponse{
		Id:          challenge.Id,
		Title:       challenge.Title,
		Description: challenge.Description,
		Category:    challenge.Category,
		Points:      challenge.Points,
		SolveCount:  challenge.SolveCount,
		IsHidden:    challenge.IsHidden,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}
