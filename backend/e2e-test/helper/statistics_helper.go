package helper

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetStatisticsGeneral() *openapi.GetStatisticsGeneralResponse {
	h.t.Helper()
	resp, err := h.client.GetStatisticsGeneralWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics general")
	return resp
}

func (h *E2EHelper) GetStatisticsChallenges() *openapi.GetStatisticsChallengesResponse {
	h.t.Helper()
	resp, err := h.client.GetStatisticsChallengesWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics challenges")
	return resp
}

func (h *E2EHelper) GetStatisticsChallengesId(id string) *openapi.GetStatisticsChallengesIDResponse {
	h.t.Helper()
	parsed, err := uuid.Parse(id)
	require.NoError(h.t, err)
	resp, err := h.client.GetStatisticsChallengesIDWithResponse(context.Background(), parsed)
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics challenge detail")
	return resp
}

func (h *E2EHelper) GetStatisticsChallengesIdExpectStatus(id string, expectStatus int) {
	h.t.Helper()
	parsed, err := uuid.Parse(id)
	require.NoError(h.t, err)
	resp, err := h.client.GetStatisticsChallengesIDWithResponse(context.Background(), parsed)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "statistics challenge detail")
}

func (h *E2EHelper) GetStatisticsScoreboard(limit int) *openapi.GetStatisticsScoreboardResponse {
	h.t.Helper()
	resp, err := h.client.GetStatisticsScoreboardWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics scoreboard")
	return resp
}
