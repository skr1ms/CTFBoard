package v1

import (
	"net/http"
	"slices"
	"strings"

	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

var publicConfigKeys = []string{
	"ctf_name", "ctf_description", "ctf_logo", "tos_url", "privacy_url",
	"theme_color_primary", "theme_color_secondary", "theme_header_html", "theme_footer_html", "theme_dark_mode",
	"social_github", "social_discord", "social_twitter", "social_website",
}

// Get all configs (admin)
// (GET /admin/configs).
func (h *Server) GetAdminConfigs(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.CompetitionParamUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetAdminConfigs", "GetAll") {
		return
	}

	httputil.RenderOK(w, r, response.FromConfigResponseList(list))
}

// Get config categories (admin)
// (GET /admin/configs/categories).
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

	slices.SortFunc(out, func(a, b openapi.ConfigCategoryItem) int { return strings.Compare(a.Name, b.Name) })
	httputil.RenderOK(w, r, out)
}

// Get configs by category (admin)
// (GET /admin/configs/category/{category}).
func (h *Server) GetAdminConfigsCategory(w http.ResponseWriter, r *http.Request, category string) {
	list, err := h.admin.CompetitionParamUC.GetByCategory(r.Context(), category)
	if h.OnError(w, r, err, "GetAdminConfigsCategory", "GetByCategory") {
		return
	}

	httputil.RenderOK(w, r, response.FromConfigResponseList(list))
}

// Set configs in batch (admin)
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

	if err := request.ValidateBatchSetConfigRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PutAdminConfigsBatch", "Validate") {
		return
	}

	params, err := request.BatchSetConfigRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminConfigsBatch", "BatchSetConfigRequestToParams") {
		return
	}

	clientIP := kitMiddleware.GetClientIPFromContext(r.Context())
	if h.OnError(w, r, h.admin.CompetitionParamUC.SetBatch(r.Context(), params, user.ID, clientIP), "PutAdminConfigsBatch", "SetBatch") {
		return
	}

	httputil.RenderOK(w, r, response.Message("configs updated"))
}

// Get public configs
// (GET /configs/public).
func (h *Server) GetConfigsPublic(w http.ResponseWriter, r *http.Request) {
	all, err := h.admin.CompetitionParamUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetConfigsPublic", "GetAll") {
		return
	}

	byKey := make(map[string]*domain.CompetitionParam, len(all))
	for _, p := range all {
		byKey[p.Key] = p
	}

	list := make([]*domain.CompetitionParam, 0, len(publicConfigKeys))
	for _, key := range publicConfigKeys {
		if p, ok := byKey[key]; ok {
			list = append(list, p)

			continue
		}

		if def, ok := domain.GetConfigDef(key); ok {
			list = append(list, &domain.CompetitionParam{
				Key: def.Key, Value: def.DefaultValue, ValueType: def.ValueType,
				Category: def.Category, Description: def.Description,
			})
		}
	}

	httputil.RenderOK(w, r, response.FromConfigListToPublicMap(list))
}

// Get config by key (admin)
// (GET /admin/configs/{key}).
func (h *Server) GetAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	cfg, err := h.admin.CompetitionParamUC.Get(r.Context(), key)
	if h.OnError(w, r, err, "GetAdminConfigsKey", "Get") {
		return
	}

	httputil.RenderOK(w, r, response.FromConfig(cfg))
}

// Set config (admin)
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

	if err := request.ValidateSetConfigRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PutAdminConfigsKey", "Validate") {
		return
	}

	clientIP := kitMiddleware.GetClientIPFromContext(r.Context())

	params, err := request.SetConfigRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminConfigsKey", "SetConfigRequestToParams") {
		return
	}

	if h.OnError(w, r, h.admin.CompetitionParamUC.Set(r.Context(), key, params.Value, params.Description, params.ValueType, params.Category, user.ID, clientIP), "PutAdminConfigsKey", "Set") {
		return
	}

	httputil.RenderOK(w, r, response.Message("config updated"))
}

// Delete config (admin)
// (DELETE /admin/configs/{key}).
func (h *Server) DeleteAdminConfigsKey(w http.ResponseWriter, r *http.Request, key string) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	clientIP := kitMiddleware.GetClientIPFromContext(r.Context())
	if h.OnError(w, r, h.admin.CompetitionParamUC.Delete(r.Context(), key, user.ID, clientIP), "DeleteAdminConfigsKey", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// Get app settings
// (GET /admin/settings).
func (h *Server) GetAdminSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.admin.SettingsUC.Get(r.Context())
	if h.OnError(w, r, err, "GetAdminSettings", "Get") {
		return
	}

	httputil.RenderOK(w, r, response.FromAppSettings(s))
}

// Update app settings
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

	clientIP := kitMiddleware.GetClientIPFromContext(r.Context())

	if err := request.ValidateUpdateAppSettingsRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PutAdminSettings", "Validate") {
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

	httputil.RenderOK(w, r, response.Message("settings updated"))
}
