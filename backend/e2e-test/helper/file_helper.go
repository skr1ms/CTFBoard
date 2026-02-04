package helper

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) DeleteChallengeFile(token, fileID string, expectStatus int) *openapi.DeleteAdminFilesIDResponse {
	h.t.Helper()
	resp, err := h.client.DeleteAdminFilesIDWithResponse(context.Background(), fileID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete file")
	return resp
}

func (h *E2EHelper) UploadChallengeFile(token, challengeID, fileName, content string) *openapi.PostAdminChallengesChallengeIDFilesResponse {
	h.t.Helper()
	return h.UploadChallengeFileExpectStatus(token, challengeID, fileName, content, http.StatusCreated)
}

func (h *E2EHelper) UploadChallengeFileExpectStatus(token, challengeID, fileName, content string, expectStatus int) *openapi.PostAdminChallengesChallengeIDFilesResponse {
	h.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", fileName)
	require.NoError(h.t, err)
	_, err = part.Write([]byte(content))
	require.NoError(h.t, err)
	contentType := w.FormDataContentType()
	require.NoError(h.t, w.Close())
	resp, err := h.client.PostAdminChallengesChallengeIDFilesWithBodyWithResponse(context.Background(), challengeID, contentType, &buf, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "upload challenge file")
	return resp
}

func (h *E2EHelper) GetChallengeFiles(token, challengeID string) *openapi.GetChallengesChallengeIDFilesResponse {
	h.t.Helper()
	return h.GetChallengeFilesExpectStatus(token, challengeID, http.StatusOK)
}

func (h *E2EHelper) GetChallengeFilesExpectStatus(token, challengeID string, expectStatus int) *openapi.GetChallengesChallengeIDFilesResponse {
	h.t.Helper()
	resp, err := h.client.GetChallengesChallengeIDFilesWithResponse(context.Background(), challengeID, nil, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get challenge files")
	return resp
}

func (h *E2EHelper) GetFileDownloadURL(token, fileID string) string {
	h.t.Helper()
	resp := h.GetFilesIDDownloadExpectStatus(token, fileID, http.StatusOK)
	require.NotNil(h.t, resp.JSON200)
	url, ok := (*resp.JSON200)["url"]
	require.True(h.t, ok, "url key in response")
	return url
}

func (h *E2EHelper) GetFilesIDDownloadExpectStatus(token, fileID string, expectStatus int) *openapi.GetFilesIDDownloadResponse {
	h.t.Helper()
	resp, err := h.client.GetFilesIDDownloadWithResponse(context.Background(), fileID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get file download url")
	return resp
}

func (h *E2EHelper) DownloadFileContent(token, url string) string {
	h.t.Helper()
	downloadURL := url
	if len(downloadURL) > 0 && downloadURL[0] == '/' {
		downloadURL = h.baseURL + downloadURL
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
	require.NoError(h.t, err)
	require.NoError(h.t, WithBearerToken(token)(context.Background(), req))
	rsp, err := http.DefaultClient.Do(req)
	require.NoError(h.t, err)
	defer rsp.Body.Close()
	require.Equal(h.t, http.StatusOK, rsp.StatusCode)
	body, err := io.ReadAll(rsp.Body)
	require.NoError(h.t, err)
	return string(body)
}
