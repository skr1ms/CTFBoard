package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/e2e-test/helper"
)

// GET /competition/status: returns status, start_time, end_time (public, no auth).
func TestCompetition_Status(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("status")

	resp := h.GetCompetitionStatus()
	require.NotNil(t, resp.JSON200)
	require.NotNil(t, resp.JSON200.Status)
	require.NotNil(t, resp.JSON200.StartTime)
	require.NotNil(t, resp.JSON200.EndTime)
}

// PUT /admin/competition: pause/resume; when paused, POST /challenges/{ID}/submit returns 403; when resumed, submit succeeds.
func TestCompetition_UpdateAndEnforce(t *testing.T) {
	t.Helper()
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), nil)
	setCompetitionMode("flexible")
	time.Sleep(6 * time.Second)

	h.PutAdminCompetitionExpectStatus(tokenAdmin, map[string]any{
		"name":              "Comp Name",
		"is_paused":         true,
		"allow_team_switch": true,
		"mode":              "flexible",
	}, http.StatusOK)

	statusResp := h.GetCompetitionStatus()
	require.NotNil(t, statusResp.JSON200)
	require.Equal(t, "paused", *statusResp.JSON200.Status)

	h.SubmitFlag(tokenUser, challengeID, "FLAG{comp}", http.StatusForbidden)

	h.PutAdminCompetitionExpectStatus(tokenAdmin, map[string]any{
		"name":              "Comp Name",
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	}, http.StatusOK)

	h.SubmitFlag(tokenUser, challengeID, "FLAG{comp}", http.StatusOK)
}

// GET /admin/competition: admin gets full competition config (name, start_time, end_time, freeze_time, etc.).
func TestCompetition_Admin_Get(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _ = h.SetupCompetition("admin_get_f")
	_, _, tokenUser := h.RegisterUserAndLogin("nonadmin_comp")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)
	h.GetAdminCompetitionExpectStatus(tokenUser, http.StatusForbidden)
}

// PUT /admin/competition: non-admin gets 403 Forbidden.
func TestCompetition_Admin_Put_Forbidden(t *testing.T) {
	t.Helper()
	t.Parallel()
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

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

func TestCompetition_PauseUnpause_ShiftsFreezeAndEndTime_StatusActive(t *testing.T) {
	t.Helper()
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("admin_pause_unpause")
	now := time.Now().UTC()
	freezeIn := now.Add(1 * time.Minute)
	endIn := now.Add(2 * time.Hour)
	setCompetitionTimes(now.Add(-1*time.Hour), endIn, &freezeIn)
	setCompetitionMode("flexible")
	time.Sleep(3 * time.Second)

	adminBefore := h.GetAdminCompetition(tokenAdmin)
	require.NotNil(t, adminBefore.JSON200)
	require.NotNil(t, adminBefore.JSON200.FreezeTime)
	require.NotNil(t, adminBefore.JSON200.EndTime)
	freezeBefore, err := time.Parse(time.RFC3339, *adminBefore.JSON200.FreezeTime)
	require.NoError(t, err)
	endBefore, err := time.Parse(time.RFC3339, *adminBefore.JSON200.EndTime)
	require.NoError(t, err)

	h.PutAdminCompetitionExpectStatus(tokenAdmin, map[string]any{
		"name":              "Pause Unpause",
		"is_paused":         true,
		"allow_team_switch": true,
		"mode":              "flexible",
	}, http.StatusOK)

	pauseDuration := 5 * time.Second
	time.Sleep(pauseDuration)

	h.PutAdminCompetitionExpectStatus(tokenAdmin, map[string]any{
		"name":              "Pause Unpause",
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	}, http.StatusOK)

	statusResp := h.GetCompetitionStatus()
	require.NotNil(t, statusResp.JSON200)
	require.Equal(t, "active", *statusResp.JSON200.Status)

	adminAfter := h.GetAdminCompetition(tokenAdmin)
	require.NotNil(t, adminAfter.JSON200)
	require.NotNil(t, adminAfter.JSON200.FreezeTime)
	require.NotNil(t, adminAfter.JSON200.EndTime)
	freezeAfter, err := time.Parse(time.RFC3339, *adminAfter.JSON200.FreezeTime)
	require.NoError(t, err)
	endAfter, err := time.Parse(time.RFC3339, *adminAfter.JSON200.EndTime)
	require.NoError(t, err)

	require.True(t, freezeAfter.After(freezeBefore), "FreezeTime should shift forward after unpause")
	require.True(t, endAfter.After(endBefore), "EndTime should shift forward after unpause")
	tolerance := 2 * time.Second
	require.InDelta(t, pauseDuration.Seconds(), freezeAfter.Sub(freezeBefore).Seconds(), tolerance.Seconds())
	require.InDelta(t, pauseDuration.Seconds(), endAfter.Sub(endBefore).Seconds(), tolerance.Seconds())
}

func TestCompetition_ForceEnd_AdminSetsEndTimeInPast_StatusEnded(t *testing.T) {
	t.Helper()
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("admin_force_end")
	now := time.Now().UTC()
	setCompetitionTimes(now.Add(-1*time.Hour), now.Add(24*time.Hour), nil)
	setCompetitionMode("flexible")
	time.Sleep(2 * time.Second)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Force End Chall", "description": "x", "flag": "flag{force_end}",
		"points": 100, "category": "misc", "is_hidden": false,
	})
	_, _, tokenUser := h.RegisterUserAndLogin("user_force_end")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	forceEnd := now.Add(-1 * time.Minute)
	forceFreeze := now.Add(-2 * time.Minute)
	h.PutAdminCompetitionExpectStatus(tokenAdmin, map[string]any{
		"name":              "CTF",
		"start_time":        now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":          forceEnd.Format(time.RFC3339),
		"freeze_time":       forceFreeze.Format(time.RFC3339),
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	}, http.StatusOK)

	statusResp := h.GetCompetitionStatus()
	require.NotNil(t, statusResp.JSON200)
	require.Equal(t, "ended", *statusResp.JSON200.Status)
	h.SubmitFlag(tokenUser, challengeID, "flag{force_end}", http.StatusForbidden)
}

