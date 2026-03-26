package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

// Create team
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

// Join team
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

// Leave team
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

// Disband team
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

// Kick member
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

	if h.OnError(w, r, h.team.TeamUC.KickMember(r.Context(), user.ID, memberIDParsed), "DeleteTeamsMembersID", "KickMember") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// Get my team
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

// Create solo team
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

// Transfer captainship
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

	if err := request.ValidateTransferCaptainRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PostTeamsTransferCaptain", "Validate") {
		return
	}

	newCaptainID, err := request.TransferCaptainRequestToParams(&req)
	if h.OnError(w, r, err, "PostTeamsTransferCaptain", "RequestConversion") {
		return
	}

	if h.OnError(w, r, h.team.TeamUC.TransferCaptain(r.Context(), user.ID, newCaptainID), "PostTeamsTransferCaptain", "TransferCaptain") {
		return
	}

	httputil.RenderOK(w, r, response.Message("captainship transferred"))
}

// Get team by ID
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

	team, err := h.team.TeamUC.GetByID(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetTeamsID", "GetByID") {
		return
	}

	if (team.IsBanned || team.IsHidden) && user.Role != domain.RoleAdmin {
		h.OnError(w, r, httperr.ErrTeamNotFound, "GetTeamsID", "team banned or hidden")

		return
	}

	httputil.RenderOK(w, r, response.FromTeamWithoutToken(team))
}

// Ban team
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
	if h.OnError(w, r, h.team.TeamUC.BanTeam(r.Context(), teamIDParsed, reason, banMembers, user.ID), "PostAdminTeamsIDBan", "BanTeam") {
		return
	}

	httputil.RenderOK(w, r, response.Message("team banned"))
}

// Unban team
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

	if h.OnError(w, r, h.team.TeamUC.UnbanTeam(r.Context(), teamIDParsed, user.ID), "DeleteAdminTeamsIDBan", "UnbanTeam") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// Set team hidden status
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

	if h.OnError(w, r, h.team.TeamUC.SetHidden(r.Context(), teamIDParsed, *hidden), "PatchAdminTeamsIDHidden", "SetHidden") {
		return
	}

	httputil.RenderOK(w, r, response.FromHiddenStatus(*hidden))
}

