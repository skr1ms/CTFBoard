package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// GET /admin/settings: admin gets app settings (app_name, verify_emails, scoreboard_visible, etc.).
func TestSettings_Admin_Get(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)
	_, tokenAdmin := h.SetupCompetition("admin_settings")

	resp := h.GetAdminSettings(tokenAdmin)
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.AppName)
	require.NotNil(t, resp.JSON200.VerifyEmails)
	require.NotNil(t, resp.JSON200.FrontendURL)
	require.NotNil(t, resp.JSON200.CorsOrigins)
	require.NotNil(t, resp.JSON200.ScoreboardVisible)
	require.NotNil(t, resp.JSON200.RegistrationOpen)
}

// PUT /admin/settings: admin updates app settings; GET reflects new values.
func TestSettings_Admin_Put(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)
	_, tokenAdmin := h.SetupCompetition("admin_settings_put")

	body := map[string]any{
		"app_name":                  "CTFBoard Test",
		"verify_emails":             true,
		"frontend_url":              "https://test.example.com",
		"cors_origins":              "https://test.example.com",
		"resend_enabled":            false,
		"resend_from_email":         "noreply@test.local",
		"resend_from_name":          "CTFBoard",
		"verify_ttl_hours":          24,
		"reset_ttl_hours":           1,
		"submit_limit_per_user":     20,
		"submit_limit_duration_min": 1,
		"scoreboard_visible":        "public",
		"registration_open":         true,
	}
	h.PutAdminSettings(tokenAdmin, body, http.StatusOK)

	resp := h.GetAdminSettings(tokenAdmin)
	require.NotNil(t, resp.JSON200)
	require.Equal(t, "CTFBoard Test", *resp.JSON200.AppName)
	require.Equal(t, "https://test.example.com", *resp.JSON200.FrontendURL)
	require.NotNil(t, resp.JSON200.SubmitLimitPerUser)
	require.Equal(t, 20, *resp.JSON200.SubmitLimitPerUser)
}

// GET /admin/settings: non-admin gets 403 Forbidden.
func TestSettings_Admin_Get_Forbidden(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, _ = h.SetupCompetition("admin_set_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_set")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.GetAdminSettingsExpectStatus(tokenUser, http.StatusForbidden)
}

// PUT /admin/settings: non-admin gets 403 Forbidden.
func TestSettings_Admin_Put_Forbidden(t *testing.T) {
	setupE2E(t)
	h := NewE2EHelper(t, nil, TestPool)

	_, _ = h.SetupCompetition("admin_set_put_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_put_set")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	body := map[string]any{
		"app_name": "X", "verify_emails": true, "frontend_url": "https://x.com",
		"cors_origins": "*", "scoreboard_visible": "public", "registration_open": true,
	}
	h.PutAdminSettings(tokenUser, body, http.StatusForbidden)
}
