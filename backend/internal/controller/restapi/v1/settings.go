package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get app settings
// (GET /admin/settings)
func (h *Server) GetAdminSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.settingsUC.Get(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAdminSettings - Get")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, response.FromAppSettings(s))
}

// Update app settings
// (PUT /admin/settings)
func (h *Server) PutAdminSettings(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestUpdateAppSettingsRequest](
		w, r, h.validator, h.logger, "UpdateAppSettings",
	)
	if !ok {
		return
	}

	user, ok := middleware.GetUser(r.Context())
	if !ok {
		RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	clientIP := GetClientIP(r)
	s := request.UpdateAppSettingsRequestToEntity(&req, 1)

	if err := h.settingsUC.Update(r.Context(), s, user.ID, clientIP); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PutAdminSettings - Update")
		handleError(w, r, err)
		return
	}

	RenderOK(w, r, map[string]string{"message": "settings updated"})
}
