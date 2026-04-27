package v1

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/oapi-codegen/runtime/types"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxAvatarSize      = 5 << 20 // 5 MB
	avatarCacheControl = "public, max-age=3600"
)

var validAvatarPath = regexp.MustCompile(`^(users|teams)/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/[a-f0-9]+_(full|thumb)\.webp$`)

// (PUT /users/me/avatar).
func (h *Server) PutUsersMeAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.ParseMultipartFormLimit(w, r, maxAvatarSize, maxAvatarSize) {
		return
	}

	var body openapi.PutUsersMeAvatarMultipartBody
	if err := helper.DecodeMultipartForm(r, &body, h.infra.Validator); err != nil {
		h.OnError(w, r, err, "PutUsersMeAvatar", "DecodeMultipartForm")

		return
	}

	reader, ok := helper.OpenAvatarFile(w, r, h.OnError, "PutUsersMeAvatar", &body.File)
	if !ok {
		return
	}

	defer func() { _ = reader.Close() }()

	fullURL, thumbURL, err := h.user.AvatarUC.UploadUserAvatar(
		r.Context(),
		user.ID,
		reader,
		body.File.Filename(),
		body.File.FileSize(),
	)
	if h.OnError(w, r, err, "PutUsersMeAvatar", "UploadUserAvatar") {
		return
	}

	httputil.RenderOK(w, r, response.FromAvatarUpload(fullURL, thumbURL))
}

// (DELETE /users/me/avatar).
func (h *Server) DeleteUsersMeAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	err := h.user.AvatarUC.DeleteUserAvatar(r.Context(), user.ID)
	if h.OnError(w, r, err, "DeleteUsersMeAvatar", "DeleteUserAvatar") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (PUT /teams/me/avatar).
func (h *Server) PutTeamsMeAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	teamID, ok := helper.RequireTeamID(w, r, user, h.OnError, "PutTeamsMeAvatar")
	if !ok {
		return
	}

	if !helper.ParseMultipartFormLimit(w, r, maxAvatarSize, maxAvatarSize) {
		return
	}

	var body openapi.PutTeamsMeAvatarMultipartBody
	if err := helper.DecodeMultipartForm(r, &body, h.infra.Validator); err != nil {
		h.OnError(w, r, err, "PutTeamsMeAvatar", "DecodeMultipartForm")

		return
	}

	reader, ok := helper.OpenAvatarFile(w, r, h.OnError, "PutTeamsMeAvatar", &body.File)
	if !ok {
		return
	}

	defer func() { _ = reader.Close() }()

	fullURL, thumbURL, err := h.user.AvatarUC.UploadTeamAvatar(
		r.Context(),
		teamID,
		user.ID,
		reader,
		body.File.Filename(),
		body.File.FileSize(),
	)
	if h.OnError(w, r, err, "PutTeamsMeAvatar", "UploadTeamAvatar") {
		return
	}

	httputil.RenderOK(w, r, response.FromAvatarUpload(fullURL, thumbURL))
}

// (DELETE /teams/me/avatar).
func (h *Server) DeleteTeamsMeAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	teamID, ok := helper.RequireTeamID(w, r, user, h.OnError, "DeleteTeamsMeAvatar")
	if !ok {
		return
	}

	err := h.user.AvatarUC.DeleteTeamAvatar(r.Context(), teamID, user.ID)
	if h.OnError(w, r, err, "DeleteTeamsMeAvatar", "DeleteTeamAvatar") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (PUT /admin/users/{ID}/avatar).
func (h *Server) PutAdminUsersIDAvatar(w http.ResponseWriter, r *http.Request, ID types.UUID) {
	if !helper.ParseMultipartFormLimit(w, r, maxAvatarSize, maxAvatarSize) {
		return
	}

	var body openapi.PutAdminUsersIDAvatarMultipartBody
	if err := helper.DecodeMultipartForm(r, &body, h.infra.Validator); err != nil {
		h.OnError(w, r, err, "PutAdminUsersIDAvatar", "DecodeMultipartForm")

		return
	}

	reader, ok := helper.OpenAvatarFile(w, r, h.OnError, "PutAdminUsersIDAvatar", &body.File)
	if !ok {
		return
	}

	defer func() { _ = reader.Close() }()

	fullURL, thumbURL, err := h.user.AvatarUC.AdminUploadUserAvatar(
		r.Context(),
		ID,
		reader,
		body.File.Filename(),
		body.File.FileSize(),
	)
	if h.OnError(w, r, err, "PutAdminUsersIDAvatar", "AdminUploadUserAvatar") {
		return
	}

	httputil.RenderOK(w, r, response.FromAvatarUpload(fullURL, thumbURL))
}

