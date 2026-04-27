package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /competition/status).
func (h *Server) GetCompetitionStatus(w http.ResponseWriter, r *http.Request) {
	comp, err := h.comp.CompetitionUC.Get(r.Context())
	if h.OnError(w, r, err, "GetCompetitionStatus", "Get") {
		return
	}

	setPublicCache(w, cacheShort, false)
	httputil.RenderOK(w, r, response.FromCompetitionStatus(comp))
}

// (GET /admin/competition).
func (h *Server) GetAdminCompetition(w http.ResponseWriter, r *http.Request) {
	comp, err := h.comp.CompetitionUC.Get(r.Context())
	if h.OnError(w, r, err, "GetAdminCompetition", "Get") {
		return
	}

	httputil.RenderOK(w, r, response.FromCompetition(comp))
}

// (PUT /admin/competition).
func (h *Server) PutAdminCompetition(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateCompetitionRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	if h.OnError(w, r, request.ValidateCompetitionTimes(&req), "PutAdminCompetition", "ValidateCompetitionTimes") {
		return
	}

	comp := request.UpdateCompetitionRequestToEntity(&req)
	optionals := request.UpdateCompetitionOptionalsFromRequest(&req)
	clientIP := kitMiddleware.GetClientIPFromContext(r.Context())

	err := h.comp.CompetitionUC.Update(r.Context(), comp, optionals, user.ID, clientIP)
	if h.OnError(w, r, err, "PutAdminCompetition", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.Message("competition updated"))
}
