package v1

import (
	"net/http"
	"time"

	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

// Get competition status (Public)
func (h *Server) GetCompetitionStatus(w http.ResponseWriter, r *http.Request) {
	comp, err := h.competitionUC.Get(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetCompetitionStatus - Get")
		handleError(w, r, err)
		return
	}

	res := response.FromCompetitionStatus(comp)

	httputil.RenderOK(w, r, res)
}

// Get competition (Admin)
func (h *Server) GetAdminCompetition(w http.ResponseWriter, r *http.Request) {
	comp, err := h.competitionUC.Get(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAdminCompetition - Get")
		handleError(w, r, err)
		return
	}

	res := response.FromCompetition(comp)

	httputil.RenderOK(w, r, res)
}

// Update competition (Admin)
func (h *Server) PutAdminCompetition(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.UpdateCompetitionRequest](
		w, r, h.validator, h.logger, "UpdateCompetition",
	)
	if !ok {
		return
	}

	if err := validateCompetitionTimes(req.StartTime, req.EndTime, req.FreezeTime); err != "" {
		httputil.RenderError(w, r, http.StatusBadRequest, err)
		return
	}

	comp := req.ToCompetition(1)

	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	clientIP := httputil.GetClientIP(r)

	if err := h.competitionUC.Update(r.Context(), comp, user.ID, clientIP); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PutAdminCompetition - Update")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "competition updated"})
}

// validateCompetitionTimes validates the competition time constraints
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
