package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /admin/configs).
func (h *Server) GetAdminConfigs(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.CompetitionParamUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetAdminConfigs", "GetAll") {
		return
	}

	httputil.RenderOK(w, r, response.FromConfigResponseList(list))
}

// (GET /admin/configs/categories).
func (h *Server) GetAdminConfigsCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.CompetitionParamUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetAdminConfigsCategories", "GetAll") {
		return
	}

	httputil.RenderOK(w, r, response.FromConfigCategories(list))
}

// (GET /admin/configs/category/{category}).
func (h *Server) GetAdminConfigsCategory(w http.ResponseWriter, r *http.Request, category string) {
	list, err := h.admin.CompetitionParamUC.GetByCategory(r.Context(), category)
	if h.OnError(w, r, err, "GetAdminConfigsCategory", "GetByCategory") {
		return
	}

	httputil.RenderOK(w, r, response.FromConfigResponseList(list))
}

// (PUT /admin/configs/batch).
func (h *Server) PutAdminConfigsBatch(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.BatchSetConfigRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.BatchSetConfigRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminConfigsBatch", "BatchSetConfigRequestToParams") {
		return
	}

	clientIP := helper.ClientIP(r)
	if h.OnError(w, r, h.admin.CompetitionParamUC.SetBatch(r.Context(), params, user.ID, clientIP), "PutAdminConfigsBatch", "SetBatch") {
		return
	}

	httputil.RenderOK(w, r, response.Message("configs updated"))
}

// (GET /configs/public).
func (h *Server) GetConfigsPublic(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.CompetitionParamUC.GetPublic(r.Context())
	if h.OnError(w, r, err, "GetConfigsPublic", "GetPublic") {
		return
	}

	setPublicCache(w, cacheShort, false)
	httputil.RenderOK(w, r, response.FromConfigListToPublicMap(list))
}

// (GET /admin/configs/{key}).
func (h *Server) GetAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	cfg, err := h.admin.CompetitionParamUC.Get(r.Context(), key)
	if h.OnError(w, r, err, "GetAdminConfigsKey", "Get") {
		return
	}

	httputil.RenderOK(w, r, response.FromConfig(cfg))
}

// (PUT /admin/configs/{key}).
func (h *Server) PutAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.SetConfigRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	clientIP := helper.ClientIP(r)

	params, err := request.SetConfigRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminConfigsKey", "SetConfigRequestToParams") {
		return
	}

	if h.OnError(w, r, h.admin.CompetitionParamUC.Set(r.Context(), key, params.Value, params.Description, params.ValueType, params.Category, user.ID, clientIP), "PutAdminConfigsKey", "Set") {
		return
	}

	httputil.RenderOK(w, r, response.Message("config updated"))
}

// (DELETE /admin/configs/{key}).
func (h *Server) DeleteAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	clientIP := helper.ClientIP(r)
	if h.OnError(w, r, h.admin.CompetitionParamUC.Delete(r.Context(), key, user.ID, clientIP), "DeleteAdminConfigsKey", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (GET /admin/settings).
func (h *Server) GetAdminSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.admin.SettingsUC.Get(r.Context())
	if h.OnError(w, r, err, "GetAdminSettings", "Get") {
		return
	}

	httputil.RenderOK(w, r, response.FromAppSettings(s))
}

// PutAdminSettings persists new application settings.
// (PUT /admin/settings).
func (h *Server) PutAdminSettings(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateAppSettingsRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	current, err := h.admin.SettingsUC.Get(r.Context())
	if h.OnError(w, r, err, "PutAdminSettings", "GetCurrentSettings") {
		return
	}

	clientIP := helper.ClientIP(r)

	s := request.UpdateAppSettingsRequestToEntity(&req, current.ID, current)

	if h.OnError(w, r, h.admin.SettingsUC.Update(r.Context(), s, user.ID, clientIP), "PutAdminSettings", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.Message("settings updated"))
}
