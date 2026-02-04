package helper

import (
	"context"
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) CreateHint(token, challengeID, content string, cost int) string {
	h.t.Helper()
	return h.CreateHintExpectStatus(token, challengeID, content, cost, http.StatusCreated)
}

func (h *E2EHelper) CreateHintExpectStatus(token, challengeID, content string, cost, expectStatus int) string {
	h.t.Helper()
	orderIndex := 1
	resp, err := h.client.PostAdminChallengesChallengeIDHintsWithResponse(context.Background(), challengeID, openapi.PostAdminChallengesChallengeIDHintsJSONRequestBody{
		Content:    content,
		Cost:       &cost,
		OrderIndex: &orderIndex,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create hint")
	if resp.JSON201 != nil && resp.JSON201.ID != nil {
		return *resp.JSON201.ID
	}
	return ""
}

func (h *E2EHelper) UnlockHint(token, challengeID, hintID string, expectStatus int) *openapi.PostChallengesChallengeIDHintsHintIDUnlockResponse {
	h.t.Helper()
	resp, err := h.client.PostChallengesChallengeIDHintsHintIDUnlockWithResponse(context.Background(), challengeID, hintID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "unlock hint")
	return resp
}

func (h *E2EHelper) GetChallengesChallengeIDHintsExpectStatus(token, challengeID string, expectStatus int) *openapi.GetChallengesChallengeIDHintsResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesChallengeIDHintsWithResponse(context.Background(), challengeID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get hints")
	return resp
}

func (h *E2EHelper) GetHintFromList(token, challengeID, hintID string) *openapi.ResponseHintResponse {
	h.t.Helper()
	resp := h.GetChallengesChallengeIDHintsExpectStatus(token, challengeID, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	for i := range *resp.JSON200 {
		c := &(*resp.JSON200)[i]
		if c.ID != nil && *c.ID == hintID {
			return c
		}
	}
	h.t.Fatalf("Hint %s not found in challenge %s list", hintID, challengeID)
	return nil
}

func (h *E2EHelper) UpdateHint(token, hintID, content string, cost, expectStatus int) *openapi.PutAdminHintsIDResponse {
	h.t.Helper()
	resp, err := h.client.PutAdminHintsIDWithResponse(context.Background(), hintID, openapi.PutAdminHintsIDJSONRequestBody{
		Content: content,
		Cost:    &cost,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update hint")
	return resp
}

func (h *E2EHelper) DeleteHint(token, hintID string, expectStatus int) *openapi.DeleteAdminHintsIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteAdminHintsIDWithResponse(context.Background(), hintID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete hint")
	return resp
}
