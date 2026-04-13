package helper

import (
	"context"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func (h *E2EHelper) PutRating(token, challengeID string, value int, review *string, expectStatus int) *openapi.PutChallengesChallengeIDRatingResponse {
	h.t.Helper()

	body := openapi.PutChallengesChallengeIDRatingJSONRequestBody{
		Value:  value,
		Review: review,
	}
	resp, err := h.client.PutChallengesChallengeIDRatingWithResponse(context.Background(), challengeID, body, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "put rating")

	return resp
}

func (h *E2EHelper) GetRatings(token, challengeID string, expectStatus int) *openapi.GetChallengesChallengeIDRatingsResponse {
	h.t.Helper()

	resp, err := h.client.GetChallengesChallengeIDRatingsWithResponse(context.Background(), challengeID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get ratings")

	return resp
}
