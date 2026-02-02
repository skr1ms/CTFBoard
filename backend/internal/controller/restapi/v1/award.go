package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
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

	user, ok := middleware.GetUser(r.Context())
	if !ok {
		RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	award, err := h.awardUC.Create(r.Context(), teamID, value, description, user.ID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAdminAwards")
		handleError(w, r, err)
		return
	}

	RenderCreated(w, r, response.FromAward(award))
}

// Get awards by team
// (GET /admin/awards/team/{teamID})
func (h *Server) GetAdminAwardsTeamTeamID(w http.ResponseWriter, r *http.Request, teamID string) {
	teamuuid, err := uuid.Parse(teamID)
	if err != nil {
		RenderInvalidID(w, r)
		return
	}

	awards, err := h.awardUC.GetByTeamID(r.Context(), teamuuid)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAdminAwardsTeamTeamID")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromAwardList(awards))
}
