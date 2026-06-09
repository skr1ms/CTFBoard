package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (POST /admin/teams/{ID}/ban).
func (h *Server) PostAdminTeamsIDBan(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.BanTeamRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	reason, banMembers := request.BanTeamRequestToParams(&req)
	if h.OnError(w, r, h.team.AdminUC.BanTeam(r.Context(), teamIDParsed, reason, banMembers, user.ID), "PostAdminTeamsIDBan", "BanTeam") {
		return
	}

	httputil.RenderOK(w, r, response.Message("team banned"))
}

// (DELETE /admin/teams/{ID}/ban).
func (h *Server) DeleteAdminTeamsIDBan(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if h.OnError(w, r, h.team.AdminUC.UnbanTeam(r.Context(), teamIDParsed, user.ID), "DeleteAdminTeamsIDBan", "UnbanTeam") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (PATCH /admin/teams/{ID}/hidden).
func (h *Server) PatchAdminTeamsIDHidden(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.SetHiddenRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	hidden, err := request.SetHiddenRequestToParams(&req)
	if h.OnError(w, r, err, "PatchAdminTeamsIDHidden", "SetHiddenRequestToParams") {
		return
	}

	if h.OnError(w, r, h.team.AdminUC.SetHidden(r.Context(), teamIDParsed, *hidden), "PatchAdminTeamsIDHidden", "SetHidden") {
		return
	}

	httputil.RenderOK(w, r, response.FromHiddenStatus(*hidden))
}

// (GET /admin/teams).
func (h *Server) GetAdminTeams(w http.ResponseWriter, r *http.Request, params openapi.GetAdminTeamsParams) {
	searchQ, ok := helper.ParseOptionalSearchQuery(w, r, params.Q, maxSearchQueryLen, h.OnError, "GetAdminTeams", "Q")
	if !ok {
		return
	}

	banStatus, err := request.AdminTeamsBanStatusFromParams(params)
	if h.OnError(w, r, err, "GetAdminTeams", "BanStatus") {
		return
	}

	visibility, err := request.AdminTeamsVisibilityFromParams(params)
	if h.OnError(w, r, err, "GetAdminTeams", "Visibility") {
		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.team.AdminUC.AdminListTeams(r.Context(), searchQ, banStatus, visibility, page, perPage)
	if h.OnError(w, r, err, "GetAdminTeams", "AdminListTeams") {
		return
	}

	httputil.RenderOK(w, r, response.FromAdminTeamList(result.Data, result.Total, result.Page, result.PerPage))
}

// (POST /admin/teams/bulk/ban).
func (h *Server) PostAdminTeamsBulkBan(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.BulkBanTeamsRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	ids, reason, banMembers := request.BulkBanTeamsRequestToParams(&req)

	result, err := h.team.AdminUC.BanTeams(r.Context(), ids, reason, banMembers, user.ID)
	if h.OnError(w, r, err, "PostAdminTeamsBulkBan", "BanTeams") {
		return
	}

	httputil.RenderOK(w, r, response.BulkAction("teams banned", result.AffectedCount))
}

// (POST /admin/teams/bulk/unban).
func (h *Server) PostAdminTeamsBulkUnban(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.BulkTeamIDsRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	result, err := h.team.AdminUC.UnbanTeams(r.Context(), request.BulkTeamIDsRequestToParams(&req), user.ID)
	if h.OnError(w, r, err, "PostAdminTeamsBulkUnban", "UnbanTeams") {
		return
	}

	httputil.RenderOK(w, r, response.BulkAction("teams unbanned", result.AffectedCount))
}

// (PATCH /admin/teams/bulk/hidden).
func (h *Server) PatchAdminTeamsBulkHidden(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.BulkSetHiddenRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	ids, hidden := request.BulkSetHiddenRequestToParams(&req)

	result, err := h.team.AdminUC.SetHiddenBulk(r.Context(), ids, hidden)
	if h.OnError(w, r, err, "PatchAdminTeamsBulkHidden", "SetHiddenBulk") {
		return
	}

	httputil.RenderOK(w, r, response.BulkAction("team visibility updated", result.AffectedCount))
}

// (PATCH /admin/teams/{ID}).
func (h *Server) PatchAdminTeamsID(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.AdminUpdateTeamRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	name, captainID, bracketID, isHidden, err := request.AdminUpdateTeamRequestToParams(&req)
	if h.OnError(w, r, err, "PatchAdminTeamsID", "RequestConversion") {
		return
	}

	team, err := h.team.AdminUC.AdminUpdate(r.Context(), teamIDParsed, name, captainID, bracketID, isHidden)
	if h.OnError(w, r, err, "PatchAdminTeamsID", "AdminUpdate") {
		return
	}

	members, err := h.team.AdminUC.AdminGetMembers(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "PatchAdminTeamsID", "AdminGetMembers") {
		return
	}

	httputil.RenderOK(w, r, response.FromAdminTeam(team, new(len(members))))
}

// (DELETE /admin/teams/{ID}).
func (h *Server) DeleteAdminTeamsID(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.team.AdminUC.AdminDelete(r.Context(), teamIDParsed), "DeleteAdminTeamsID", "AdminDelete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (GET /admin/teams/{ID}/members).
func (h *Server) GetAdminTeamsIDMembers(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	members, err := h.team.AdminUC.AdminGetMembers(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetAdminTeamsIDMembers", "AdminGetMembers") {
		return
	}

	httputil.RenderOK(w, r, response.FromAdminUserSlice(members))
}

// (GET /admin/teams/{ID}/missing-challenges).
func (h *Server) GetAdminTeamsIDMissingChallenges(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	challenges, err := h.challenge.AdminUC.GetMissingChallengesByTeamID(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetAdminTeamsIDMissingChallenges", "GetMissingChallengesByTeamID") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallenges(challenges))
}

// (POST /admin/teams/{ID}/members).
func (h *Server) PostAdminTeamsIDMembers(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.AdminAddMemberRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	userID, err := request.AdminAddMemberRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminTeamsIDMembers", "RequestConversion") {
		return
	}

	if h.OnError(w, r, h.team.AdminUC.AdminAddMember(r.Context(), teamIDParsed, userID), "PostAdminTeamsIDMembers", "AdminAddMember") {
		return
	}

	httputil.RenderOK(w, r, response.Message("member added"))
}

// (DELETE /admin/teams/{ID}/members/{userID}).
func (h *Server) DeleteAdminTeamsIDMembersUserID(w http.ResponseWriter, r *http.Request, ID, userID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	memberIDParsed, ok := httputil.ParseUUID(w, r, userID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.team.AdminUC.AdminRemoveMember(r.Context(), teamIDParsed, memberIDParsed), "DeleteAdminTeamsIDMembersUserID", "AdminRemoveMember") {
		return
	}

	httputil.RenderNoContent(w, r)
}
