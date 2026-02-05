package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func getPagePerPageRatings(page, perPage *int) (int, int) {
	p := 1
	if page != nil && *page > 0 {
		p = *page
	}
	pp := 20
	if perPage != nil && *perPage > 0 {
		pp = *perPage
	}
	if pp > 100 {
		pp = 100
	}
	return p, pp
}

// Get global ratings
// (GET /ratings)
func (h *Server) GetRatings(w http.ResponseWriter, r *http.Request, params openapi.GetRatingsParams) {
	page, perPage := getPagePerPageRatings(params.Page, params.PerPage)
	items, total, err := h.ratingUC.GetGlobalRatings(r.Context(), page, perPage)
	if h.OnError(w, r, err, "GetRatings", "GetGlobalRatings") {
		return
	}
	helper.RenderOK(w, r, response.FromGlobalRatingsList(items, total))
}

// Get team rating
// (GET /ratings/team/{ID})
func (h *Server) GetRatingsTeamID(w http.ResponseWriter, r *http.Request, id string) {
	teamID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}
	global, eventRatings, err := h.ratingUC.GetTeamRating(r.Context(), teamID)
	if h.OnError(w, r, err, "GetRatingsTeamID", "GetTeamRating") {
		return
	}
	helper.RenderOK(w, r, response.FromTeamRating(global, eventRatings))
}

// Get CTF events list (admin)
// (GET /admin/ctf-events)
func (h *Server) GetAdminCtfEvents(w http.ResponseWriter, r *http.Request) {
	list, err := h.ratingUC.GetCTFEvents(r.Context())
	if h.OnError(w, r, err, "GetAdminCtfEvents", "GetCTFEvents") {
		return
	}
	helper.RenderOK(w, r, response.FromCTFEventList(list))
}

// Create CTF event (admin)
// (POST /admin/ctf-events)
func (h *Server) PostAdminCtfEvents(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RequestCreateCTFEventRequest](w, r, h.validator, h.logger, "PostAdminCtfEvents")
	if !ok {
		return
	}
	weight := 1.0
	if req.Weight != nil {
		weight = float64(*req.Weight)
	}
	event, err := h.ratingUC.CreateCTFEvent(r.Context(), req.Name, req.StartTime, req.EndTime, weight)
	if h.OnError(w, r, err, "PostAdminCtfEvents", "CreateCTFEvent") {
		return
	}
	helper.RenderCreated(w, r, response.FromCTFEvent(event))
}

// Finalize CTF event (admin)
// (POST /admin/ctf-events/{ID}/finalize)
func (h *Server) PostAdminCtfEventsIDFinalize(w http.ResponseWriter, r *http.Request, id string) {
	eventID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}
	if h.OnError(w, r, h.ratingUC.FinalizeCTFEvent(r.Context(), eventID), "PostAdminCtfEventsIDFinalize", "FinalizeCTFEvent") {
		return
	}
	helper.RenderNoContent(w, r)
}
