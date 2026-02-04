package helper

import (
	"context"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetPages(expectStatus int) *openapi.GetPagesResponse {
	h.t.Helper()
	resp, err := h.client.GetPagesWithResponse(context.Background())
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get pages")
	return resp
}

func (h *E2EHelper) GetPageBySlug(slug string, expectStatus int) *openapi.GetPagesSlugResponse {
	h.t.Helper()
	resp, err := h.client.GetPagesSlugWithResponse(context.Background(), slug)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get page by slug")
	return resp
}

func (h *E2EHelper) CreatePage(token, title, slug, content string, isDraft bool, orderIndex, expectStatus int) *openapi.PostAdminPagesResponse {
	h.t.Helper()
	cnt := content
	resp, err := h.client.PostAdminPagesWithResponse(context.Background(), openapi.PostAdminPagesJSONRequestBody{
		Title:      title,
		Slug:       slug,
		Content:    &cnt,
		IsDraft:    &isDraft,
		OrderIndex: &orderIndex,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create page")
	return resp
}

func (h *E2EHelper) UpdatePage(token, id, title, slug, content string, isDraft bool, orderIndex, expectStatus int) *openapi.PutAdminPagesIDResponse {
	h.t.Helper()
	cnt := content
	resp, err := h.client.PutAdminPagesIDWithResponse(context.Background(), id, openapi.PutAdminPagesIDJSONRequestBody{
		Title:      title,
		Slug:       slug,
		Content:    &cnt,
		IsDraft:    &isDraft,
		OrderIndex: &orderIndex,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update page")
	return resp
}

func (h *E2EHelper) DeletePage(token, id string, expectStatus int) *openapi.DeleteAdminPagesIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteAdminPagesIDWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete page")
	return resp
}
