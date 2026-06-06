package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /admin/submissions).
func (h *Server) GetAdminSubmissions(w http.ResponseWriter, r *http.Request, params openapi.GetAdminSubmissionsParams) {
	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)
	forceLive := adminForceLive(params.Live)

	result, err := h.comp.SubmissionUC.GetAll(r.Context(), page, perPage, forceLive)
	if h.OnError(w, r, err, "GetAdminSubmissions", "GetAll") {
		return
	}

	httputil.RenderOK(w, r, response.FromSubmissionList(result.Data, result.Total, result.Page, result.PerPage))
}

// (GET /admin/submissions/challenge/{challengeID}).
func (h *Server) GetAdminSubmissionsChallengeChallengeID(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetAdminSubmissionsChallengeChallengeIDParams) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)
	forceLive := adminForceLive(params.Live)

	result, err := h.comp.SubmissionUC.GetByChallenge(r.Context(), challengeIDParsed, page, perPage, forceLive)
	if h.OnError(w, r, err, "GetAdminSubmissionsChallengeChallengeID", "GetByChallenge") {
		return
	}

	httputil.RenderOK(w, r, response.FromSubmissionList(result.Data, result.Total, result.Page, result.PerPage))
}

// (GET /admin/submissions/challenge/{challengeID}/stats).
func (h *Server) GetAdminSubmissionsChallengeChallengeIDStats(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetAdminSubmissionsChallengeChallengeIDStatsParams) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	forceLive := adminForceLive(params.Live)

	stats, err := h.comp.SubmissionUC.GetStats(r.Context(), challengeIDParsed, forceLive)
	if h.OnError(w, r, err, "GetAdminSubmissionsChallengeChallengeIDStats", "GetStats") {
		return
	}

	httputil.RenderOK(w, r, response.FromSubmissionStats(stats))
}

// (GET /admin/submissions/user/{userID}).
func (h *Server) GetAdminSubmissionsUserUserID(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetAdminSubmissionsUserUserIDParams) {
	userIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)
	forceLive := adminForceLive(params.Live)

	result, err := h.comp.SubmissionUC.GetByUser(r.Context(), userIDParsed, page, perPage, forceLive)
	if h.OnError(w, r, err, "GetAdminSubmissionsUserUserID", "GetByUser") {
		return
	}

	httputil.RenderOK(w, r, response.FromSubmissionList(result.Data, result.Total, result.Page, result.PerPage))
}

// (GET /admin/submissions/team/{teamID}).
func (h *Server) GetAdminSubmissionsTeamTeamID(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetAdminSubmissionsTeamTeamIDParams) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)
	forceLive := adminForceLive(params.Live)

	result, err := h.comp.SubmissionUC.GetByTeam(r.Context(), teamIDParsed, page, perPage, forceLive)
	if h.OnError(w, r, err, "GetAdminSubmissionsTeamTeamID", "GetByTeam") {
		return
	}

	httputil.RenderOK(w, r, response.FromSubmissionList(result.Data, result.Total, result.Page, result.PerPage))
}

// (GET /admin/submissions/{ID}).
func (h *Server) GetAdminSubmissionsID(w http.ResponseWriter, r *http.Request, ID string) {
	submissionIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	sub, err := h.comp.SubmissionUC.GetByID(r.Context(), submissionIDParsed)
	if h.OnError(w, r, err, "GetAdminSubmissionsID", "GetByID") {
		return
	}

	httputil.RenderOK(w, r, response.FromSubmission(sub))
}

// (PATCH /admin/submissions/{ID}).
func (h *Server) PatchAdminSubmissionsID(w http.ResponseWriter, r *http.Request, ID string) {
	submissionIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.AdminUpdateSubmissionRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	if req.Discard != nil && *req.Discard {
		sub, err := h.comp.SubmissionUC.Discard(r.Context(), submissionIDParsed)
		if h.OnError(w, r, err, "PatchAdminSubmissionsID", "Discard") {
			return
		}

		httputil.RenderOK(w, r, response.FromSubmission(sub))

		return
	}

	isCorrect, err := request.AdminUpdateSubmissionRequestToParams(&req)
	if h.OnError(w, r, err, "PatchAdminSubmissionsID", "AdminUpdateSubmissionRequestToParams") {
		return
	}

	sub, err := h.comp.SubmissionUC.Update(r.Context(), submissionIDParsed, *isCorrect)
	if h.OnError(w, r, err, "PatchAdminSubmissionsID", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.FromSubmission(sub))
}

// (DELETE /admin/submissions/{ID}).
func (h *Server) DeleteAdminSubmissionsID(w http.ResponseWriter, r *http.Request, ID string) {
	submissionIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.comp.SubmissionUC.Delete(r.Context(), submissionIDParsed), "DeleteAdminSubmissionsID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (POST /admin/submissions).
func (h *Server) PostAdminSubmissions(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.AdminCreateSubmissionRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.AdminCreateSubmissionRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminSubmissions", "ParseParams") {
		return
	}

	sub, err := h.comp.SubmissionUC.AdminCreate(r.Context(), params.UserID, params.TeamID, params.ChallengeID, params.SubmittedFlag, params.IsCorrect, params.IP)
	if h.OnError(w, r, err, "PostAdminSubmissions", "AdminCreate") {
		return
	}

	httputil.RenderCreated(w, r, response.FromSubmission(sub))
}
