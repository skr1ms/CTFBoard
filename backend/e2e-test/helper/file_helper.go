package helper

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
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
	return resp.JSON200.URL
}

func (h *E2EHelper) GetFilesIDDownloadExpectStatus(token, fileID string, expectStatus int) *openapi.GetFilesIDDownloadResponse {
	h.t.Helper()
	resp, err := h.client.GetFilesIDDownloadWithResponse(context.Background(), fileID, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "get file download url")
	return resp
}

func (h *E2EHelper) downloadURLToAbsolute(rawURL string) string {
	if len(rawURL) > 0 && rawURL[0] == '/' {
		return h.baseURL + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err == nil {
		baseParsed, err2 := url.Parse(h.baseURL)
		if err2 == nil {
			parsed.Scheme = baseParsed.Scheme
			parsed.Host = baseParsed.Host
			return parsed.String()
		}
	}
	return rawURL
}

func (h *E2EHelper) GetFileDownloadByURLEXpectStatus(token, rawURL string, expectStatus int) {
	h.t.Helper()
	downloadURL := h.downloadURLToAbsolute(rawURL)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
	require.NoError(h.t, err)
	require.NoError(h.t, WithBearerToken(token)(context.Background(), req))
	rsp, err := http.DefaultClient.Do(req)
	require.NoError(h.t, err)
	defer rsp.Body.Close()
	body, err := io.ReadAll(rsp.Body)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, rsp.StatusCode, body, "GET download by URL")
}

func (h *E2EHelper) DownloadFileContent(token, rawURL string) string {
	h.t.Helper()
	downloadURL := h.downloadURLToAbsolute(rawURL)
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
