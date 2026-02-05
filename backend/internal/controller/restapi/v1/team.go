package v1

import (
	"errors"
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Create team
// (POST /teams)
func (h *Server) PostTeams(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RequestCreateTeamRequest](
		w, r, h.validator, h.logger, "PostTeams",
	)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	name, confirmReset := request.CreateTeamRequestToParams(&req)
	team, err := h.teamUC.Create(r.Context(), name, user.ID, false, confirmReset)
	if h.OnError(w, r, err, "PostTeams", "Create") {
		return
	}

	helper.RenderCreated(w, r, response.FromTeam(team))
}

// Join team
// (POST /teams/join)
func (h *Server) PostTeamsJoin(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RequestJoinTeamRequest](
		w, r, h.validator, h.logger, "PostTeamsJoin",
	)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	inviteToken, confirmReset := request.JoinTeamRequestToParams(&req)
	inviteTokenuuid, ok := helper.ParseUUID(w, r, inviteToken)
	if !ok {
		return
	}

	team, err := h.teamUC.Join(r.Context(), inviteTokenuuid, user.ID, confirmReset)
	if h.OnError(w, r, err, "PostTeamsJoin", "Join") {
		return
	}

	helper.RenderOK(w, r, response.FromTeam(team))
}

// Leave team
// (POST /teams/leave)
func (h *Server) PostTeamsLeave(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if h.OnError(w, r, h.teamUC.Leave(r.Context(), user.ID), "PostTeamsLeave", "Leave") {
		return
	}

	helper.RenderOK(w, r, map[string]string{"message": "Successfully left the team"})
}

// Disband team
// (DELETE /teams/me)
func (h *Server) DeleteTeamsMe(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if h.OnError(w, r, h.teamUC.DisbandTeam(r.Context(), user.ID), "DeleteTeamsMe", "DisbandTeam") {
		return
	}

	helper.RenderOK(w, r, map[string]string{"message": "Team disbanded successfully"})
}

// Kick member
// (DELETE /teams/members/{ID})
func (h *Server) DeleteTeamsMembersID(w http.ResponseWriter, r *http.Request, ID string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	targetuserID, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.teamUC.KickMember(r.Context(), user.ID, targetuserID), "DeleteTeamsMembersID", "KickMember") {
		return
	}

	helper.RenderOK(w, r, map[string]string{"message": "Member kicked successfully"})
}

// Get my team
// (GET /teams/my)
func (h *Server) GetTeamsMy(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	team, members, err := h.teamUC.GetMyTeam(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, entityError.ErrTeamNotFound) {
			helper.RenderError(w, r, http.StatusNotFound, "user is not in a team")
			return
		}
		if h.OnError(w, r, err, "GetTeamsMy", "GetMyTeam") {
			return
		}
	}

	helper.RenderOK(w, r, response.FromTeamWithMembers(team, members))
}

// Create solo team
// (POST /teams/solo)
func (h *Server) PostTeamsSolo(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.RequestCreateTeamRequest](
		w, r, h.validator, h.logger, "PostTeamsSolo",
	)
	if !ok {
		return
	}

	_, confirmReset := request.CreateTeamRequestToParams(&req)
	team, err := h.teamUC.CreateSoloTeam(r.Context(), user.ID, confirmReset)
	if h.OnError(w, r, err, "PostTeamsSolo", "CreateSoloTeam") {
		return
	}

	helper.RenderCreated(w, r, response.FromTeam(team))
}

// Transfer captainship
// (POST /teams/transfer-captain)
func (h *Server) PostTeamsTransferCaptain(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RequestTransferCaptainRequest](
		w, r, h.validator, h.logger, "PostTeamsTransferCaptain",
	)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	newCaptainID := request.TransferCaptainRequestToNewCaptainID(&req)
	newCaptainuuid, ok := helper.ParseUUID(w, r, newCaptainID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.teamUC.TransferCaptain(r.Context(), user.ID, newCaptainuuid), "PostTeamsTransferCaptain", "TransferCaptain") {
		return
	}

	helper.RenderOK(w, r, map[string]string{"message": "Captainship transferred successfully"})
}

// Get team by ID
// (GET /teams/{ID})
func (h *Server) GetTeamsID(w http.ResponseWriter, r *http.Request, ID string) {
	teamuuid, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	team, err := h.teamUC.GetByID(r.Context(), teamuuid)
	if h.OnError(w, r, err, "GetTeamsID", "GetByID") {
		return
	}

	helper.RenderOK(w, r, response.FromTeamWithoutToken(team))
}

// Ban team
// (POST /admin/teams/{ID}/ban)
func (h *Server) PostAdminTeamsIDBan(w http.ResponseWriter, r *http.Request, ID string) {
	teamuuid, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.RequestBanTeamRequest](
		w, r, h.validator, h.logger, "PostAdminTeamsIDBan",
	)
	if !ok {
		return
	}

	reason := request.BanTeamRequestToReason(&req)
	if err := h.teamUC.BanTeam(r.Context(), teamuuid, reason); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAdminTeamsIDBan")
		helper.HandleError(w, r, err)
		return
	}

	helper.RenderOK(w, r, map[string]string{"message": "team banned"})
}

// Unban team
// (DELETE /admin/teams/{ID}/ban)
func (h *Server) DeleteAdminTeamsIDBan(w http.ResponseWriter, r *http.Request, ID string) {
	teamuuid, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.teamUC.UnbanTeam(r.Context(), teamuuid), "DeleteAdminTeamsIDBan", "UnbanTeam") {
		return
	}

	helper.RenderOK(w, r, map[string]string{"message": "team unbanned"})
}

// Set team hidden status
// (PATCH /admin/teams/{ID}/hidden)
func (h *Server) PatchAdminTeamsIDHidden(w http.ResponseWriter, r *http.Request, ID string) {
	teamuuid, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.RequestSetHiddenRequest](
		w, r, h.validator, h.logger, "PatchAdminTeamsIDHidden",
	)
	if !ok {
		return
	}

	hidden := request.SetHiddenRequestToHidden(&req)
	if h.OnError(w, r, h.teamUC.SetHidden(r.Context(), teamuuid, hidden), "PatchAdminTeamsIDHidden", "SetHidden") {
		return
	}

	helper.RenderOK(w, r, map[string]bool{"hidden": hidden})
}
