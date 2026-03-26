package helper

import (
	"context"
	"net/http"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

type adminSettingsBody struct {
	AppName                          string `mapstructure:"app_name"`
	CorsOrigins                      string `mapstructure:"cors_origins"`
	FrontendURL                      string `mapstructure:"frontend_url"`
	ResendFromEmail                  string `mapstructure:"resend_from_email"`
	ResendFromName                   string `mapstructure:"resend_from_name"`
	SubmitLimitPerUser               *int   `mapstructure:"submit_limit_per_user"`
	SubmitLimitDurationMin           *int   `mapstructure:"submit_limit_duration_min"`
	VerifyTTLHours                   *int   `mapstructure:"verify_ttl_hours"`
	ResetTTLHours                    *int   `mapstructure:"reset_ttl_hours"`
	VerifyEmails                     *bool  `mapstructure:"verify_emails"`
	RegistrationOpen                 *bool  `mapstructure:"registration_open"`
	ResendEnabled                    *bool  `mapstructure:"resend_enabled"`
	ScoreboardVisible                string `mapstructure:"scoreboard_visible"`
	WriteupEnabled                   *bool  `mapstructure:"writeup_enabled"`
	RateLimitForgotPasswordPerMinute *int   `mapstructure:"rate_limit_forgot_password_per_minute"`
}

func bodyToPutAdminSettingsRequest(body map[string]any) (openapi.PutAdminSettingsJSONRequestBody, error) {
	var b adminSettingsBody

	err := decodeMap(body, &b)
	if err != nil {
		return openapi.PutAdminSettingsJSONRequestBody{}, err
	}

	req := openapi.PutAdminSettingsJSONRequestBody{
		AppName:         new(b.AppName),
		CorsOrigins:     new(b.CorsOrigins),
		FrontendURL:     new(b.FrontendURL),
		ResendFromEmail: new(b.ResendFromEmail),
		ResendFromName:  new(b.ResendFromName),
	}
	if b.SubmitLimitPerUser != nil && *b.SubmitLimitPerUser != 0 {
		req.SubmitLimitPerUser = b.SubmitLimitPerUser
	}

	if b.SubmitLimitDurationMin != nil && *b.SubmitLimitDurationMin != 0 {
		req.SubmitLimitDurationMin = b.SubmitLimitDurationMin
	}

	if b.VerifyTTLHours != nil && *b.VerifyTTLHours != 0 {
		req.VerifyTTLHours = b.VerifyTTLHours
	}

	if b.ResetTTLHours != nil && *b.ResetTTLHours != 0 {
		req.ResetTTLHours = b.ResetTTLHours
	}

	if b.VerifyEmails != nil {
		req.VerifyEmails = b.VerifyEmails
	}

	if b.RegistrationOpen != nil {
		req.RegistrationOpen = b.RegistrationOpen
	}

	if b.ResendEnabled != nil {
		req.ResendEnabled = b.ResendEnabled
	}

	if b.ScoreboardVisible != "" {
		req.ScoreboardVisible = (*openapi.UpdateAppSettingsRequestScoreboardVisible)(&b.ScoreboardVisible)
	}

	if b.WriteupEnabled != nil {
		req.WriteupEnabled = b.WriteupEnabled
	}

	if b.RateLimitForgotPasswordPerMinute != nil && *b.RateLimitForgotPasswordPerMinute >= 1 {
		req.RateLimitForgotPasswordPerMinute = b.RateLimitForgotPasswordPerMinute
	}

	return req, nil
}

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

	req, err := bodyToPutAdminSettingsRequest(body)
	require.NoError(h.t, err)
	resp, err := h.client.PutAdminSettingsWithResponse(context.Background(), req, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "put admin settings")

	return resp
}

func (h *E2EHelper) PutAdminSettingsExpectOneOf(token string, body map[string]any, allowedStatuses []int) *openapi.PutAdminSettingsResponse {
	h.t.Helper()

	req, err := bodyToPutAdminSettingsRequest(body)
	require.NoError(h.t, err)
	resp, err := h.client.PutAdminSettingsWithResponse(context.Background(), req, WithBearerToken(token))
	require.NoError(h.t, err)
	require.Contains(h.t, allowedStatuses, resp.StatusCode(), "put admin settings: status %d not in %v body=%s", resp.StatusCode(), allowedStatuses, string(resp.Body))

	return resp
}
