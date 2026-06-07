package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
)

// (GET /challenges/solutions).
func (h *Server) GetChallengesSolutions(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.CheckOptionalTeamBan(w, r, h.team.ReadUC, user.TeamID, h.OnError, "GetChallengesSolutions") {
		return
	}

	isAdmin := helper.IsAdmin(user)

	entries, err := h.challenge.ReadUC.ListSolutions(r.Context(), user.TeamID, isAdmin)
	if h.OnError(w, r, err, "GetChallengesSolutions", "ListSolutions") {
		return
	}

	urls, err := h.challenge.FileUC.BuildDownloadURLs(r.Context(), response.ChallengeSolutionEntryFiles(entries), user.TeamID, isAdmin)
	if h.OnError(w, r, err, "GetChallengesSolutions", "BuildDownloadURLs") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallengeSolutionEntryList(entries, urls))
}

// (GET /challenges/{challengeID}/solution).
func (h *Server) GetChallengesChallengeIDSolution(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.CheckOptionalTeamBan(w, r, h.team.ReadUC, user.TeamID, h.OnError, "GetChallengesChallengeIDSolution") {
		return
	}

	isAdmin := helper.IsAdmin(user)

	solution, err := h.challenge.ReadUC.GetSolution(r.Context(), challengeIDParsed, user.TeamID, isAdmin)
	if h.OnError(w, r, err, "GetChallengesChallengeIDSolution", "GetSolution") {
		return
	}

	urls, err := h.challenge.FileUC.BuildDownloadURLs(r.Context(), solution.Files, user.TeamID, isAdmin)
	if h.OnError(w, r, err, "GetChallengesChallengeIDSolution", "BuildDownloadURLs") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallengeSolution(solution, urls))
}
