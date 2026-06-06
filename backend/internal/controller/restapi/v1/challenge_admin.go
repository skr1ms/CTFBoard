package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (POST /admin/challenges/recalc-points).
func (h *Server) PostAdminChallengesRecalcPoints(w http.ResponseWriter, r *http.Request) {
	if err := h.challenge.ChallengeUC.RecalcAllDynamicPoints(r.Context()); h.OnError(w, r, err, "PostAdminChallengesRecalcPoints", "RecalcAllDynamicPoints") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (POST /admin/challenges).
func (h *Server) PostAdminChallenges(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.CreateChallengeRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.CreateChallengeRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminChallenges", "RequestConversion") {
		return
	}

	challenge, err := h.challenge.ChallengeUC.Create(r.Context(), params)
	if h.OnError(w, r, err, "PostAdminChallenges", "Create") {
		return
	}

	httputil.RenderCreated(w, r, response.FromChallenge(challenge))
}

// (DELETE /admin/challenges/{ID}).
func (h *Server) DeleteAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	clientIP := helper.ClientIP(r)

	err := h.challenge.ChallengeUC.Delete(r.Context(), challengeIDParsed, user.ID, clientIP)
	if h.OnError(w, r, err, "DeleteAdminChallengesID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (PUT /admin/challenges/{ID}).
func (h *Server) PutAdminChallengesID(w http.ResponseWriter, r *http.Request, ID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateChallengeRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.UpdateChallengeRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminChallengesID", "RequestConversion") {
		return
	}

	challenge, err := h.challenge.ChallengeUC.Update(r.Context(), challengeIDParsed, params)
	if h.OnError(w, r, err, "PutAdminChallengesID", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallenge(challenge))
}

// (POST /admin/challenges/{challengeID}/solution).
func (h *Server) PostAdminChallengesChallengeIDSolution(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.AdminUpsertSolutionRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	solution, err := h.challenge.ChallengeUC.AdminUpsertSolution(r.Context(), challengeIDParsed, request.AdminUpsertSolutionRequestToParams(&req))
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDSolution", "AdminUpsertSolution") {
		return
	}

	urls, err := h.challenge.FileUC.BuildDownloadURLs(r.Context(), solution.Files, nil, true)
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDSolution", "BuildDownloadURLs") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallengeSolution(solution, urls))
}

// (DELETE /admin/challenges/{challengeID}/solution).
func (h *Server) DeleteAdminChallengesChallengeIDSolution(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	err := h.challenge.ChallengeUC.AdminDeleteSolution(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "DeleteAdminChallengesChallengeIDSolution", "AdminDeleteSolution") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (GET /admin/challenges/{challengeID}/flags).
func (h *Server) GetAdminChallengesChallengeIDFlags(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	flags, err := h.challenge.ChallengeUC.GetFlags(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetAdminChallengesChallengeIDFlags", "GetFlags") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallengeFlags(flags))
}

// (PUT /admin/challenges/{challengeID}/requirements).
func (h *Server) PutAdminChallengesChallengeIDRequirements(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.SetChallengeRequirementsRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	requirementIDs, err := request.ParseUUIDSlice(req.RequirementIds, "requirement_id")
	if h.OnError(w, r, err, "PutAdminChallengesChallengeIDRequirements", "ParseRequirementIDs") {
		return
	}

	err = h.challenge.ChallengeUC.SetRequirements(r.Context(), challengeIDParsed, requirementIDs)
	if h.OnError(w, r, err, "PutAdminChallengesChallengeIDRequirements", "SetRequirements") {
		return
	}

	httputil.RenderNoContent(w, r)
}
