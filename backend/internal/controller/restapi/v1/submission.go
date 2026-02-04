package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get all submissions (admin)
// (GET /admin/submissions)
func (h *Server) GetAdminSubmissions(w http.ResponseWriter, r *http.Request, params openapi.GetAdminSubmissionsParams) {
	page, perPage := getPagePerPage(params.Page, params.PerPage)

	items, total, err := h.submissionUC.GetAll(r.Context(), page, perPage)
	if h.OnError(w, r, err, "GetAdminSubmissions", "GetAll") {
		return
	}

	RenderOK(w, r, response.FromSubmissionList(items, total, page, perPage))
}

// Get submissions by challenge (admin)
// (GET /admin/submissions/challenge/{challengeID})
func (h *Server) GetAdminSubmissionsChallengeChallengeID(w http.ResponseWriter, r *http.Request, challengeID string, params openapi.GetAdminSubmissionsChallengeChallengeIDParams) {
	cid, ok := ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	page, perPage := getPagePerPage(params.Page, params.PerPage)

	items, total, err := h.submissionUC.GetByChallenge(r.Context(), cid, page, perPage)
	if h.OnError(w, r, err, "GetAdminSubmissionsChallengeChallengeID", "GetByChallenge") {
		return
	}

	RenderOK(w, r, response.FromSubmissionList(items, total, page, perPage))
}

// Get submission stats by challenge (admin)
// (GET /admin/submissions/challenge/{challengeID}/stats)
func (h *Server) GetAdminSubmissionsChallengeChallengeIDStats(w http.ResponseWriter, r *http.Request, challengeID string) {
	cid, ok := ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	stats, err := h.submissionUC.GetStats(r.Context(), cid)
	if h.OnError(w, r, err, "GetAdminSubmissionsChallengeChallengeIDStats", "GetStats") {
		return
	}

	RenderOK(w, r, response.FromSubmissionStats(stats))
}

// Get submissions by user (admin)
// (GET /admin/submissions/user/{userID})
func (h *Server) GetAdminSubmissionsUserUserID(w http.ResponseWriter, r *http.Request, userID string, params openapi.GetAdminSubmissionsUserUserIDParams) {
	uid, ok := ParseUUID(w, r, userID)
	if !ok {
		return
	}

	page, perPage := getPagePerPage(params.Page, params.PerPage)

	items, total, err := h.submissionUC.GetByUser(r.Context(), uid, page, perPage)
	if h.OnError(w, r, err, "GetAdminSubmissionsUserUserID", "GetByUser") {
		return
	}

	RenderOK(w, r, response.FromSubmissionList(items, total, page, perPage))
}

// Get submissions by team (admin)
// (GET /admin/submissions/team/{teamID})
func (h *Server) GetAdminSubmissionsTeamTeamID(w http.ResponseWriter, r *http.Request, teamID string, params openapi.GetAdminSubmissionsTeamTeamIDParams) {
	tid, ok := ParseUUID(w, r, teamID)
	if !ok {
		return
	}

	page, perPage := getPagePerPage(params.Page, params.PerPage)

	items, total, err := h.submissionUC.GetByTeam(r.Context(), tid, page, perPage)
	if h.OnError(w, r, err, "GetAdminSubmissionsTeamTeamID", "GetByTeam") {
		return
	}

	RenderOK(w, r, response.FromSubmissionList(items, total, page, perPage))
}

func getPagePerPage(page, perPage *int) (int, int) {
	p := 1
	if page != nil && *page > 0 {
		p = *page
	}
	pp := 20
	if perPage != nil && *perPage > 0 && *perPage <= 100 {
		pp = *perPage
	}
	return p, pp
}
