package helper

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) AdminExport(token string, includeUsers, includeAwards bool) *openapi.GetAdminExportResponse {
	h.t.Helper()
	return h.AdminExportExpectStatus(token, includeUsers, includeAwards, http.StatusOK)
}

func (h *E2EHelper) AdminExportExpectStatus(token string, includeUsers, includeAwards bool, expectStatus int) *openapi.GetAdminExportResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminExportWithResponse(context.Background(), &openapi.GetAdminExportParams{
		IncludeUsers:  &includeUsers,
		IncludeAwards: &includeAwards,
	}, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin export")
	return resp
}

func (h *E2EHelper) AdminExportZip(token string) *openapi.GetAdminExportZipResponse {
	h.t.Helper()
	return h.AdminExportZipExpectStatus(token, http.StatusOK)
}

func (h *E2EHelper) AdminExportZipExpectStatus(token string, expectStatus int) *openapi.GetAdminExportZipResponse {
	h.t.Helper()
	resp, err := h.client.GetAdminExportZipWithResponse(context.Background(), nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin export zip")
	return resp
}

func (h *E2EHelper) AdminImport(token string, fileContent []byte, fileName, conflictMode string, expectStatus int) *openapi.PostAdminImportResponse {
	h.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", fileName)
	require.NoError(h.t, err)
	_, err = part.Write(fileContent)
	require.NoError(h.t, err)
	if conflictMode != "" {
		require.NoError(h.t, w.WriteField("conflict_mode", conflictMode))
	}
	contentType := w.FormDataContentType()
	require.NoError(h.t, w.Close())
	resp, err := h.client.PostAdminImportWithBodyWithResponse(context.Background(), contentType, &buf, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin import")
	return resp
}
