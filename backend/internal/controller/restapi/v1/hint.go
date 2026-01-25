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

type hintRoutes struct {
	hintUC    *usecase.HintUseCase
	userUC    *usecase.UserUseCase
	validator validator.Validator
	logger    logger.Logger
}

func NewHintRoutes(
	router chi.Router,
	hintUC *usecase.HintUseCase,
	userUC *usecase.UserUseCase,
	validator validator.Validator,
	logger logger.Logger,
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
		r.Use(restapiMiddleware.InjectUser(userUC))
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
	challengeUUID, ok := httputil.ParseUUIDParam(w, r, "challengeId")
	if !ok {
		return
	}

	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	hints, err := h.hintUC.GetByChallengeID(r.Context(), challengeUUID, user.TeamId)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetByChallengeID - GetByChallengeID")
		handleError(w, r, err)
		return
	}

	res := response.FromHintWithUnlockList(hints)

	httputil.RenderOK(w, r, res)
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
	hintUUID, ok := httputil.ParseUUIDParam(w, r, "hintId")
	if !ok {
		return
	}

	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	if user.TeamId == nil {
		httputil.RenderError(w, r, http.StatusBadRequest, "user must be in a team")
		return
	}

	hint, err := h.hintUC.UnlockHint(r.Context(), *user.TeamId, hintUUID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - UnlockHint - UnlockHint")
		handleError(w, r, err)
		return
	}

	res := response.FromUnlockedHint(hint)

	httputil.RenderOK(w, r, res)
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
	challengeUUID, ok := httputil.ParseUUIDParam(w, r, "challengeId")
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[request.CreateHintRequest](
		w, r, h.validator, h.logger, "CreateHint",
	)
	if !ok {
		return
	}

	hint, err := h.hintUC.Create(r.Context(), challengeUUID, req.Content, req.Cost, req.OrderIndex)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Create - Create")
		handleError(w, r, err)
		return
	}

	res := response.FromHint(hint)

	httputil.RenderCreated(w, r, res)
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
	hintUUID, ok := httputil.ParseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[request.UpdateHintRequest](
		w, r, h.validator, h.logger, "UpdateHint",
	)
	if !ok {
		return
	}

	hint, err := h.hintUC.Update(r.Context(), hintUUID, req.Content, req.Cost, req.OrderIndex)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Update - Update")
		handleError(w, r, err)
		return
	}

	res := response.FromHint(hint)

	httputil.RenderOK(w, r, res)
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
	hintUUID, ok := httputil.ParseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if err := h.hintUC.Delete(r.Context(), hintUUID); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - Delete - Delete")
		handleError(w, r, err)
		return
	}

	httputil.RenderNoContent(w, r)
}
