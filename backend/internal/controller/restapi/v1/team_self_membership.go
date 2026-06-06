package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (POST /teams).
func (h *Server) PostTeams(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.CreateTeamRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	name, confirmReset := request.CreateTeamRequestToParams(&req)
	if confirmReset {
		team, err := h.team.TeamUC.ConfirmCreate(r.Context(), name, user.ID, false)
		if h.OnError(w, r, err, "PostTeams", "ConfirmCreate") {
			return
		}

		httputil.RenderCreated(w, r, response.FromTeam(team))

		return
	}

	result, err := h.team.TeamUC.TryCreate(r.Context(), name, user.ID, false)
	if h.OnError(w, r, err, "PostTeams", "TryCreate") {
		return
	}

	if result.RequiresConfirm {
		httputil.RenderOK(w, r, response.FromConfirmationRequired(string(result.ConfirmationReason), response.FromAffectedData(result.AffectedData)))

		return
	}

	httputil.RenderCreated(w, r, response.FromTeam(result.Team))
}

// (POST /teams/join).
func (h *Server) PostTeamsJoin(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.JoinTeamRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	inviteToken, confirmReset := request.JoinTeamRequestToParams(&req)

	inviteTokenID, ok := httputil.ParseUUID(w, r, inviteToken)
	if !ok {
		return
	}

	team, err := h.team.TeamUC.Join(r.Context(), inviteTokenID, user.ID, confirmReset)
	if h.OnError(w, r, err, "PostTeamsJoin", "Join") {
		return
	}

	httputil.RenderOK(w, r, response.FromTeam(team))
}

// (POST /teams/leave).
func (h *Server) PostTeamsLeave(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if h.OnError(w, r, h.team.TeamUC.Leave(r.Context(), user.ID), "PostTeamsLeave", "Leave") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (DELETE /teams/me).
func (h *Server) DeleteTeamsMe(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if h.OnError(w, r, h.team.TeamUC.DisbandTeam(r.Context(), user.ID), "DeleteTeamsMe", "DisbandTeam") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (POST /teams/solo).
func (h *Server) PostTeamsSolo(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.CreateSoloTeamRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	confirmReset := request.CreateSoloTeamRequestToParams(&req)

	team, err := h.team.TeamUC.CreateSoloTeam(r.Context(), user.ID, confirmReset)
	if h.OnError(w, r, err, "PostTeamsSolo", "CreateSoloTeam") {
		return
	}

	httputil.RenderCreated(w, r, response.FromTeam(team))
}
