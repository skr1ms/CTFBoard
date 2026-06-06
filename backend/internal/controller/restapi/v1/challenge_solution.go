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

	if !h.checkWriteupEnabled(w, r, "GetChallengesSolutions", "WriteupCheck") {
		return
	}

	if user.TeamID == nil {
		httputil.RenderOK(w, r, response.EmptyChallengeSolutionEntryList())

		return
	}

	if _, ok := helper.RequireUnbannedTeam(w, r, h.team.TeamUC, *user.TeamID, h.OnError, "GetChallengesSolutions"); !ok {
		return
	}

	entries, err := h.challenge.ChallengeUC.ListSolutions(r.Context(), *user.TeamID)
	if h.OnError(w, r, err, "GetChallengesSolutions", "ListSolutions") {
		return
	}

	urls, err := h.challenge.FileUC.BuildDownloadURLs(r.Context(), response.ChallengeSolutionEntryFiles(entries), user.TeamID, helper.IsAdmin(user))
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

	if !h.checkWriteupEnabled(w, r, "GetChallengesChallengeIDSolution", "WriteupCheck") {
		return
	}

	if !helper.CheckOptionalTeamBan(w, r, h.team.TeamUC, user.TeamID, h.OnError, "GetChallengesChallengeIDSolution") {
		return
	}

	solution, err := h.challenge.ChallengeUC.GetSolution(r.Context(), challengeIDParsed, user.TeamID)
	if h.OnError(w, r, err, "GetChallengesChallengeIDSolution", "GetSolution") {
		return
	}

	urls, err := h.challenge.FileUC.BuildDownloadURLs(r.Context(), solution.Files, user.TeamID, helper.IsAdmin(user))
	if h.OnError(w, r, err, "GetChallengesChallengeIDSolution", "BuildDownloadURLs") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallengeSolution(solution, urls))
}
