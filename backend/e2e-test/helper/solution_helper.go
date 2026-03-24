package helper

import (
	"context"
	"net/http"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func (h *E2EHelper) AdminUpsertSolution(token, challengeID, content string, expectStatus int) *openapi.PostAdminChallengesChallengeIDSolutionResponse {
	h.t.Helper()
	resp, err := h.client.PostAdminChallengesChallengeIDSolutionWithResponse(
		context.Background(),
		challengeID,
		openapi.PostAdminChallengesChallengeIDSolutionJSONRequestBody{Content: content},
		WithBearerToken(token),
	)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin upsert solution")
	return resp
}

func (h *E2EHelper) AdminDeleteSolution(token, challengeID string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.DeleteAdminChallengesChallengeIDSolutionWithResponse(
		context.Background(),
		challengeID,
		WithBearerToken(token),
	)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin delete solution")
}

func (h *E2EHelper) GetSolution(token, challengeID string, expectStatus int) *openapi.GetChallengesChallengeIDSolutionResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesChallengeIDSolutionWithResponse(
		context.Background(),
		challengeID,
		WithBearerToken(token),
	)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get solution")
	return resp
}

func (h *E2EHelper) ListSolutions(token string, expectStatus int) *openapi.GetChallengesSolutionsResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesSolutionsWithResponse(
		context.Background(),
		WithBearerToken(token),
	)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "list solutions")
	return resp
}

func (h *E2EHelper) GetSolutionExpectOneOf(token, challengeID string, allowedStatuses []int) *openapi.GetChallengesChallengeIDSolutionResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesChallengeIDSolutionWithResponse(
		context.Background(),
		challengeID,
		WithBearerToken(token),
	)
	require.NoError(h.t, err)
	require.Contains(h.t, allowedStatuses, resp.StatusCode(), "get solution: status %d not in %v body=%s", resp.StatusCode(), allowedStatuses, string(resp.Body))
	return resp
}

func (h *E2EHelper) ListSolutionsExpectOneOf(token string, allowedStatuses []int) *openapi.GetChallengesSolutionsResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesSolutionsWithResponse(
		context.Background(),
		WithBearerToken(token),
	)
	require.NoError(h.t, err)
	require.Contains(h.t, allowedStatuses, resp.StatusCode(), "list solutions: status %d not in %v body=%s", resp.StatusCode(), allowedStatuses, string(resp.Body))
	return resp
}

func (h *E2EHelper) EnableWriteups(tokenAdmin string) {
	h.t.Helper()
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		resp := h.PutAdminSettingsExpectOneOf(tokenAdmin, map[string]any{"writeup_enabled": true}, []int{http.StatusOK, http.StatusForbidden, http.StatusConflict})
		if resp.StatusCode() != http.StatusConflict {
			return
		}
		if i < maxRetries-1 {
			time.Sleep(50 * time.Millisecond)
		} else {
			require.Fail(h.t, "put admin settings: still 409 after retries")
		}
	}
}

func (h *E2EHelper) DisableWriteups(tokenAdmin string) int {
	h.t.Helper()
	resp := h.PutAdminSettingsExpectOneOf(tokenAdmin, map[string]any{"writeup_enabled": false}, []int{http.StatusOK, http.StatusForbidden, http.StatusConflict})
	return resp.StatusCode()
}
