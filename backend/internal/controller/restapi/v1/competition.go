package v1

import (
	"net/http"
	"time"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get competition status
// (GET /competition/status)
func (h *Server) GetCompetitionStatus(w http.ResponseWriter, r *http.Request) {
	comp, err := h.competitionUC.Get(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetCompetitionStatus - Get")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromCompetitionStatus(comp))
}

// Get competition
// (GET /admin/competition)
func (h *Server) GetAdminCompetition(w http.ResponseWriter, r *http.Request) {
	comp, err := h.competitionUC.Get(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAdminCompetition - Get")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromCompetition(comp))
}

// Update competition
// (PUT /admin/competition)
func (h *Server) PutAdminCompetition(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestUpdateCompetitionRequest](
		w, r, h.validator, h.logger, "UpdateCompetition",
	)
	if !ok {
		return
	}

	if req.Name == "" {
		RenderError(w, r, http.StatusBadRequest, "name is required")
		return
	}

	if err := validateCompetitionTimes(req.StartTime, req.EndTime, req.FreezeTime); err != "" {
		RenderError(w, r, http.StatusBadRequest, err)
		return
	}

	comp := request.UpdateCompetitionRequestToEntity(&req, 1)

	user, ok := middleware.GetUser(r.Context())
	if !ok {
		RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	clientIP := GetClientIP(r)

	if err := h.competitionUC.Update(r.Context(), comp, user.ID, clientIP); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PutAdminCompetition - Update")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, map[string]string{"message": "competition updated"})
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
