package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Create award
// (POST /admin/awards)
func (h *Server) PostAdminAwards(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.CreateAwardRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminAwards",
	)
	if !ok {
		return
	}

	teamID, value, description, err := request.CreateAwardRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminAwards", "RequestConversion") {
		return
	}

	award, err := h.team.AwardUC.Create(r.Context(), teamID, value, description, user.ID)
	if h.OnError(w, r, err, "PostAdminAwards", "Create") {
		return
	}

	helper.RenderCreated(w, r, response.FromAward(award))
}

// Get all awards
// (GET /admin/awards)
func (h *Server) GetAdminAwards(w http.ResponseWriter, r *http.Request, params openapi.GetAdminAwardsParams) {
	if params.TeamID != nil && *params.TeamID != "" {
		teamIDParsed, ok := helper.ParseUUID(w, r, *params.TeamID)
		if !ok {
			return
		}
		awards, err := h.team.AwardUC.GetByTeamID(r.Context(), teamIDParsed)
		if h.OnError(w, r, err, "GetAdminAwards", "GetByTeamID") {
			return
		}
		helper.RenderOK(w, r, response.FromAwardList(awards))
		return
	}

	awards, err := h.team.AwardUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetAdminAwards", "GetAll") {
		return
	}

	helper.RenderOK(w, r, response.FromAwardList(awards))
}

// Get award by ID
// (GET /admin/awards/{ID})
func (h *Server) GetAdminAwardsID(w http.ResponseWriter, r *http.Request, ID string) {
	awardIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	award, err := h.team.AwardUC.GetByID(r.Context(), awardIDParsed)
	if h.OnError(w, r, err, "GetAdminAwardsID", "GetByID") {
		return
	}

	helper.RenderOK(w, r, response.FromAward(award))
}

// Delete award
// (DELETE /admin/awards/{ID})
func (h *Server) DeleteAdminAwardsID(w http.ResponseWriter, r *http.Request, ID string) {
	awardIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	err := h.team.AwardUC.Delete(r.Context(), awardIDParsed)
	if h.OnError(w, r, err, "DeleteAdminAwardsID", "Delete") {
		return
	}

	helper.RenderNoContent(w, r)
}

// Get awards by team
// (GET /admin/awards/team/{teamID})
func (h *Server) GetAdminAwardsTeamTeamID(w http.ResponseWriter, r *http.Request, teamID string) {
	teamIDParsed, ok := helper.ParseUUID(w, r, teamID)
	if !ok {
		return
	}

	awards, err := h.team.AwardUC.GetByTeamID(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetAdminAwardsTeamTeamID", "GetByTeamID") {
		return
	}

	helper.RenderOK(w, r, response.FromAwardList(awards))
}
