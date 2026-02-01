package e2e_test

import (
	"net/http"
	"testing"
	"time"
)

// GET /competition/status: returns status, start_time, end_time (public, no auth).
func TestCompetition_Status(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	h.GetCompetitionStatus().
		ContainsKey("status").
		ContainsKey("start_time").
		ContainsKey("end_time")
}

// PUT /admin/competition: pause/resume; when paused, POST /challenges/{ID}/submit returns 403; when resumed, submit succeeds.
func TestCompetition_UpdateAndEnforce(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, _, tokenAdmin := h.RegisterAdmin("admin_comp")

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title":       "Comp Challenge",
		"description": "Test competition challenge",
		"flag":        "FLAG{comp}",
		"points":      100,
		"category":    "web",
		"is_hidden":   false,
	})

	_, _, tokenUser := h.RegisterUserAndLogin("comp_user")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	now := time.Now().UTC()
	h.UpdateCompetition(tokenAdmin, map[string]any{
		"name":              "Comp Name",
		"start_time":        now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":          now.Add(24 * time.Hour).Format(time.RFC3339),
		"is_paused":         true,
		"allow_team_switch": true,
		"mode":              "flexible",
	})

	h.GetCompetitionStatus().
		Value("status").String().IsEqual("paused")

	h.SubmitFlag(tokenUser, challengeID, "FLAG{comp}", http.StatusForbidden)

	h.UpdateCompetition(tokenAdmin, map[string]any{
		"name":              "Comp Name",
		"start_time":        now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":          now.Add(24 * time.Hour).Format(time.RFC3339),
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	})

	h.SubmitFlag(tokenUser, challengeID, "FLAG{comp}", http.StatusOK)
}

// GET /admin/competition: admin gets full competition config (name, start_time, end_time, freeze_time, etc.).
func TestCompetition_Admin_Get(t *testing.T) {
	e := setupE2E(t)
	h := NewE2EHelper(t, e, TestPool)

	_, tokenAdmin := h.SetupCompetition("admin_get")

	obj := h.GetAdminCompetition(tokenAdmin)
	obj.ContainsKey("name")
	obj.ContainsKey("start_time")
	obj.ContainsKey("end_time")
}
