package v1

import (
	"net/http"

	"github.com/google/uuid"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

// Get challenges list
// (GET /challenges)
func (h *Server) GetChallenges(w http.ResponseWriter, r *http.Request) {
	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	challenges, err := h.challengeUC.GetAll(r.Context(), user.TeamID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetChallenges")
		handleError(w, r, err)
		return
	}

	res := response.FromChallengeList(challenges)

	httputil.RenderOK(w, r, res)
}

// Submit flag
// (POST /challenges/{ID}/submit)
func (h *Server) PostChallengesIDSubmit(w http.ResponseWriter, r *http.Request, ID string) {
	challengeuuid, err := uuid.Parse(ID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	req, ok := httputil.DecodeAndValidate[request.SubmitFlagRequest](
		w, r, h.validator, h.logger, "PostChallengesIDSubmit",
	)
	if !ok {
		return
	}

	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	valid, err := h.challengeUC.SubmitFlag(r.Context(), challengeuuid, req.Flag, user.ID, user.TeamID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostChallengesIDSubmit")
		handleError(w, r, err)
		return
	}

	if !valid {
		httputil.RenderError(w, r, http.StatusBadRequest, "invalid flag")
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "flag accepted"})
}

// Create challenge
// (POST /admin/challenges)
func (h *Server) PostAdminChallenges(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.CreateChallengeRequest](
		w, r, h.validator, h.logger, "PostAdminChallenges",
	)
	if !ok {
		return
	}

	challenge, err := h.challengeUC.Create(
		r.Context(),
		req.Title,
		req.Description,
		req.Category,
		req.Points,
		req.InitialValue,
		req.MinValue,
		req.Decay,
		req.Flag,
		req.IsHidden,
		req.IsRegex,
		req.IsCaseInsensitive,
	)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAdminChallenges")
		handleError(w, r, err)
		return
	}

	res := response.FromChallenge(challenge)

	httputil.RenderCreated(w, r, res)
}

// Delete challenge
// (DELETE /admin/challenges/{ID})
func (h *Server) DeleteAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeuuid, err := uuid.Parse(ID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	clientIP := httputil.GetClientIP(r)

	err = h.challengeUC.Delete(r.Context(), challengeuuid, user.ID, clientIP)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - DeleteAdminChallengesID")
		handleError(w, r, err)
		return
	}

	httputil.RenderNoContent(w, r)
}

// Update challenge
// (PUT /admin/challenges/{ID})
func (h *Server) PutAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeuuid, err := uuid.Parse(ID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	req, ok := httputil.DecodeAndValidate[request.UpdateChallengeRequest](
		w, r, h.validator, h.logger, "PutAdminChallengesID",
	)
	if !ok {
		return
	}

	challenge, err := h.challengeUC.Update(
		r.Context(),
		challengeuuid,
		req.Title,
		req.Description,
		req.Category,
		req.Points,
		req.InitialValue,
		req.MinValue,
		req.Decay,
		req.Flag,
		req.IsHidden,
		req.IsRegex,
		req.IsCaseInsensitive,
	)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PutAdminChallengesID")
		handleError(w, r, err)
		return
	}

	res := response.FromChallenge(challenge)

	httputil.RenderOK(w, r, res)
}
