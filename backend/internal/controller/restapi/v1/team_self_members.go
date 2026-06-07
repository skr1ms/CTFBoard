package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (DELETE /teams/members/{ID}).
func (h *Server) DeleteTeamsMembersID(w http.ResponseWriter, r *http.Request, ID string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	memberIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.team.SelfUC.KickMember(r.Context(), user.ID, memberIDParsed), "DeleteTeamsMembersID", "KickMember") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (POST /teams/transfer-captain).
func (h *Server) PostTeamsTransferCaptain(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.TransferCaptainRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	newCaptainID, err := request.TransferCaptainRequestToParams(&req)
	if h.OnError(w, r, err, "PostTeamsTransferCaptain", "RequestConversion") {
		return
	}

	if h.OnError(w, r, h.team.SelfUC.TransferCaptain(r.Context(), user.ID, newCaptainID), "PostTeamsTransferCaptain", "TransferCaptain") {
		return
	}

	httputil.RenderOK(w, r, response.Message("captainship transferred"))
}

// (PATCH /teams/me).
func (h *Server) PatchTeamsMe(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateTeamRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.UpdateTeamRequestToParams(&req)
	if h.OnError(w, r, err, "PatchTeamsMe", "RequestConversion") {
		return
	}

	team, err := h.team.SelfUC.UpdateMyTeam(r.Context(), user.ID, params)
	if h.OnError(w, r, err, "PatchTeamsMe", "UpdateMyTeam") {
		return
	}

	httputil.RenderOK(w, r, response.FromTeamProfile(team))
}

// (GET /teams/me/invite).
func (h *Server) GetTeamsMeInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	team, err := h.team.SelfUC.GetInviteToken(r.Context(), user.ID)
	if h.OnError(w, r, err, "GetTeamsMeInvite", "GetInviteToken") {
		return
	}

	httputil.RenderOK(w, r, response.FromTeamInvite(team.InviteToken.String()))
}

// (POST /teams/me/invite).
func (h *Server) PostTeamsMeInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	team, err := h.team.SelfUC.RegenerateInviteToken(r.Context(), user.ID)
	if h.OnError(w, r, err, "PostTeamsMeInvite", "RegenerateInviteToken") {
		return
	}

	httputil.RenderOK(w, r, response.FromTeamInvite(team.InviteToken.String()))
}
