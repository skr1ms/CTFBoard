package helper

import (
	"context"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetChallengeComments(token, challengeID string, expectStatus int) *openapi.GetChallengesChallengeIDCommentsResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesChallengeIDCommentsWithResponse(context.Background(), challengeID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get comments")
	return resp
}

func (h *E2EHelper) CreateComment(token, challengeID, content string, expectStatus int) *openapi.PostChallengesChallengeIDCommentsResponse {
	h.t.Helper()
	resp, err := h.client.PostChallengesChallengeIDCommentsWithResponse(context.Background(), challengeID, openapi.PostChallengesChallengeIDCommentsJSONRequestBody{
		Content: content,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create comment")
	return resp
}

func (h *E2EHelper) DeleteComment(token, id string, expectStatus int) *openapi.DeleteCommentsIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteCommentsIDWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete comment")
	return resp
}
