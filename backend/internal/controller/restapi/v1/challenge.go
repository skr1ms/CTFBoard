package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get challenges list
// (GET /challenges)
func (h *Server) GetChallenges(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUser(r.Context())
	if !ok {
		RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	challenges, err := h.challengeUC.GetAll(r.Context(), user.TeamID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetChallenges")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromChallengeList(challenges))
}

// Submit flag
// (POST /challenges/{ID}/submit)
func (h *Server) PostChallengesIDSubmit(w http.ResponseWriter, r *http.Request, ID string) {
	challengeuuid, err := uuid.Parse(ID)
	if err != nil {
		RenderInvalidID(w, r)
		return
	}

	req, ok := DecodeAndValidate[openapi.RequestSubmitFlagRequest](
		w, r, h.validator, h.logger, "PostChallengesIDSubmit",
	)
	if !ok {
		return
	}

	user, ok := middleware.GetUser(r.Context())
	if !ok {
		RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	flag := request.SubmitFlagRequestToFlag(&req)
	valid, err := h.challengeUC.SubmitFlag(r.Context(), challengeuuid, flag, user.ID, user.TeamID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostChallengesIDSubmit")
		handleError(w, r, err)
		return
	}

	if !valid {
		RenderError(w, r, http.StatusBadRequest, "invalid flag")
		return
	}

	RenderOK(w, r, map[string]string{"message": "flag accepted"})
}

// Create challenge
// (POST /admin/challenges)
func (h *Server) PostAdminChallenges(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestCreateChallengeRequest](
		w, r, h.validator, h.logger, "PostAdminChallenges",
	)
	if !ok {
		return
	}

	title, desc, cat, pts, initVal, minVal, decay, flag, isHidden, isRegex, isCaseInsens, flagRegex := request.CreateChallengeRequestToParams(&req)
	challenge, err := h.challengeUC.Create(
		r.Context(),
		title, desc, cat, pts, initVal, minVal, decay, flag,
		isHidden, isRegex, isCaseInsens, flagRegex,
	)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAdminChallenges")
		handleError(w, r, err)
		return
	}

	RenderCreated(w, r, response.FromChallenge(challenge))
}

// Delete challenge
// (DELETE /admin/challenges/{ID})
func (h *Server) DeleteAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeuuid, err := uuid.Parse(ID)
	if err != nil {
		RenderInvalidID(w, r)
		return
	}

	user, ok := middleware.GetUser(r.Context())
	if !ok {
		RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	clientIP := GetClientIP(r)

	err = h.challengeUC.Delete(r.Context(), challengeuuid, user.ID, clientIP)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - DeleteAdminChallengesID")
		handleError(w, r, err)
		return
	}

	RenderNoContent(w, r)
}

// Update challenge
// (PUT /admin/challenges/{ID})
func (h *Server) PutAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeuuid, err := uuid.Parse(ID)
	if err != nil {
		RenderInvalidID(w, r)
		return
	}

	req, ok := DecodeAndValidate[openapi.RequestUpdateChallengeRequest](
		w, r, h.validator, h.logger, "PutAdminChallengesID",
	)
	if !ok {
		return
	}

	title, desc, cat, pts, initVal, minVal, decay, flag, isHidden, isRegex, isCaseInsens, flagRegex := request.UpdateChallengeRequestToParams(&req)
	challenge, err := h.challengeUC.Update(
		r.Context(),
		challengeuuid,
		title, desc, cat, pts, initVal, minVal, decay, flag,
		isHidden, isRegex, isCaseInsens, flagRegex,
	)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PutAdminChallengesID")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromChallenge(challenge))
}