// (DELETE /admin/users/{ID}/avatar).
func (h *Server) DeleteAdminUsersIDAvatar(w http.ResponseWriter, r *http.Request, ID types.UUID) {
	err := h.user.AvatarUC.AdminDeleteUserAvatar(r.Context(), ID)
	if h.OnError(w, r, err, "DeleteAdminUsersIDAvatar", "AdminDeleteUserAvatar") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (PUT /admin/teams/{ID}/avatar).
func (h *Server) PutAdminTeamsIDAvatar(w http.ResponseWriter, r *http.Request, ID types.UUID) {
	if !helper.ParseMultipartFormLimit(w, r, maxAvatarSize, maxAvatarSize) {
		return
	}

	var body openapi.PutAdminTeamsIDAvatarMultipartBody
	if err := helper.DecodeMultipartForm(r, &body, h.infra.Validator); err != nil {
		h.OnError(w, r, err, "PutAdminTeamsIDAvatar", "DecodeMultipartForm")

		return
	}

	reader, ok := helper.OpenAvatarFile(w, r, h.OnError, "PutAdminTeamsIDAvatar", &body.File)
	if !ok {
		return
	}

	defer func() { _ = reader.Close() }()

	fullURL, thumbURL, err := h.user.AvatarUC.AdminUploadTeamAvatar(
		r.Context(),
		ID,
		reader,
		body.File.Filename(),
		body.File.FileSize(),
	)
	if h.OnError(w, r, err, "PutAdminTeamsIDAvatar", "AdminUploadTeamAvatar") {
		return
	}

	httputil.RenderOK(w, r, response.FromAvatarUpload(fullURL, thumbURL))
}

// (DELETE /admin/teams/{ID}/avatar).
func (h *Server) DeleteAdminTeamsIDAvatar(w http.ResponseWriter, r *http.Request, ID types.UUID) {
	err := h.user.AvatarUC.AdminDeleteTeamAvatar(r.Context(), ID)
	if h.OnError(w, r, err, "DeleteAdminTeamsIDAvatar", "AdminDeleteTeamAvatar") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// GetAvatarByPath serves stored avatar images at /avatars/*. The path is validated
// against validAvatarPath regex (users|teams/<uuid>/<hash>_(full|thumb).webp) before
// any storage access. Response headers enforce nosniff, a restrictive CSP
// (default-src 'none'), and a 1-hour public cache to reduce storage load.
func (h *Server) GetAvatarByPath(w http.ResponseWriter, r *http.Request, path string) {
	if !validAvatarPath.MatchString(path) {
		h.OnError(w, r, apperr.NewValidationErrorf("invalid avatar path"), "GetAvatarByPath", "PathValidate")

		return
	}

	reader, err := h.infra.StorageProvider.Download(r.Context(), path)
	if h.OnError(w, r, err, "GetAvatarByPath", "Download") {
		return
	}

	defer func() { _ = reader.Close() }()

	// Extract content hash from path: (users|teams)/<uuid>/<hash>_(full|thumb).webp
	// validAvatarPath already guaranteed this structure, so no bounds check needed.
	parts := strings.Split(path, "/")
	fileName := parts[2]
	hash := fileName[:strings.IndexByte(fileName, '_')]

	etag := `"` + hash + `"`
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)

		return
	}

	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", avatarCacheControl)
	w.Header().Set("Vary", "Accept-Encoding")
	w.Header().Set("ETag", etag)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")

	if _, err := io.Copy(w, reader); err != nil {
		h.infra.Logger.WithError(err).Warn("GetAvatarByPath - failed to stream avatar")
	}
}
