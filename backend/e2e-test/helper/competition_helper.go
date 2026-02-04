package helper

import (
	"context"
	"net/http"
	"time"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func parseTimeField(data map[string]any, key string) *time.Time {
	switch v := data[key].(type) {
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			return &t
		}
	case time.Time:
		return &v
	}
	return nil
}

func buildCompetitionBody(data map[string]any) openapi.PutAdminCompetitionJSONRequestBody {
	body := openapi.PutAdminCompetitionJSONRequestBody{Name: getStr(data, "name", "Test CTF")}
	if v, ok := data["is_public"].(bool); ok {
		body.IsPublic = &v
	}
	if v, ok := data["flag_regex"].(string); ok {
		body.FlagRegex = &v
	}
	if v, ok := data["is_paused"].(bool); ok {
		body.IsPaused = &v
	}
	if v, ok := data["allow_team_switch"].(bool); ok {
		body.AllowTeamSwitch = &v
	}
	if v, ok := data["mode"].(string); ok {
		body.Mode = &v
	}
	body.StartTime = parseTimeField(data, "start_time")
	body.EndTime = parseTimeField(data, "end_time")
	body.FreezeTime = parseTimeField(data, "freeze_time")
	return body
}

func (h *E2EHelper) GetCompetitionStatus() *openapi.GetCompetitionStatusResponse {
	h.t.Helper()
	resp, err := h.client.GetCompetitionStatusWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "competition status")
	return resp
}

func (h *E2EHelper) UpdateCompetition(token string, data map[string]any) {
	h.t.Helper()
	h.PutAdminCompetitionExpectStatus(token, data, http.StatusOK)
}

func (h *E2EHelper) PutAdminCompetitionExpectStatus(token string, data map[string]any, expectStatus int) {
	h.t.Helper()
	body := buildCompetitionBody(data)
	resp, err := h.client.PutAdminCompetitionWithResponse(context.Background(), body, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update competition")
}

func (h *E2EHelper) SetCompetitionRegex(token, regex string) {
	h.t.Helper()
	now := time.Now().UTC()
	h.UpdateCompetition(token, map[string]any{
		"name":              "Test CTF",
		"is_public":         true,
		"flag_regex":        regex,
		"start_time":        now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":          now.Add(24 * time.Hour).Format(time.RFC3339),
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	})
}

func (h *E2EHelper) StartCompetition(adminToken string) {
	h.t.Helper()
	now := time.Now().UTC()
	h.UpdateCompetition(adminToken, map[string]any{
		"name":              "Test CTF",
		"start_time":        now.Add(-1 * time.Hour).Format(time.RFC3339),
		"end_time":          now.Add(24 * time.Hour).Format(time.RFC3339),
		"is_paused":         false,
		"allow_team_switch": true,
		"mode":              "flexible",
	})
}

func (h *E2EHelper) GetAdminCompetition(token string) *openapi.GetAdminCompetitionResponse {
	h.t.Helper()
	return h.GetAdminCompetitionExpectStatus(token, http.StatusOK)
}

func (h *E2EHelper) GetAdminCompetitionExpectStatus(token string, expectStatus int) *openapi.GetAdminCompetitionResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminCompetitionWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin competition")
	return resp
}
