package helper

import (
	"context"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func (h *E2EHelper) CreateAppeal(ctx context.Context, token, message string, expectStatus int) *openapi.PostAppealsResponse {
	h.t.Helper()

	resp, err := h.client.PostAppealsWithResponse(ctx, openapi.PostAppealsJSONRequestBody{
		Message: message,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create appeal")

	return resp
}

func (h *E2EHelper) GetMyAppeals(ctx context.Context, token string, expectStatus int) *openapi.GetAppealsMeResponse {
	h.t.Helper()

	resp, err := h.client.GetAppealsMeWithResponse(ctx, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get my appeals")

	return resp
}

func (h *E2EHelper) AdminListAppeals(ctx context.Context, adminToken string, decision *openapi.GetAdminAppealsParamsDecision, page, perPage *int, expectStatus int) *openapi.GetAdminAppealsResponse {
	h.t.Helper()

	params := &openapi.GetAdminAppealsParams{
		Decision: decision,
		Page:     page,
		PerPage:  perPage,
	}
	resp, err := h.client.GetAdminAppealsWithResponse(ctx, params, WithBearerToken(adminToken))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin list appeals")

	return resp
}

func (h *E2EHelper) AdminReviewAppeal(ctx context.Context, adminToken, appealID string, decision openapi.ReviewAppealRequestDecision, adminResponse *string, expectStatus int) *openapi.PatchAdminAppealsIDResponse {
	h.t.Helper()

	resp, err := h.client.PatchAdminAppealsIDWithResponse(ctx, appealID, openapi.PatchAdminAppealsIDJSONRequestBody{
		Decision:      decision,
		AdminResponse: adminResponse,
	}, WithBearerToken(adminToken))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin review appeal")

	return resp
}
