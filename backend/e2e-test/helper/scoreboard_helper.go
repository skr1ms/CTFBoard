package helper

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetScoreboard() *openapi.GetScoreboardResponse {
	h.t.Helper()
	resp, err := h.client.GetScoreboardWithResponse(context.Background(), &openapi.GetScoreboardParams{})
	require.NoError(h.t, err)
	return resp
}

func (h *E2EHelper) AssertTeamScore(teamName string, expectedPoints int) {
	h.t.Helper()
	resp := h.GetScoreboard()
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "scoreboard")
	require.NotNil(h.t, resp.JSON200)
	for _, entry := range *resp.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == teamName {
			require.NotNil(h.t, entry.Points, "team %s has nil points", teamName)
			require.Equal(h.t, expectedPoints, *entry.Points, "team %s points", teamName)
			return
		}
	}
	h.t.Fatalf("Team %s not found in scoreboard", teamName)
}

func (h *E2EHelper) AssertTeamScoreAtLeast(teamName string, minPoints int) {
	h.t.Helper()
	resp := h.GetScoreboard()
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "scoreboard")
	require.NotNil(h.t, resp.JSON200)
	for _, entry := range *resp.JSON200 {
		if entry.TeamName != nil && *entry.TeamName == teamName {
			require.NotNil(h.t, entry.Points, "team %s has nil points", teamName)
			require.GreaterOrEqual(h.t, *entry.Points, minPoints, "team %s points", teamName)
			return
		}
	}
	h.t.Fatalf("Team %s not found in scoreboard", teamName)
}

func (h *E2EHelper) GetScoreboardWithBracket(bracketID string) *openapi.GetScoreboardResponse {
	h.t.Helper()
	bid, err := uuid.Parse(bracketID)
	require.NoError(h.t, err)
	params := &openapi.GetScoreboardParams{Bracket: &bid}
	resp, err := h.client.GetScoreboardWithResponse(context.Background(), params)
	require.NoError(h.t, err)
	return resp
}

func (h *E2EHelper) GetScoreboardGraph(top int) *openapi.GetScoreboardGraphResponse {
	h.t.Helper()
	params := (*openapi.GetScoreboardGraphParams)(nil)
	if top > 0 {
		params = &openapi.GetScoreboardGraphParams{Top: &top}
	}
	resp, err := h.client.GetScoreboardGraphWithResponse(context.Background(), params)
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "scoreboard graph")
	return resp
}
