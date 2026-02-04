package helper

import (
	"context"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetAdminConfigs(token string, expectStatus int) *openapi.GetAdminConfigsResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminConfigsWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin configs list")
	return resp
}

func (h *E2EHelper) PutAdminConfig(token, key, value, valueType, description string, expectStatus int) *openapi.PutAdminConfigsKeyResponse {
	h.t.Helper()
	vt := openapi.RequestSetConfigRequestValueType(valueType)
	desc := description
	resp, err := h.client.PutAdminConfigsKeyWithResponse(context.Background(), key, openapi.PutAdminConfigsKeyJSONRequestBody{
		Value:       value,
		ValueType:   &vt,
		Description: &desc,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin config upsert")
	return resp
}

func (h *E2EHelper) GetAdminConfigKey(token, key string, expectStatus int) *openapi.GetAdminConfigsKeyResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminConfigsKeyWithResponse(context.Background(), key, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get admin config key")
	return resp
}

func (h *E2EHelper) DeleteAdminConfig(token, key string, expectStatus int) *openapi.DeleteAdminConfigsKeyResponse {
	h.t.Helper()
	resp, err := h.client.DeleteAdminConfigsKeyWithResponse(context.Background(), key, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete admin config")
	return resp
}
