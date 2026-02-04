package helper

import (
	"context"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetTags(expectStatus int) *openapi.GetTagsResponse {
	h.t.Helper()
	resp, err := h.client.GetTagsWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get tags")
	return resp
}

func (h *E2EHelper) CreateTag(token, name, color string, expectStatus int) *openapi.PostAdminTagsResponse {
	h.t.Helper()
	clr := color
	resp, err := h.client.PostAdminTagsWithResponse(context.Background(), openapi.PostAdminTagsJSONRequestBody{
		Name:  name,
		Color: &clr,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create tag")
	return resp
}

func (h *E2EHelper) UpdateTag(token, id, name, color string, expectStatus int) *openapi.PutAdminTagsIDResponse {
	h.t.Helper()
	clr := color
	resp, err := h.client.PutAdminTagsIDWithResponse(context.Background(), id, openapi.PutAdminTagsIDJSONRequestBody{
		Name:  name,
		Color: &clr,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update tag")
	return resp
}

func (h *E2EHelper) DeleteTag(token, id string, expectStatus int) *openapi.DeleteAdminTagsIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteAdminTagsIDWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete tag")
	return resp
}
