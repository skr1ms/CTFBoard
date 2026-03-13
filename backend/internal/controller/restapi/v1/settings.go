package v1

import (
	"net/http"
	"sort"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

var publicConfigExactKeys = []string{"ctf_name", "ctf_description", "ctf_logo", "tos_url", "privacy_url"}

func publicConfigKeys() []string {
	seen := make(map[string]struct{})
	for _, k := range publicConfigExactKeys {
		seen[k] = struct{}{}
	}
	for k := range entity.ConfigRegistry {
		if strings.HasPrefix(k, "theme_") || strings.HasPrefix(k, "social_") {
			seen[k] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Get all configs (admin)
// (GET /admin/configs)
func (h *Server) GetAdminConfigs(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.CompetitionParamUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetAdminConfigs", "GetAll") {
		return
	}
	helper.RenderOK(w, r, response.FromConfigResponseList(list))
}

// (GET /admin/configs/categories)
func (h *Server) GetAdminConfigsCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.CompetitionParamUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetAdminConfigsCategories", "GetAll") {
		return
	}
	counts := make(map[string]int)
	for _, p := range list {
		if p.Category != "" {
			counts[p.Category]++
		}
	}
	out := make([]openapi.ConfigCategoryItem, 0, len(counts))
	for name, count := range counts {
		out = append(out, openapi.ConfigCategoryItem{Name: name, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	helper.RenderOK(w, r, out)
}

// (GET /admin/configs/category/{category})
func (h *Server) GetAdminConfigsCategory(w http.ResponseWriter, r *http.Request, category string) {
	list, err := h.admin.CompetitionParamUC.GetByCategory(r.Context(), category)
	if h.OnError(w, r, err, "GetAdminConfigsCategory", "GetByCategory") {
		return
	}
	helper.RenderOK(w, r, response.FromConfigResponseList(list))
}

// (PUT /admin/configs/batch)
func (h *Server) PutAdminConfigsBatch(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.BatchSetConfigRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PutAdminConfigsBatch",
	)
	if !ok {
		return
	}
	params, err := request.BatchSetConfigRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminConfigsBatch", "BatchSetConfigRequestToParams") {
		return
	}
	clientIP := helper.GetClientIP(r, h.infra.TrustedProxyCIDRs)
	if h.OnError(w, r, h.admin.CompetitionParamUC.SetBatch(r.Context(), params, user.ID, clientIP), "PutAdminConfigsBatch", "SetBatch") {
		return
	}
	helper.RenderOK(w, r, response.Message("configs updated"))
}

// (GET /configs/public)
func (h *Server) GetConfigsPublic(w http.ResponseWriter, r *http.Request) {
	keys := publicConfigKeys()
	list := make([]*entity.CompetitionParam, 0, len(keys))
	for _, key := range keys {
		p, err := h.admin.CompetitionParamUC.Get(r.Context(), key)
		if err != nil {
			if def, ok := entity.ConfigRegistry[key]; ok {
				list = append(list, &entity.CompetitionParam{
					Key: def.Key, Value: def.DefaultValue, ValueType: def.ValueType,
					Category: def.Category, Description: def.Description,
				})
			}
			continue
		}
		list = append(list, p)
	}
	helper.RenderOK(w, r, response.FromConfigList(list))
}

// Get config by key (admin)
// (GET /admin/configs/{key})
func (h *Server) GetAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	cfg, err := h.admin.CompetitionParamUC.Get(r.Context(), key)
	if h.OnError(w, r, err, "GetAdminConfigsKey", "Get") {
		return
	}
	helper.RenderOK(w, r, response.FromConfig(cfg))
}

// Set config (admin)
// (PUT /admin/configs/{key})
func (h *Server) PutAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.SetConfigRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PutAdminConfigsKey",
	)
	if !ok {
		return
	}
	clientIP := helper.GetClientIP(r, h.infra.TrustedProxyCIDRs)
	params, err := request.SetConfigRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminConfigsKey", "SetConfigRequestToParams") {
		return
	}
	if h.OnError(w, r, h.admin.CompetitionParamUC.Set(r.Context(), key, params.Value, params.Description, params.ValueType, user.ID, clientIP), "PutAdminConfigsKey", "Set") {
		return
	}
	helper.RenderOK(w, r, response.Message("config updated"))
}

// Delete config (admin)
// (DELETE /admin/configs/{key})
func (h *Server) DeleteAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	clientIP := helper.GetClientIP(r, h.infra.TrustedProxyCIDRs)
	if h.OnError(w, r, h.admin.CompetitionParamUC.Delete(r.Context(), key, user.ID, clientIP), "DeleteAdminConfigsKey", "Delete") {
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
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	req, ok := helper.DecodeAndValidate[openapi.UpdateAppSettingsRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PutAdminSettings",
	)
	if !ok {
		return
	}

	current, err := h.admin.SettingsUC.Get(r.Context())
	if h.OnError(w, r, err, "PutAdminSettings", "GetCurrentSettings") {
		return
	}

	clientIP := helper.GetClientIP(r, h.infra.TrustedProxyCIDRs)

	if err := request.ValidateUpdateAppSettingsRequest(&req); h.OnError(w, r, err, "PutAdminSettings", "Validate") {
		return
	}

	s := request.UpdateAppSettingsRequestToEntity(&req, current.ID, current)

	if h.OnError(w, r, h.admin.SettingsUC.Update(r.Context(), s, user.ID, clientIP), "PutAdminSettings", "Update") {
		return
	}
	if h.infra.RateLimitConfigCache != nil {
		h.infra.RateLimitConfigCache.Invalidate()
	}
	if h.infra.ScoreboardVisibilityCache != nil {
		h.infra.ScoreboardVisibilityCache.Invalidate()
	}

	helper.RenderOK(w, r, response.Message("settings updated"))
}
