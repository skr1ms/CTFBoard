package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
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

	items := make([]openapi.BanAppealResponse, len(appeals))
	for i, a := range appeals {
		items[i] = response.FromBanAppeal(a)
	}

	httputil.RenderOK(w, r, openapi.BanAppealListResponse{Appeals: &items})
}

// (GET /admin/appeals).
func (h *Server) GetAdminAppeals(w http.ResponseWriter, r *http.Request, params openapi.GetAdminAppealsParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.IsAdmin(user) {
		h.OnError(w, r, apperr.ErrAccessDenied, "GetAdminAppeals", "RequireAdmin")

		return
	}

	var decision *domain.AppealDecision

	if params.Decision != nil {
		d := domain.AppealDecision(string(*params.Decision))
		decision = &d
	}

	page, perPage := h.pageParams(r.Context(), params.Page, params.PerPage)

	result, err := h.user.AppealUC.ListAppeals(r.Context(), decision, page, perPage)
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
		h.OnError(w, r, apperr.ErrAccessDenied, "PatchAdminAppealsID", "RequireAdmin")

		return
	}

	appealID, err := uuid.Parse(id)
	if err != nil {
		h.OnError(w, r, apperr.ErrAppealNotFound, "PatchAdminAppealsID", "ParseID")

		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.ReviewAppealRequest](w, r, h.infra.Validator)
	if !ok {
		return
	}

	decision := domain.AppealDecision(string(req.Decision))

	var adminResponse *string

	if req.AdminResponse != nil && *req.AdminResponse != "" {
		adminResponse = req.AdminResponse
	}

	appeal, err := h.user.AppealUC.ReviewAppeal(r.Context(), appealID, decision, adminResponse, actor.ID)
	if h.OnError(w, r, err, "PatchAdminAppealsID", "ReviewAppeal") {
		return
	}

	httputil.RenderOK(w, r, response.FromBanAppeal(appeal))
}
