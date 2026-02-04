package helper

import (
	"context"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetBrackets(expectStatus int) *openapi.GetBracketsResponse {
	h.t.Helper()
	resp, err := h.client.GetBracketsWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get brackets")
	return resp
}

func (h *E2EHelper) CreateBracket(token, name, desc string, isDefault bool, expectStatus int) *openapi.PostAdminBracketsResponse {
	h.t.Helper()
	d := desc
	resp, err := h.client.PostAdminBracketsWithResponse(context.Background(), openapi.PostAdminBracketsJSONRequestBody{
		Name:        name,
		Description: &d,
		IsDefault:   &isDefault,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create bracket")
	return resp
}

func (h *E2EHelper) SetTeamBracket(token, teamID, bracketID string, expectStatus int) *openapi.PatchAdminTeamsIDBracketResponse {
	h.t.Helper()
	bid, err := uuid.Parse(bracketID)
	require.NoError(h.t, err)
	resp, err := h.client.PatchAdminTeamsIDBracketWithResponse(context.Background(), teamID, openapi.PatchAdminTeamsIDBracketJSONRequestBody{
		BracketID: &bid,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "set team bracket")
	return resp
}

func (h *E2EHelper) UpdateBracket(token, id, name, desc string, isDefault bool, expectStatus int) *openapi.PutAdminBracketsIDResponse {
	h.t.Helper()
	d := desc
	resp, err := h.client.PutAdminBracketsIDWithResponse(context.Background(), id, openapi.PutAdminBracketsIDJSONRequestBody{
		Name:        name,
		Description: &d,
		IsDefault:   &isDefault,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update bracket")
	return resp
}

func (h *E2EHelper) DeleteBracket(token, id string, expectStatus int) *openapi.DeleteAdminBracketsIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteAdminBracketsIDWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete bracket")
	return resp
}
