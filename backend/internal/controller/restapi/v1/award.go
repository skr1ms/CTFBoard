package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Create award
// (POST /admin/awards)
func (h *Server) PostAdminAwards(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestCreateAwardRequest](
		w, r, h.validator, h.logger, "PostAdminAwards",
	)
	if !ok {
		return
	}

	teamID, value, description, err := request.CreateAwardRequestToParams(&req)
	if err != nil {
		RenderError(w, r, http.StatusBadRequest, "invalid team ID")
		return
	}

	user, ok := RequireUser(w, r)
	if !ok {
		return
	}

	award, err := h.awardUC.Create(r.Context(), teamID, value, description, user.ID)
	if h.OnError(w, r, err, "PostAdminAwards", "Create") {
		return
	}

	RenderCreated(w, r, response.FromAward(award))
}

// Get awards by team
// (GET /admin/awards/team/{teamID})
func (h *Server) GetAdminAwardsTeamTeamID(w http.ResponseWriter, r *http.Request, teamID string) {
	teamuuid, ok := ParseUUID(w, r, teamID)
	if !ok {
		return
	}

	awards, err := h.awardUC.GetByTeamID(r.Context(), teamuuid)
	if h.OnError(w, r, err, "GetAdminAwardsTeamTeamID", "GetByTeamID") {
		return
	}

	RenderOK(w, r, response.FromAwardList(awards))
}
