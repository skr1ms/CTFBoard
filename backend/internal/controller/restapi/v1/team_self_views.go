package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /teams/my).
func (h *Server) GetTeamsMy(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	team, members, minTeamSize, meetsMinSize, err := h.team.TeamUC.GetMyTeam(r.Context(), user.ID)
	if h.OnError(w, r, err, "GetTeamsMy", "GetMyTeam") {
		return
	}

	resp := response.FromTeamWithMembers(team, members, minTeamSize, meetsMinSize)
	if team.IsBanned {
		resp.InviteToken = nil
	}

	httputil.RenderOK(w, r, resp)
}

// (GET /teams/me/solves).
func (h *Server) GetTeamsMeSolves(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	teamID, ok := helper.RequireTeamID(w, r, user, h.OnError, "GetTeamsMeSolves")
	if !ok {
		return
	}

	solves, err := h.team.TeamUC.GetTeamSolves(r.Context(), teamID)
	if h.OnError(w, r, err, "GetTeamsMeSolves", "GetTeamSolves") {
		return
	}

	httputil.RenderOK(w, r, response.FromSolveWithDetailsList(solves))
}

// (GET /teams/me/fails).
func (h *Server) GetTeamsMeFails(w http.ResponseWriter, r *http.Request, params openapi.GetTeamsMeFailsParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	teamID, ok := helper.RequireTeamID(w, r, user, h.OnError, "GetTeamsMeFails")
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	fails, err := h.team.TeamUC.GetTeamFails(r.Context(), teamID, page, perPage)
	if h.OnError(w, r, err, "GetTeamsMeFails", "GetTeamFails") {
		return
	}

	httputil.RenderOK(w, r, response.FromFailListSelf(fails.Data, fails.Total, fails.Page, fails.PerPage))
}

// (GET /teams/me/awards).
func (h *Server) GetTeamsMeAwards(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	teamID, ok := helper.RequireTeamID(w, r, user, h.OnError, "GetTeamsMeAwards")
	if !ok {
		return
	}

	awards, err := h.team.TeamUC.GetTeamAwards(r.Context(), teamID)
	if h.OnError(w, r, err, "GetTeamsMeAwards", "GetTeamAwards") {
		return
	}

	httputil.RenderOK(w, r, response.FromAwardList(awards))
}
