package v1

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	restapiMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
)

type hintRoutes struct {
	hintUC    *usecase.HintUseCase
	userUC    *usecase.UserUseCase
	validator validator.Validator
	logger    logger.Interface
}

func NewHintRoutes(
	router chi.Router,
	hintUC *usecase.HintUseCase,
	userUC *usecase.UserUseCase,
	validator validator.Validator,
	logger logger.Interface,
	jwtService *jwt.JWTService,
) {
	routes := hintRoutes{
		hintUC:    hintUC,
		userUC:    userUC,
		validator: validator,
		logger:    logger,
	}

	router.Route("/challenges/{challengeId}/hints", func(r chi.Router) {
		r.Use(restapiMiddleware.Auth(jwtService))
		r.Get("/", routes.GetByChallengeID)
		r.Post("/{hintId}/unlock", routes.UnlockHint)
	})

	router.Route("/admin/challenges/{challengeId}/hints", func(r chi.Router) {
		r.Use(restapiMiddleware.Auth(jwtService))
		r.Use(restapiMiddleware.Admin)
		r.Post("/", routes.Create)
	})

	router.Route("/admin/hints", func(r chi.Router) {
		r.Use(restapiMiddleware.Auth(jwtService))
		r.Use(restapiMiddleware.Admin)
		r.Put("/{id}", routes.Update)
		r.Delete("/{id}", routes.Delete)
	})
}

