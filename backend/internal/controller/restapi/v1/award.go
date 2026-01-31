package v1

import (
	"net/http"

	"github.com/google/uuid"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

// Create award
// (POST /admin/awards)
func (h *Server) PostAdminAwards(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.CreateAwardRequest](
		w, r, h.validator, h.logger, "PostAdminAwards",
	)
	if !ok {
		return
	}

	teamuuid, err := uuid.Parse(req.TeamID)
	if err != nil {
		httputil.RenderError(w, r, http.StatusBadRequest, "invalid team ID")
		return
	}

	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	award, err := h.awardUC.Create(r.Context(), teamuuid, req.Value, req.Description, user.ID)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAdminAwards")
		handleError(w, r, err)
		return
	}

	res := response.FromAward(award)

	httputil.RenderCreated(w, r, res)
}

// Get awards by team
// (GET /admin/awards/team/{teamID})
func (h *Server) GetAdminAwardsTeamTeamID(w http.ResponseWriter, r *http.Request, teamID string) {
	teamuuid, err := uuid.Parse(teamID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	awards, err := h.awardUC.GetByTeamID(r.Context(), teamuuid)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAdminAwardsTeamTeamID")
		handleError(w, r, err)
		return
	}

	res := response.FromAwardList(awards)

	httputil.RenderOK(w, r, res)
}
