package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (POST /appeals).
func (h *Server) PostAppeals(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.CreateAppealRequest](w, r, h.infra.Validator)
	if !ok {
		return
	}

	appeal, err := h.user.AppealUC.CreateAppeal(r.Context(), user.ID, req.Message)
	if h.OnError(w, r, err, "PostAppeals", "CreateAppeal") {
		return
	}

	httputil.RenderCreated(w, r, response.FromBanAppeal(appeal))
}

// (GET /appeals/me).
func (h *Server) GetAppealsMe(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	appeals, err := h.user.AppealUC.GetAppealsByUser(r.Context(), user.ID)
	if h.OnError(w, r, err, "GetAppealsMe", "GetAppealsByUser") {
		return
	}

	httputil.RenderOK(w, r, response.FromBanAppeals(appeals))
}

// (GET /admin/appeals).
func (h *Server) GetAdminAppeals(w http.ResponseWriter, r *http.Request, params openapi.GetAdminAppealsParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.IsAdmin(user) {
		h.OnError(w, r, helper.ErrAccessDenied, "GetAdminAppeals", "RequireAdmin")

		return
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.user.AppealUC.ListAppeals(r.Context(), request.AppealDecisionFromParams(params), page, perPage)
	if h.OnError(w, r, err, "GetAdminAppeals", "ListAppeals") {
		return
	}

	httputil.RenderOK(w, r, response.FromBanAppealList(result.Data, result.Total, result.Page, result.PerPage))
}

// (PATCH /admin/appeals/{ID}).
func (h *Server) PatchAdminAppealsID(w http.ResponseWriter, r *http.Request, id string) {
	actor, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.IsAdmin(actor) {
		h.OnError(w, r, helper.ErrAccessDenied, "PatchAdminAppealsID", "RequireAdmin")

		return
	}

	appealID, ok := httputil.ParseUUID(w, r, id)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.ReviewAppealRequest](w, r, h.infra.Validator)
	if !ok {
		return
	}

	decision, adminResponse := request.ReviewAppealRequestToParams(&req)

	appeal, err := h.user.AppealUC.ReviewAppeal(r.Context(), appealID, decision, adminResponse, actor.ID)
	if h.OnError(w, r, err, "PatchAdminAppealsID", "ReviewAppeal") {
		return
	}

	httputil.RenderOK(w, r, response.FromBanAppeal(appeal))
}
