package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Create award
// (POST /admin/awards).
func (h *Server) PostAdminAwards(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.CreateAwardRequest](
		w, r, h.infra.Validator,
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

	httputil.RenderCreated(w, r, response.FromAward(award))
}

// Get all awards
// (GET /admin/awards).
func (h *Server) GetAdminAwards(w http.ResponseWriter, r *http.Request, params openapi.GetAdminAwardsParams) {
	if params.TeamID != nil && *params.TeamID != "" {
		teamIDParsed, ok := httputil.ParseUUID(w, r, *params.TeamID)
		if !ok {
			return
		}

		awards, err := h.team.AwardUC.GetByTeamID(r.Context(), teamIDParsed)
		if h.OnError(w, r, err, "GetAdminAwards", "GetByTeamID") {
			return
		}

		httputil.RenderOK(w, r, response.FromAwardList(awards))

		return
	}

	awards, err := h.team.AwardUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetAdminAwards", "GetAll") {
		return
	}

	httputil.RenderOK(w, r, response.FromAwardList(awards))
}

// Get award by ID
// (GET /admin/awards/{ID}).
func (h *Server) GetAdminAwardsID(w http.ResponseWriter, r *http.Request, ID string) {
	awardIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	award, err := h.team.AwardUC.GetByID(r.Context(), awardIDParsed)
	if h.OnError(w, r, err, "GetAdminAwardsID", "GetByID") {
		return
	}

	httputil.RenderOK(w, r, response.FromAward(award))
}

// Delete award
// (DELETE /admin/awards/{ID}).
func (h *Server) DeleteAdminAwardsID(w http.ResponseWriter, r *http.Request, ID string) {
	awardIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	err := h.team.AwardUC.Delete(r.Context(), awardIDParsed)
	if h.OnError(w, r, err, "DeleteAdminAwardsID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// Get awards by team
// (GET /admin/awards/team/{teamID}).
func (h *Server) GetAdminAwardsTeamTeamID(w http.ResponseWriter, r *http.Request, teamID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, teamID)
	if !ok {
		return
	}

	awards, err := h.team.AwardUC.GetByTeamID(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "GetAdminAwardsTeamTeamID", "GetByTeamID") {
		return
	}

	httputil.RenderOK(w, r, response.FromAwardList(awards))
}
