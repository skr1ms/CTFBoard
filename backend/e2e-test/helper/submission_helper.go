package helper

import (
	"context"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetAdminSubmissions(token string, page, perPage, expectStatus int) *openapi.GetAdminSubmissionsResponse {
	h.t.Helper()
	params := &openapi.GetAdminSubmissionsParams{Page: &page, PerPage: &perPage}
	resp, err := h.client.GetAdminSubmissionsWithResponse(context.Background(), params, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin submissions")
	return resp
}

func (h *E2EHelper) GetAdminSubmissionsByChallenge(token, challengeID string, page, perPage, expectStatus int) *openapi.GetAdminSubmissionsChallengeChallengeIDResponse {
	h.t.Helper()
	params := &openapi.GetAdminSubmissionsChallengeChallengeIDParams{Page: &page, PerPage: &perPage}
	resp, err := h.client.GetAdminSubmissionsChallengeChallengeIDWithResponse(context.Background(), challengeID, params, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin submissions by challenge")
	return resp
}

func (h *E2EHelper) GetAdminSubmissionStatsByChallenge(token, challengeID string, expectStatus int) *openapi.GetAdminSubmissionsChallengeChallengeIDStatsResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminSubmissionsChallengeChallengeIDStatsWithResponse(context.Background(), challengeID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin submission stats by challenge")
	return resp
}

func (h *E2EHelper) GetAdminSubmissionsByUser(token, userID string, page, perPage, expectStatus int) *openapi.GetAdminSubmissionsUserUserIDResponse {
	h.t.Helper()
	params := &openapi.GetAdminSubmissionsUserUserIDParams{Page: &page, PerPage: &perPage}
	resp, err := h.client.GetAdminSubmissionsUserUserIDWithResponse(context.Background(), userID, params, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin submissions by user")
	return resp
}

func (h *E2EHelper) GetAdminSubmissionsByTeam(token, teamID string, page, perPage, expectStatus int) *openapi.GetAdminSubmissionsTeamTeamIDResponse {
	h.t.Helper()
	params := &openapi.GetAdminSubmissionsTeamTeamIDParams{Page: &page, PerPage: &perPage}
	resp, err := h.client.GetAdminSubmissionsTeamTeamIDWithResponse(context.Background(), teamID, params, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin submissions by team")
	return resp
}
