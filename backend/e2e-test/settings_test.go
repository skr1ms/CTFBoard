package e2e_test

import (
	"net/http"
	"testing"
)

func TestSettings_Admin_Get(t *testing.T) {
	h := NewE2EHelper(t, setupE2E(t), TestPool)
	_, tokenAdmin := h.SetupCompetition("admin_settings")

	obj := h.GetAdminSettings(tokenAdmin)
	obj.ContainsKey("app_name")
	obj.ContainsKey("verify_emails")
	obj.ContainsKey("frontend_url")
	obj.ContainsKey("cors_origins")
	obj.ContainsKey("scoreboard_visible")
	obj.ContainsKey("registration_open")
}

func TestSettings_Admin_Put(t *testing.T) {
	h := NewE2EHelper(t, setupE2E(t), TestPool)
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

	obj := h.GetAdminSettings(tokenAdmin)
	obj.HasValue("app_name", "CTFBoard Test")
	obj.HasValue("frontend_url", "https://test.example.com")
	obj.HasValue("submit_limit_per_user", 20)
}