// List teams with search and pagination
// (GET /teams).
func (h *Server) GetTeams(w http.ResponseWriter, r *http.Request, params openapi.GetTeamsParams) {
	q := ""

	if params.Q != nil && *params.Q != "" {
		var ok bool

		q, ok = helper.ParseSearchQuery(w, r, params.Q, maxSearchQueryLen, h.OnError, "GetTeams", "Q")
		if !ok {
			return
		}
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.team.TeamUC.ListTeams(r.Context(), &q, page, perPage)
	if h.OnError(w, r, err, "GetTeams", "ListTeams") {
		return
	}

	httputil.RenderOK(w, r, response.FromTeamList(result.Data, result.Total, result.Page, result.PerPage))
}

// Get my team's solves
// (GET /teams/me/solves).
func (h *Server) GetTeamsMeSolves(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if user.TeamID == nil {
		h.OnError(w, r, httperr.ErrUserNotInTeam, "GetTeamsMeSolves", "TeamIDNil")

		return
	}

	solves, err := h.team.TeamUC.GetTeamSolves(r.Context(), *user.TeamID)
	if h.OnError(w, r, err, "GetTeamsMeSolves", "GetTeamSolves") {
		return
	}

	httputil.RenderOK(w, r, response.FromSolveWithDetailsList(solves))
}

// Get team's solves by team ID
// (GET /teams/{ID}/solves).
func (h *Server) GetTeamsIDSolves(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	isOwnTeam := user.TeamID != nil && *user.TeamID == teamIDParsed
	if !isOwnTeam && user.Role != domain.RoleAdmin {
		h.OnError(w, r, httperr.ErrAccessDenied, "GetTeamsIDSolves", "AccessCheck")

		return
	}

	solves, err := h.team.TeamUC.GetTeamSolves(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetTeamsIDSolves", "GetTeamSolves") {
		return
	}

	httputil.RenderOK(w, r, response.FromSolveWithDetailsList(solves))
}

// Get my team's failed submissions
// (GET /teams/me/fails).
func (h *Server) GetTeamsMeFails(w http.ResponseWriter, r *http.Request, params openapi.GetTeamsMeFailsParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if user.TeamID == nil {
		h.OnError(w, r, httperr.ErrUserNotInTeam, "GetTeamsMeFails", "TeamIDNil")

		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	fails, err := h.team.TeamUC.GetTeamFails(r.Context(), *user.TeamID, page, perPage)
	if h.OnError(w, r, err, "GetTeamsMeFails", "GetTeamFails") {
		return
	}

	httputil.RenderOK(w, r, response.FromFailListPublic(fails.Data, fails.Total, fails.Page, fails.PerPage))
}

// Get team's failed submissions by team ID
// (GET /teams/{ID}/fails).
func (h *Server) GetTeamsIDFails(w http.ResponseWriter, r *http.Request, ID string, params openapi.GetTeamsIDFailsParams) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	isOwnTeam := user.TeamID != nil && *user.TeamID == teamIDParsed
	if !isOwnTeam && user.Role != domain.RoleAdmin {
		h.OnError(w, r, httperr.ErrAccessDenied, "GetTeamsIDFails", "AccessCheck")

		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	fails, err := h.team.TeamUC.GetTeamFails(r.Context(), teamIDParsed, page, perPage)
	if h.OnError(w, r, err, "GetTeamsIDFails", "GetTeamFails") {
		return
	}

	httputil.RenderOK(w, r, response.FromFailListPublic(fails.Data, fails.Total, fails.Page, fails.PerPage))
}

// Get my team's awards
// (GET /teams/me/awards).
func (h *Server) GetTeamsMeAwards(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if user.TeamID == nil {
		h.OnError(w, r, httperr.ErrUserNotInTeam, "GetTeamsMeAwards", "TeamIDNil")

		return
	}

	awards, err := h.team.TeamUC.GetTeamAwards(r.Context(), *user.TeamID)
	if h.OnError(w, r, err, "GetTeamsMeAwards", "GetTeamAwards") {
		return
	}

	httputil.RenderOK(w, r, response.FromAwardList(awards))
}

// Get team's awards by team ID
// (GET /teams/{ID}/awards).
func (h *Server) GetTeamsIDAwards(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	isOwnTeam := user.TeamID != nil && *user.TeamID == teamIDParsed
	if !isOwnTeam && user.Role != domain.RoleAdmin {
		h.OnError(w, r, httperr.ErrAccessDenied, "GetTeamsIDAwards", "AccessCheck")

		return
	}

	awards, err := h.team.TeamUC.GetTeamAwards(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetTeamsIDAwards", "GetTeamAwards") {
		return
	}

	httputil.RenderOK(w, r, response.FromAwardList(awards))
}

// List teams (admin) with search and pagination
// (GET /admin/teams).
func (h *Server) GetAdminTeams(w http.ResponseWriter, r *http.Request, params openapi.GetAdminTeamsParams) {
	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	var searchQ *string

	if params.Q != nil && *params.Q != "" {
		s, ok := helper.ParseSearchQuery(w, r, params.Q, maxSearchQueryLen, h.OnError, "GetAdminTeams", "Q")
		if !ok {
			return
		}

		searchQ = &s
	}

	result, err := h.team.TeamUC.AdminListTeams(r.Context(), searchQ, page, perPage)
	if h.OnError(w, r, err, "GetAdminTeams", "AdminListTeams") {
		return
	}

	httputil.RenderOK(w, r, response.FromAdminTeamList(result.Data, result.Total, result.Page, result.PerPage))
}

// Update team (admin)
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

	team, err := h.team.TeamUC.AdminUpdate(r.Context(), teamIDParsed, name, captainID, bracketID, isHidden)
	if h.OnError(w, r, err, "PatchAdminTeamsID", "AdminUpdate") {
		return
	}

	members, err := h.team.TeamUC.AdminGetMembers(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "PatchAdminTeamsID", "AdminGetMembers") {
		return
	}

	httputil.RenderOK(w, r, response.FromAdminTeam(team, new(len(members))))
}

// Delete team (admin)
// (DELETE /admin/teams/{ID}).
func (h *Server) DeleteAdminTeamsID(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.team.TeamUC.AdminDelete(r.Context(), teamIDParsed), "DeleteAdminTeamsID", "AdminDelete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// Get team members (admin)
// (GET /admin/teams/{ID}/members).
func (h *Server) GetAdminTeamsIDMembers(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	members, err := h.team.TeamUC.AdminGetMembers(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetAdminTeamsIDMembers", "AdminGetMembers") {
		return
	}

	httputil.RenderOK(w, r, response.FromAdminUserSlice(members))
}

// Get team missing challenges (admin)
// (GET /admin/teams/{ID}/missing-challenges).
func (h *Server) GetAdminTeamsIDMissingChallenges(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	challenges, err := h.challenge.ChallengeUC.GetMissingChallengesByTeamID(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetAdminTeamsIDMissingChallenges", "GetMissingChallengesByTeamID") {
		return
	}

	httputil.RenderOK(w, r, response.FromChallenges(challenges))
}

// Add team member (admin)
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

	if h.OnError(w, r, h.team.TeamUC.AdminAddMember(r.Context(), teamIDParsed, userID), "PostAdminTeamsIDMembers", "AdminAddMember") {
		return
	}

	httputil.RenderOK(w, r, response.Message("member added"))
}

// Remove team member (admin)
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

	if h.OnError(w, r, h.team.TeamUC.AdminRemoveMember(r.Context(), teamIDParsed, memberIDParsed), "DeleteAdminTeamsIDMembersUserID", "AdminRemoveMember") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// Update my team
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

	team, err := h.team.TeamUC.UpdateMyTeam(r.Context(), user.ID, request.UpdateTeamRequestToParams(&req))
	if h.OnError(w, r, err, "PatchTeamsMe", "UpdateMyTeam") {
		return
	}

	httputil.RenderOK(w, r, response.FromTeam(team))
}

// Get team invite token
// (GET /teams/me/invite).
func (h *Server) GetTeamsMeInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	team, err := h.team.TeamUC.GetInviteToken(r.Context(), user.ID)
	if h.OnError(w, r, err, "GetTeamsMeInvite", "GetInviteToken") {
		return
	}

	httputil.RenderOK(w, r, response.FromTeamInvite(team.InviteToken.String()))
}

// Regenerate team invite token
// (POST /teams/me/invite).
func (h *Server) PostTeamsMeInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	team, err := h.team.TeamUC.RegenerateInviteToken(r.Context(), user.ID)
	if h.OnError(w, r, err, "PostTeamsMeInvite", "RegenerateInviteToken") {
		return
	}

	httputil.RenderOK(w, r, response.FromTeamInvite(team.InviteToken.String()))
}
