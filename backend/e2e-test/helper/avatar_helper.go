package helper

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func MakeAvatarPNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func (h *E2EHelper) buildAvatarMultipart(imageData []byte) (*bytes.Buffer, string) {
	h.t.Helper()

	var buf bytes.Buffer

	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", "avatar.png")
	require.NoError(h.t, err)
	_, err = part.Write(imageData)
	require.NoError(h.t, err)
	require.NoError(h.t, w.Close())

	return &buf, w.FormDataContentType()
}

func (h *E2EHelper) UploadUserAvatar(token string, imageData []byte, expectStatus int) *openapi.PutUsersMeAvatarResponse {
	h.t.Helper()
	body, contentType := h.buildAvatarMultipart(imageData)
	resp, err := h.client.PutUsersMeAvatarWithBodyWithResponse(context.Background(), contentType, body, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "upload user avatar")

	return resp
}

func (h *E2EHelper) DeleteUserAvatar(token string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.DeleteUsersMeAvatarWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete user avatar")
}

func (h *E2EHelper) UploadTeamAvatar(token string, imageData []byte, expectStatus int) *openapi.PutTeamsMeAvatarResponse {
	h.t.Helper()
	body, contentType := h.buildAvatarMultipart(imageData)
	resp, err := h.client.PutTeamsMeAvatarWithBodyWithResponse(context.Background(), contentType, body, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "upload team avatar")

	return resp
}

func (h *E2EHelper) DeleteTeamAvatar(token string, expectStatus int) {
	h.t.Helper()
	resp, err := h.client.DeleteTeamsMeAvatarWithResponse(context.Background(), WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "delete team avatar")
}

func (h *E2EHelper) AdminUploadUserAvatar(token, userID string, imageData []byte, expectStatus int) *openapi.PutAdminUsersIDAvatarResponse {
	h.t.Helper()

	id := uuid.MustParse(userID)
	body, contentType := h.buildAvatarMultipart(imageData)
	resp, err := h.client.PutAdminUsersIDAvatarWithBodyWithResponse(context.Background(), id, contentType, body, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin upload user avatar")

	return resp
}

func (h *E2EHelper) AdminDeleteUserAvatar(token, userID string, expectStatus int) {
	h.t.Helper()

	id := uuid.MustParse(userID)
	resp, err := h.client.DeleteAdminUsersIDAvatarWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin delete user avatar")
}

func (h *E2EHelper) AdminUploadTeamAvatar(token, teamID string, imageData []byte, expectStatus int) *openapi.PutAdminTeamsIDAvatarResponse {
	h.t.Helper()

	id := uuid.MustParse(teamID)
	body, contentType := h.buildAvatarMultipart(imageData)
	resp, err := h.client.PutAdminTeamsIDAvatarWithBodyWithResponse(context.Background(), id, contentType, body, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin upload team avatar")

	return resp
}

func (h *E2EHelper) AdminDeleteTeamAvatar(token, teamID string, expectStatus int) {
	h.t.Helper()

	id := uuid.MustParse(teamID)
	resp, err := h.client.DeleteAdminTeamsIDAvatarWithResponse(context.Background(), id, WithBearerToken(token))
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode(), resp.Body, "admin delete team avatar")
}

func (h *E2EHelper) GetAvatarByPath(path string, expectStatus int) {
	h.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, h.baseURL+"/api/v1/avatars/"+path, http.NoBody)
	require.NoError(h.t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(h.t, err)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(h.t, err)
	RequireStatus(h.t, expectStatus, resp.StatusCode, body, "get avatar by path")
}