func TestCompetition_UnpauseAfterEndTimePassed_StatusEnded(t *testing.T) {
	t.Helper()
	t.Cleanup(resetCompetitionToActive)
	h := helper.NewE2EHelper(t, nil, TestPool, TestRedis, GetTestBaseURL())

	_, _, tokenAdmin := h.RegisterAdmin("admin_unpause_ended")
	now := time.Now().UTC()
	setCompetitionTimes(now.Add(-2*time.Hour), now.Add(24*time.Hour), nil)
	setCompetitionMode("flexible")
	time.Sleep(2 * time.Second)

	challengeID := h.CreateChallenge(tokenAdmin, map[string]any{
		"title": "Unpause Ended Chall", "description": "x", "flag": "flag{unpause_ended}",
		"points": 100, "category": "misc", "is_hidden": false,
	})
	_, _, tokenUser := h.RegisterUserAndLogin("user_unpause_ended")
	h.CreateSoloTeam(tokenUser, http.StatusCreated)

	startPast := now.Add(-2 * time.Hour)
	endPast := now.Add(-1 * time.Hour)
	pausedAt := now.Add(-30 * time.Minute)
	ctx := context.Background()
	_, err := TestPool.Exec(ctx, `UPDATE competition SET start_time = $1, end_time = $2, is_paused = TRUE, paused_at = $3, updated_at = now() WHERE id = 1`,
		startPast, endPast, pausedAt)
	require.NoError(t, err)
	if TestRedis != nil {
		require.NoError(t, TestRedis.Del(ctx, "competition").Err())
	}
	time.Sleep(6 * time.Second)

	require.True(t, h.PollCompetitionStatus("paused", 10*time.Second), "competition should be paused (end_time passed)")
	statusBefore := h.GetCompetitionStatus()
	require.NotNil(t, statusBefore.JSON200)
	require.Equal(t, "paused", *statusBefore.JSON200.Status)
	h.SubmitFlag(tokenUser, challengeID, "flag{unpause_ended}", http.StatusForbidden)

	h.PutAdminCompetitionExpectStatus(tokenAdmin, map[string]any{
		"name":              "CTF",
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	}, http.StatusOK)

	statusAfter := h.GetCompetitionStatus()
	require.NotNil(t, statusAfter.JSON200)
	require.Equal(t, "ended", *statusAfter.JSON200.Status)
	h.SubmitFlag(tokenUser, challengeID, "flag{unpause_ended}", http.StatusForbidden)
}
