package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/e2e-test/helper"
	"github.com/stretchr/testify/require"
)

// GET /competition/status: returns status, start_time, end_time (public, no auth).
func TestCompetition_Status(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	resp := h.GetCompetitionStatus()
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Status)
	require.NotNil(t, resp.JSON200.StartTime)
	require.NotNil(t, resp.JSON200.EndTime)
}

// PUT /admin/competition: pause/resume; when paused, POST /challenges/{ID}/submit returns 403; when resumed, submit succeeds.
func TestCompetition_UpdateAndEnforce(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

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

	statusResp := h.GetCompetitionStatus()
	require.NotNil(t, statusResp.JSON200)
	require.Equal(t, "paused", *statusResp.JSON200.Status)

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
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, tokenAdmin := h.SetupCompetition("admin_get")

	obj := h.GetAdminCompetition(tokenAdmin)
	require.NotNil(t, obj.JSON200)
	require.NotNil(t, obj.JSON200.Name)
	require.NotNil(t, obj.JSON200.StartTime)
	require.NotNil(t, obj.JSON200.EndTime)
}

// GET /admin/competition: non-admin gets 403 Forbidden.
func TestCompetition_Admin_Get_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_get_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_comp")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.GetAdminCompetitionExpectStatus(tokenUser, http.StatusForbidden)
}

// PUT /admin/competition: non-admin gets 403 Forbidden.
func TestCompetition_Admin_Put_Forbidden(t *testing.T) {
	t.Helper()
	setupE2E(t)
	h := helper.NewE2EHelper(t, nil, TestPool, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_put_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_put")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	now := time.Now().UTC()
	h.PutAdminCompetitionExpectStatus(tokenUser, map[string]any{
		"name": "X", "start_time": now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time": now.Add(24 * time.Hour).Format(time.RFC3339), "is_paused": false,
		"allow_team_switch": true, "mode": "flexible",
	}, http.StatusForbidden)
}
