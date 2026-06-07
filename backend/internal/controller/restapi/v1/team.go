package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func (h *Server) requirePublicTeamStatsVisible(w http.ResponseWriter, r *http.Request, teamID uuid.UUID, op string) bool {
	team, err := h.team.ReadUC.GetByID(r.Context(), teamID)
	if h.OnError(w, r, err, op, "GetByID") {
		return false
	}

	viewer, _ := helper.CurrentUser(r)
	if !helper.TeamStatsVisibleToViewer(team, viewer) {
		h.OnError(w, r, helper.ErrTeamNotFound, op, "HiddenTeam")

		return false
	}

	return true
}

// (GET /teams/{ID}).
func (h *Server) GetTeamsID(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	team, err := h.team.ReadUC.GetByID(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetTeamsID", "GetByID") {
		return
	}

	if (team.IsBanned || team.IsHidden) && !helper.IsAdmin(user) {
		h.OnError(w, r, helper.ErrTeamNotFound, "GetTeamsID", "team banned or hidden")

		return
	}

	profile, err := h.team.ReadUC.GetProfile(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetTeamsID", "GetProfile") {
		return
	}

	httputil.RenderOK(w, r, response.FromPublicTeamProfile(profile))
}

// (GET /teams).
func (h *Server) GetTeams(w http.ResponseWriter, r *http.Request, params openapi.GetTeamsParams) {
	search, ok := helper.ParseOptionalSearchQuery(w, r, params.Q, maxSearchQueryLen, h.OnError, "GetTeams", "Q")
	if !ok {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.team.ReadUC.ListTeams(r.Context(), search, page, perPage)
	if h.OnError(w, r, err, "GetTeams", "ListTeams") {
		return
	}

	httputil.RenderOK(w, r, response.FromTeamList(result.Data, result.Total, result.Page, result.PerPage))
}

// (GET /teams/{ID}/solves).
func (h *Server) GetTeamsIDSolves(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if !h.requirePublicTeamStatsVisible(w, r, teamIDParsed, "GetTeamsIDSolves") {
		return
	}

	solves, err := h.team.ReadUC.GetTeamSolves(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetTeamsIDSolves", "GetTeamSolves") {
		return
	}

	httputil.RenderOK(w, r, response.FromSolveWithDetailsList(solves))
}

// (GET /teams/{ID}/fails).
func (h *Server) GetTeamsIDFails(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetTeamsIDFailsParams) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if !h.requirePublicTeamStatsVisible(w, r, teamIDParsed, "GetTeamsIDFails") {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	fails, err := h.team.ReadUC.GetTeamFails(r.Context(), teamIDParsed, page, perPage)
	if h.OnError(w, r, err, "GetTeamsIDFails", "GetTeamFails") {
		return
	}

	httputil.RenderOK(w, r, response.FromFailListPublic(fails.Data, fails.Total, fails.Page, fails.PerPage))
}

// (GET /teams/{ID}/awards).
func (h *Server) GetTeamsIDAwards(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if !h.requirePublicTeamStatsVisible(w, r, teamIDParsed, "GetTeamsIDAwards") {
		return
	}

	awards, err := h.team.ReadUC.GetTeamAwards(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetTeamsIDAwards", "GetTeamAwards") {
		return
	}

	httputil.RenderOK(w, r, response.FromAwardList(awards))
}
