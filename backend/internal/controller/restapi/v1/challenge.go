package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /challenges).
func (h *Server) GetChallenges(w http.ResponseWriter, r *http.Request, params openapi.GetChallengesParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	var tagID *uuid.UUID

	if params.Tag != nil && *params.Tag != "" {
		id, ok := httputil.ParseUUIDField(w, r, *params.Tag, "tag")
		if !ok {
			return
		}

		tagID = &id
	}

	// Non-admin users see an empty list before the competition starts so the
	// frontend can render a countdown banner instead of a 403 error.
	if !helper.IsAdmin(user) {
		comp, err := h.comp.CompetitionUC.Get(r.Context())
		if h.OnError(w, r, err, "GetChallenges", "GetCompetition") {
			return
		}

		if helper.IsCompetitionNotStarted(comp) {
			httputil.RenderOK(w, r, response.FromChallengeList(nil))

			return
		}
	}

	challenges, err := h.challenge.ChallengeUC.GetAll(r.Context(), user.TeamID, tagID)
	if h.OnError(w, r, err, "GetChallenges", "GetAll") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallengeList(challenges))
}

// (POST /challenges/{challengeID}/submit).
func (h *Server) PostChallengesChallengeIDSubmit(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.SubmitFlagRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	flag, err := request.SubmitFlagRequestToParams(&req)
	if h.OnError(w, r, err, "PostChallengesChallengeIDSubmit", "RequestConversion") {
		return
	}

	clientIP := helper.ClientIP(r)
	valid, submitErr := h.challenge.ChallengeUC.SubmitFlag(r.Context(), challengeIDParsed, flag, user.ID, user.TeamID, clientIP)

	if h.OnError(w, r, submitErr, "PostChallengesChallengeIDSubmit", "SubmitFlag") {
		return
	}

	if !valid {
		httputil.RenderOK(w, r, response.FromSubmitFlag(false, "incorrect flag"))

		return
	}

	httputil.RenderOK(w, r, response.FromSubmitFlag(true, "flag accepted"))
}

// (GET /challenges/{challengeID}).
func (h *Server) GetChallengesChallengeID(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	var teamID *uuid.UUID

	if user.TeamID != nil {
		teamID = user.TeamID
	}

	detail, err := h.challenge.ChallengeUC.GetDetail(r.Context(), challengeIDParsed, teamID)
	if h.OnError(w, r, err, "GetChallengesChallengeID", "GetDetail") {
		return
	}

	helper.TrackChallengeOpenAsync(r.Context(), h.infra.Logger, h.user.TrackingUC, user.ID, challengeIDParsed, helper.ClientIP(r))

	httputil.RenderOK(w, r, response.FromChallengeDetail(detail))
}

// (GET /challenges/{challengeID}/solves).
func (h *Server) GetChallengesChallengeIDSolves(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	solves, err := h.challenge.ChallengeUC.GetSolves(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDSolves", "GetSolves") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallengeSolves(solves))
}

// (GET /challenges/{challengeID}/tags).
func (h *Server) GetChallengesChallengeIDTags(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	tags, err := h.challenge.TagUC.GetByChallengeID(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDTags", "GetByChallengeID") {
		return
	}

	httputil.RenderOK(w, r, response.FromTagList(tags))
}

// (GET /challenges/types).
func (h *Server) GetChallengesTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.challenge.ChallengeUC.GetTypes(r.Context())
	if h.OnError(w, r, err, "GetChallengesTypes", "GetTypes") {
		return
	}

	setPublicCache(w, cacheStatic, false)
	httputil.RenderOK(w, r, response.FromChallengeTypes(types))
}

// (GET /challenges/{challengeID}/requirements).
func (h *Server) GetChallengesChallengeIDRequirements(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	requirements, err := h.challenge.ChallengeUC.GetRequirements(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetChallengesChallengeIDRequirements", "GetRequirements") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallengeRequirements(requirements))
}
