package helper

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func (h *E2EHelper) GetStatisticsGeneral(token string) *openapi.GetStatisticsGeneralResponse {
	h.t.Helper()
	resp, err := h.client.GetStatisticsGeneralWithResponse(context.Background(), nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics general")

	return resp
}

func (h *E2EHelper) GetStatisticsChallenges(token string) *openapi.GetStatisticsChallengesResponse {
	h.t.Helper()
	resp, err := h.client.GetStatisticsChallengesWithResponse(context.Background(), nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics challenges")

	return resp
}

func (h *E2EHelper) GetStatisticsChallengesId(token, id string) *openapi.GetStatisticsChallengesIDResponse {
	h.t.Helper()

	parsed, err := uuid.Parse(id)
	require.NoError(h.t, err)
	resp, err := h.client.GetStatisticsChallengesIDWithResponse(context.Background(), parsed, nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics challenge detail")

	return resp
}

func (h *E2EHelper) GetStatisticsChallengesIdExpectStatus(token, id string, expectStatus int) {
	h.t.Helper()

	parsed, err := uuid.Parse(id)
	require.NoError(h.t, err)
	resp, err := h.client.GetStatisticsChallengesIDWithResponse(context.Background(), parsed, nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "statistics challenge detail")
}

func (h *E2EHelper) GetStatisticsScoreboard(token string, limit int) *openapi.GetStatisticsScoreboardResponse {
	h.t.Helper()

	params := &openapi.GetStatisticsScoreboardParams{Limit: &limit}
	resp, err := h.client.GetStatisticsScoreboardWithResponse(context.Background(), params, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, http.StatusOK, resp.StatusCode(), resp.Body, "statistics scoreboard")

	return resp
}
