package helper

import (
	"context"
	"time"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetRatings(page, perPage, expectStatus int) *openapi.GetRatingsResponse {
	h.t.Helper()
	params := &openapi.GetRatingsParams{Page: &page, PerPage: &perPage}
	resp, err := h.client.GetRatingsWithResponse(context.Background(), params)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get ratings")
	return resp
}

func (h *E2EHelper) GetTeamRatings(teamID string, expectStatus int) *openapi.GetRatingsTeamIDResponse {
	h.t.Helper()
	resp, err := h.client.GetRatingsTeamIDWithResponse(context.Background(), teamID)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get team ratings")
	return resp
}

func (h *E2EHelper) GetAdminCTFEvents(token string, expectStatus int) *openapi.GetAdminCtfEventsResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminCtfEventsWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin ctf-events list")
	return resp
}

func (h *E2EHelper) CreateCTFEvent(token, name string, start, end time.Time, weight float32, expectStatus int) *openapi.PostAdminCtfEventsResponse {
	h.t.Helper()
	w := weight
	resp, err := h.client.PostAdminCtfEventsWithResponse(context.Background(), openapi.PostAdminCtfEventsJSONRequestBody{
		Name:      name,
		StartTime: start,
		EndTime:   end,
		Weight:    &w,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin ctf-events create")
	return resp
}

func (h *E2EHelper) FinalizeCTFEvent(token, id string, expectStatus int) *openapi.PostAdminCtfEventsIDFinalizeResponse {
	h.t.Helper()
	resp, err := h.client.PostAdminCtfEventsIDFinalizeWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin ctf-events finalize")
	return resp
}
