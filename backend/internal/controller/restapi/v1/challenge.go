package v1

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	restapiMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
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
	logger        logger.Logger
}

func NewChallengeRoutes(router chi.Router,
	challengeUC *usecase.ChallengeUseCase,
	solveUC *usecase.SolveUseCase,
	userUC *usecase.UserUseCase,
	competitionUC *usecase.CompetitionUseCase,
	validator validator.Validator,
	logger logger.Logger,
	jwtService *jwt.JWTService,
	redisClient *redis.Client,
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

	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.InjectUser(userUC)).Get("/challenges", routes.GetAll)

	ipLimit := restapiMiddleware.RateLimit(redisClient, "submit:ip", int64(submitLimit*3), durationLimit, func(r *http.Request) (string, error) {
		return httputil.GetClientIP(r), nil
	})

	userLimit := restapiMiddleware.RateLimit(redisClient, "submit:user", int64(submitLimit), durationLimit, func(r *http.Request) (string, error) {
		user, ok := restapiMiddleware.GetUser(r.Context())
		if !ok {
			return "", http.ErrNoCookie
		}
		return user.Id.String(), nil
	})

	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.InjectUser(userUC), restapiMiddleware.CompetitionActive(competitionUC), ipLimit, userLimit).Post("/challenges/{id}/submit", routes.SubmitFlag)
	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.Admin).Post("/admin/challenges", routes.Create)
	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.Admin).Put("/admin/challenges/{id}", routes.Update)
	router.With(restapiMiddleware.Auth(jwtService), restapiMiddleware.Admin, restapiMiddleware.InjectUser(userUC)).Delete("/admin/challenges/{id}", routes.Delete)
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
	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	challenges, err := h.challengeUC.GetAll(r.Context(), user.TeamId)
	if err != nil {
		h.logger.WithError(err).Error("http - v1 - GetAll - GetAll")
		handleError(w, r, err)
		return
	}

	res := response.FromChallengeList(challenges)

	httputil.RenderOK(w, r, res)
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
	challengeUUID, ok := httputil.ParseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[request.SubmitFlagRequest](
		w, r, h.validator, h.logger, "SubmitFlag",
	)
	if !ok {
		return
	}

	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	valid, err := h.challengeUC.SubmitFlag(r.Context(), challengeUUID, req.Flag, user.Id, user.TeamId)
	if err != nil {
		h.logger.WithError(err).Error("http - v1 - SubmitFlag")
		handleError(w, r, err)
		return
	}

	if !valid {
		httputil.RenderError(w, r, http.StatusBadRequest, "invalid flag")
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "flag accepted"})
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
	req, ok := httputil.DecodeAndValidate[request.CreateChallengeRequest](
		w, r, h.validator, h.logger, "Create",
	)
	if !ok {
		return
	}

	challenge, err := h.challengeUC.Create(r.Context(), req.Title, req.Description, req.Category, req.Points, req.InitialValue, req.MinValue, req.Decay, req.Flag, req.IsHidden, req.IsRegex, req.IsCaseInsensitive)
	if err != nil {
		h.logger.WithError(err).Error("http - v1 - Create - Create")
		handleError(w, r, err)
		return
	}

	res := response.FromChallenge(challenge)

	httputil.RenderCreated(w, r, res)
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
	challengeUUID, ok := httputil.ParseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	user, ok := restapiMiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	clientIP := httputil.GetClientIP(r)

	err := h.challengeUC.Delete(r.Context(), challengeUUID, user.Id, clientIP)
	if err != nil {
		h.logger.WithError(err).Error("http - v1 - Delete")
		handleError(w, r, err)
		return
	}

	httputil.RenderNoContent(w, r)
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
	challengeUUID, ok := httputil.ParseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[request.UpdateChallengeRequest](
		w, r, h.validator, h.logger, "Update",
	)
	if !ok {
		return
	}

	challenge, err := h.challengeUC.Update(r.Context(), challengeUUID, req.Title, req.Description, req.Category, req.Points, req.InitialValue, req.MinValue, req.Decay, req.Flag, req.IsHidden, req.IsRegex, req.IsCaseInsensitive)
	if err != nil {
		h.logger.WithError(err).Error("http - v1 - Update - Update")
		handleError(w, r, err)
		return
	}

	res := response.FromChallenge(challenge)

	httputil.RenderOK(w, r, res)
}
