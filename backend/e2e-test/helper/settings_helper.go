package helper

import (
	"context"
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetAdminSettings(token string) *openapi.GetAdminSettingsResponse {
	h.t.Helper()
	return h.GetAdminSettingsExpectStatus(token, http.StatusOK)
}

func (h *E2EHelper) GetAdminSettingsExpectStatus(token string, expectStatus int) *openapi.GetAdminSettingsResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminSettingsWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin settings")
	return resp
}

func (h *E2EHelper) PutAdminSettings(token string, body map[string]any, expectStatus int) *openapi.PutAdminSettingsResponse {
	h.t.Helper()
	req := openapi.PutAdminSettingsJSONRequestBody{
		AppName:         getStr(body, "app_name", ""),
		CorsOrigins:     getStr(body, "cors_origins", ""),
		FrontendURL:     getStr(body, "frontend_url", ""),
		ResendFromEmail: getStr(body, "resend_from_email", ""),
		ResendFromName:  getStr(body, "resend_from_name", ""),
	}
	if v := getInt(body, "submit_limit_per_user"); v != 0 {
		req.SubmitLimitPerUser = &v
	}
	if v := getInt(body, "submit_limit_duration_min"); v != 0 {
		req.SubmitLimitDurationMin = &v
	}
	if v := getInt(body, "verify_ttl_hours"); v != 0 {
		req.VerifyTTLHours = &v
	}
	if v := getInt(body, "reset_ttl_hours"); v != 0 {
		req.ResetTTLHours = &v
	}
	if v, ok := body["verify_emails"].(bool); ok {
		req.VerifyEmails = &v
	}
	if v, ok := body["registration_open"].(bool); ok {
		req.RegistrationOpen = &v
	}
	if v, ok := body["resend_enabled"].(bool); ok {
		req.ResendEnabled = &v
	}
	if v, ok := body["scoreboard_visible"].(string); ok {
		req.ScoreboardVisible = (*openapi.RequestUpdateAppSettingsRequestScoreboardVisible)(&v)
	}
	resp, err := h.client.PutAdminSettingsWithResponse(context.Background(), req, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "put admin settings")
	return resp
}