// @Summary      Get hints for challenge
// @Description  Returns list of hints for a challenge. Content is hidden for non-unlocked hints.
// @Tags         Hints
// @Produce      json
// @Security     BearerAuth
// @Param        challengeId  path      string  true  "Challenge ID"
// @Success      200          {array}   response.HintResponse
// @Failure      401          {object}  ErrorResponse
// @Router       /challenges/{challengeId}/hints [get]
func (h *hintRoutes) GetByChallengeID(w http.ResponseWriter, r *http.Request) {
	challengeId := chi.URLParam(r, "challengeId")
	if challengeId == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, errors.New("challenge id is required"))
		return
	}

	userId := r.Context().Value(restapiMiddleware.UserIDKey).(string)
	user, err := h.userUC.GetByID(r.Context(), userId)
	if err != nil {
		h.logger.Error("restapi - v1 - GetByChallengeID - GetByID", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	hints, err := h.hintUC.GetByChallengeID(r.Context(), challengeId, user.TeamId)
	if err != nil {
		h.logger.Error("restapi - v1 - GetByChallengeID - GetByChallengeID", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	res := make([]response.HintResponse, 0, len(hints))
	for _, hint := range hints {
		hr := response.HintResponse{
			Id:         hint.Hint.Id,
			Cost:       hint.Hint.Cost,
			OrderIndex: hint.Hint.OrderIndex,
			Unlocked:   hint.Unlocked,
		}
		if hint.Unlocked {
			hr.Content = &hint.Hint.Content
		}
		res = append(res, hr)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

// @Summary      Unlock hint
// @Description  Unlocks a hint by spending team points
// @Tags         Hints
// @Produce      json
// @Security     BearerAuth
// @Param        challengeId  path      string  true  "Challenge ID"
// @Param        hintId       path      string  true  "Hint ID"
// @Success      200          {object}  response.HintResponse
// @Failure      400          {object}  ErrorResponse
// @Failure      401          {object}  ErrorResponse
// @Failure      402          {object}  ErrorResponse  "Insufficient points"
// @Failure      404          {object}  ErrorResponse
// @Failure      409          {object}  ErrorResponse  "Already unlocked"
// @Router       /challenges/{challengeId}/hints/{hintId}/unlock [post]
func (h *hintRoutes) UnlockHint(w http.ResponseWriter, r *http.Request) {
	hintId := chi.URLParam(r, "hintId")
	if hintId == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, errors.New("hint id is required"))
		return
	}

	userId := restapiMiddleware.GetUserID(r.Context())
	user, err := h.userUC.GetByID(r.Context(), userId)
	if err != nil {
		h.logger.Error("restapi - v1 - UnlockHint - GetByID", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	if user.TeamId == nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "user must be in a team"})
		return
	}

	hint, err := h.hintUC.UnlockHint(r.Context(), *user.TeamId, hintId)
	if err != nil {
		if errors.Is(err, entityError.ErrHintNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "hint not found"})
			return
		}
		if errors.Is(err, entityError.ErrHintAlreadyUnlocked) {
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, map[string]string{"error": "hint already unlocked"})
			return
		}
		if errors.Is(err, entityError.ErrInsufficientPoints) {
			render.Status(r, http.StatusPaymentRequired)
			render.JSON(w, r, map[string]string{"error": "insufficient points"})
			return
		}
		h.logger.Error("restapi - v1 - UnlockHint - UnlockHint", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	res := response.HintResponse{
		Id:         hint.Id,
		Cost:       hint.Cost,
		OrderIndex: hint.OrderIndex,
		Content:    &hint.Content,
		Unlocked:   true,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

// @Summary      Create hint
// @Description  Creates a new hint for a challenge. Admin only.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        challengeId  path      string                  true  "Challenge ID"
// @Param        request      body      request.CreateHintRequest  true  "Hint data"
// @Success      201          {object}  response.HintAdminResponse
// @Failure      400          {object}  ErrorResponse
// @Failure      401          {object}  ErrorResponse
// @Failure      403          {object}  ErrorResponse
// @Router       /admin/challenges/{challengeId}/hints [post]
func (h *hintRoutes) Create(w http.ResponseWriter, r *http.Request) {
	challengeId := chi.URLParam(r, "challengeId")
	if challengeId == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, errors.New("challenge id is required"))
		return
	}

	var req request.CreateHintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	hint, err := h.hintUC.Create(r.Context(), challengeId, req.Content, req.Cost, req.OrderIndex)
	if err != nil {
		h.logger.Error("restapi - v1 - Create - Create", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	res := response.HintAdminResponse{
		Id:          hint.Id,
		ChallengeId: hint.ChallengeId,
		Content:     hint.Content,
		Cost:        hint.Cost,
		OrderIndex:  hint.OrderIndex,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, res)
}

// @Summary      Update hint
// @Description  Updates hint data. Admin only.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                  true  "Hint ID"
// @Param        request  body      request.UpdateHintRequest  true  "Hint data"
// @Success      200      {object}  response.HintAdminResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Router       /admin/hints/{id} [put]
func (h *hintRoutes) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, errors.New("hint id is required"))
		return
	}

	var req request.UpdateHintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	if err := h.validator.Validate(req); err != nil {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, err)
		return
	}

	hint, err := h.hintUC.Update(r.Context(), id, req.Content, req.Cost, req.OrderIndex)
	if err != nil {
		if errors.Is(err, entityError.ErrHintNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "hint not found"})
			return
		}
		h.logger.Error("restapi - v1 - Update - Update", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	res := response.HintAdminResponse{
		Id:          hint.Id,
		ChallengeId: hint.ChallengeId,
		Content:     hint.Content,
		Cost:        hint.Cost,
		OrderIndex:  hint.OrderIndex,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

// @Summary      Delete hint
// @Description  Deletes hint. Admin only.
// @Tags         Admin
// @Security     BearerAuth
// @Param        id   path      string  true  "Hint ID"
// @Success      204  "No Content"
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /admin/hints/{id} [delete]
func (h *hintRoutes) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Status(r, http.StatusBadRequest)
		handleError(w, r, errors.New("hint id is required"))
		return
	}

	if err := h.hintUC.Delete(r.Context(), id); err != nil {
		if errors.Is(err, entityError.ErrHintNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "hint not found"})
			return
		}
		h.logger.Error("restapi - v1 - Delete - Delete", err)
		render.Status(r, http.StatusInternalServerError)
		handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusNoContent)
}
