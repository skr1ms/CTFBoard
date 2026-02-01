package v1

import (
	"net/http"

	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

func (h *Server) GetAdminSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.settingsUC.Get(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAdminSettings - Get")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, map[string]any{
		"app_name":                  s.AppName,
		"verify_emails":             s.VerifyEmails,
		"frontend_url":              s.FrontendURL,
		"cors_origins":              s.CORSOrigins,
		"resend_enabled":            s.ResendEnabled,
		"resend_from_email":         s.ResendFromEmail,
		"resend_from_name":          s.ResendFromName,
		"verify_ttl_hours":          s.VerifyTTLHours,
		"reset_ttl_hours":           s.ResetTTLHours,
		"submit_limit_per_user":     s.SubmitLimitPerUser,
		"submit_limit_duration_min": s.SubmitLimitDurationMin,
		"scoreboard_visible":        s.ScoreboardVisible,
		"registration_open":         s.RegistrationOpen,
		"updated_at":                s.UpdatedAt,
	})
}

func (h *Server) PutAdminSettings(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.UpdateAppSettingsRequest](
		w, r, h.validator, h.logger, "UpdateAppSettings",
	)
	if !ok {
		return
	}

	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	clientIP := httputil.GetClientIP(r)
	s := req.ToAppSettings(1)

	if err := h.settingsUC.Update(r.Context(), s, user.ID, clientIP); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PutAdminSettings - Update")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, map[string]string{"message": "settings updated"})
}
