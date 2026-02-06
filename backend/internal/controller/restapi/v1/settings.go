package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get all configs (admin)
// (GET /admin/configs)
func (h *Server) GetAdminConfigs(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.DynamicConfigUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetAdminConfigs", "GetAll") {
		return
	}
	out := make([]openapi.ResponseConfigResponse, len(list))
	for i, c := range list {
		out[i] = response.FromConfig(c)
	}
	helper.RenderOK(w, r, out)
}

// Get config by key (admin)
// (GET /admin/configs/{key})
func (h *Server) GetAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	cfg, err := h.admin.DynamicConfigUC.Get(r.Context(), key)
	if h.OnError(w, r, err, "GetAdminConfigsKey", "Get") {
		return
	}
	helper.RenderOK(w, r, response.FromConfig(cfg))
}

// Set config (admin)
// (PUT /admin/configs/{key})
func (h *Server) PutAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	req, ok := helper.DecodeAndValidate[openapi.RequestSetConfigRequest](w, r, h.infra.Validator, h.infra.Logger, "PutAdminConfigsKey")
	if !ok {
		return
	}
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	clientIP := helper.GetClientIP(r)
	valueType := request.SetConfigRequestToValueType(req.ValueType)
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	if h.OnError(w, r, h.admin.DynamicConfigUC.Set(r.Context(), key, req.Value, description, valueType, user.ID, clientIP), "PutAdminConfigsKey", "Set") {
		return
	}
	helper.RenderOK(w, r, map[string]string{"message": "config updated"})
}

// Delete config (admin)
// (DELETE /admin/configs/{key})
func (h *Server) DeleteAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	clientIP := helper.GetClientIP(r)
	if h.OnError(w, r, h.admin.DynamicConfigUC.Delete(r.Context(), key, user.ID, clientIP), "DeleteAdminConfigsKey", "Delete") {
		return
	}
	helper.RenderNoContent(w, r)
}

// Get app settings
// (GET /admin/settings)
func (h *Server) GetAdminSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.admin.SettingsUC.Get(r.Context())
	if h.OnError(w, r, err, "GetAdminSettings", "Get") {
		return
	}

	helper.RenderOK(w, r, response.FromAppSettings(s))
}

// Update app settings
// (PUT /admin/settings)
func (h *Server) PutAdminSettings(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.RequestUpdateAppSettingsRequest](
		w, r, h.infra.Validator, h.infra.Logger, "UpdateAppSettings",
	)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	clientIP := helper.GetClientIP(r)
	s := request.UpdateAppSettingsRequestToEntity(&req, 1)

	if h.OnError(w, r, h.admin.SettingsUC.Update(r.Context(), s, user.ID, clientIP), "PutAdminSettings", "Update") {
		return
	}

	helper.RenderOK(w, r, map[string]string{"message": "settings updated"})
}
