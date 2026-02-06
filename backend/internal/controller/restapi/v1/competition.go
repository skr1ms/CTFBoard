package v1

import (
	"net/http"
	"time"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get competition status
// (GET /competition/status)
func (h *Server) GetCompetitionStatus(w http.ResponseWriter, r *http.Request) {
	comp, err := h.comp.CompetitionUC.Get(r.Context())
	if h.OnError(w, r, err, "GetCompetitionStatus", "Get") {
		return
	}

	helper.RenderOK(w, r, response.FromCompetitionStatus(comp))
}

// Get competition
// (GET /admin/competition)
func (h *Server) GetAdminCompetition(w http.ResponseWriter, r *http.Request) {
	comp, err := h.comp.CompetitionUC.Get(r.Context())
	if h.OnError(w, r, err, "GetAdminCompetition", "Get") {
		return
	}

	helper.RenderOK(w, r, response.FromCompetition(comp))
}

// Update competition
// (PUT /admin/competition)
func (h *Server) PutAdminCompetition(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RequestUpdateCompetitionRequest](
		w, r, h.infra.Validator, h.infra.Logger, "UpdateCompetition",
	)
	if !ok {
		return
	}

	if req.Name == "" {
		helper.RenderError(w, r, http.StatusBadRequest, "name is required")
		return
	}

	if err := validateCompetitionTimes(req.StartTime, req.EndTime, req.FreezeTime); err != "" {
		helper.RenderError(w, r, http.StatusBadRequest, err)
		return
	}

	comp := request.UpdateCompetitionRequestToEntity(&req, 1)

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	clientIP := helper.GetClientIP(r)

	err := h.comp.CompetitionUC.Update(r.Context(), comp, user.ID, clientIP)
	if h.OnError(w, r, err, "PutAdminCompetition", "Update") {
		return
	}

	helper.RenderOK(w, r, map[string]string{"message": "competition updated"})
}

func validateCompetitionTimes(startTime, endTime, freezeTime *time.Time) string {
	if endTime != nil && startTime != nil && endTime.Before(*startTime) {
		return "end_time must be after start_time"
	}
	if freezeTime != nil && endTime != nil && freezeTime.After(*endTime) {
		return "freeze_time must be before end_time"
	}
	if freezeTime != nil && startTime != nil && freezeTime.Before(*startTime) {
		return "freeze_time must be after start_time"
	}
	return ""
}
