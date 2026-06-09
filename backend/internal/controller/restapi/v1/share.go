package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (POST /shares).
func (h *Server) PostShares(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	teamID, ok := helper.RequireTeamID(w, r, user, h.OnError, "PostShares")
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.CreateShareRequest](w, r, h.infra.Validator)
	if !ok {
		return
	}

	params, err := request.CreateShareRequestToParams(&req, user.ID, teamID)
	if h.OnError(w, r, err, "PostShares", "CreateShareRequestToParams") {
		return
	}

	link, err := h.comp.ShareUC.CreateSolveShare(r.Context(), params)
	if h.OnError(w, r, err, "PostShares", "CreateSolveShare") {
		return
	}

	httputil.RenderCreated(w, r, response.FromShareLink(link))
}

// (GET /shares/solve).
func (h *Server) GetSharesSolve(w http.ResponseWriter, r *http.Request, params openapi.GetSharesSolveParams) {
	share, err := h.comp.ShareUC.ResolveSolveShare(r.Context(), uuid.UUID(params.SolveID), params.Mac)
	if h.OnError(w, r, err, "GetSharesSolve", "ResolveSolveShare") {
		return
	}

	html, err := response.SolveShareHTML(share)
	if h.OnError(w, r, err, "GetSharesSolve", "SolveShareHTML") {
		return
	}

	helper.RenderTrustedHTML(w, http.StatusOK, html)
}
