package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Get all submissions (admin)
// (GET /admin/submissions)
func (h *Server) GetAdminSubmissions(w http.ResponseWriter, r *http.Request, params openapi.GetAdminSubmissionsParams) {
	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.comp.SubmissionUC.GetAll(r.Context(), page, perPage)
	if h.OnError(w, r, err, "GetAdminSubmissions", "GetAll") {
		return
	}

	helper.RenderOK(w, r, response.FromSubmissionList(result.Data, result.Total, result.Page, result.PerPage))
}

// Get submissions by challenge (admin)
// (GET /admin/submissions/challenge/{challengeID})
func (h *Server) GetAdminSubmissionsChallengeChallengeID(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetAdminSubmissionsChallengeChallengeIDParams) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.comp.SubmissionUC.GetByChallenge(r.Context(), challengeIDParsed, page, perPage)
	if h.OnError(w, r, err, "GetAdminSubmissionsChallengeChallengeID", "GetByChallenge") {
		return
	}

	helper.RenderOK(w, r, response.FromSubmissionList(result.Data, result.Total, result.Page, result.PerPage))
}

// Get submission stats by challenge (admin)
// (GET /admin/submissions/challenge/{challengeID}/stats)
func (h *Server) GetAdminSubmissionsChallengeChallengeIDStats(w http.ResponseWriter, r *http.Request, ID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	stats, err := h.comp.SubmissionUC.GetStats(r.Context(), challengeIDParsed)
	if h.OnError(w, r, err, "GetAdminSubmissionsChallengeChallengeIDStats", "GetStats") {
		return
	}

	helper.RenderOK(w, r, response.FromSubmissionStats(stats))
}

// Get submissions by user (admin)
// (GET /admin/submissions/user/{userID})
func (h *Server) GetAdminSubmissionsUserUserID(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetAdminSubmissionsUserUserIDParams) {
	userIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.comp.SubmissionUC.GetByUser(r.Context(), userIDParsed, page, perPage)
	if h.OnError(w, r, err, "GetAdminSubmissionsUserUserID", "GetByUser") {
		return
	}

	helper.RenderOK(w, r, response.FromSubmissionList(result.Data, result.Total, result.Page, result.PerPage))
}

// Get submissions by team (admin)
// (GET /admin/submissions/team/{teamID})
func (h *Server) GetAdminSubmissionsTeamTeamID(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetAdminSubmissionsTeamTeamIDParams) {
	teamIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.comp.SubmissionUC.GetByTeam(r.Context(), teamIDParsed, page, perPage)
	if h.OnError(w, r, err, "GetAdminSubmissionsTeamTeamID", "GetByTeam") {
		return
	}

	helper.RenderOK(w, r, response.FromSubmissionList(result.Data, result.Total, result.Page, result.PerPage))
}

// Get submission by ID (admin)
// (GET /admin/submissions/{ID})
func (h *Server) GetAdminSubmissionsID(w http.ResponseWriter, r *http.Request, ID string) {
	submissionIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	sub, err := h.comp.SubmissionUC.GetByID(r.Context(), submissionIDParsed)
	if h.OnError(w, r, err, "GetAdminSubmissionsID", "GetByID") {
		return
	}
	helper.RenderOK(w, r, response.FromSubmission(sub))
}

// Update submission (admin)
// (PATCH /admin/submissions/{ID})
func (h *Server) PatchAdminSubmissionsID(w http.ResponseWriter, r *http.Request, ID string) {
	submissionIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.AdminUpdateSubmissionRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PatchAdminSubmissionsID",
	)
	if !ok {
		return
	}
	isCorrect := request.AdminUpdateSubmissionRequestToParams(&req)
	sub, err := h.comp.SubmissionUC.Update(r.Context(), submissionIDParsed, isCorrect)
	if h.OnError(w, r, err, "PatchAdminSubmissionsID", "Update") {
		return
	}
	helper.RenderOK(w, r, response.FromSubmission(sub))
}

// Delete submission (admin)
// (DELETE /admin/submissions/{ID})
func (h *Server) DeleteAdminSubmissionsID(w http.ResponseWriter, r *http.Request, ID string) {
	submissionIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	if h.OnError(w, r, h.comp.SubmissionUC.Delete(r.Context(), submissionIDParsed), "DeleteAdminSubmissionsID", "Delete") {
		return
	}
	helper.RenderNoContent(w, r)
}

// Create submission (admin)
// (POST /admin/submissions)
func (h *Server) PostAdminSubmissions(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.AdminCreateSubmissionRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminSubmissions",
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
	helper.RenderCreated(w, r, response.FromSubmission(sub))
}
