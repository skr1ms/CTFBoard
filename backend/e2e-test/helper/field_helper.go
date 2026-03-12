package helper

import (
	"context"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func (h *E2EHelper) GetFields(entityType string, expectStatus int) *openapi.GetFieldsResponse {
	h.t.Helper()
	et := openapi.GetFieldsParamsEntityType(entityType)
	params := openapi.GetFieldsParams{EntityType: &et}
	resp, err := h.client.GetFieldsWithResponse(context.Background(), &params)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get fields")
	return resp
}

func (h *E2EHelper) CreateField(token, name, fieldType, entityType string, required bool, expectStatus int) *openapi.PostAdminFieldsResponse {
	h.t.Helper()
	ft := openapi.CreateFieldRequestFieldType(fieldType)
	et := openapi.CreateFieldRequestEntityType(entityType)
	resp, err := h.client.PostAdminFieldsWithResponse(context.Background(), openapi.PostAdminFieldsJSONRequestBody{
		Name:       name,
		FieldType:  ft,
		EntityType: et,
		Required:   &required,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "create field")
	return resp
}

func (h *E2EHelper) DeleteField(token, id string, expectStatus int) *openapi.DeleteAdminFieldsIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteAdminFieldsIDWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete field")
	return resp
}

func (h *E2EHelper) UpdateField(token, id, name, fieldType string, required bool, expectStatus int) *openapi.PutAdminFieldsIDResponse {
	h.t.Helper()
	ft := openapi.UpdateFieldRequestFieldType(fieldType)
	resp, err := h.client.PutAdminFieldsIDWithResponse(context.Background(), id, openapi.PutAdminFieldsIDJSONRequestBody{
		Name:      name,
		FieldType: ft,
		Required:  &required,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "update field")
	return resp
}
