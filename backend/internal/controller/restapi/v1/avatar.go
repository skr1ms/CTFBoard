package v1

import (
	"net/http"

	"github.com/oapi-codegen/runtime/types"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxAvatarSize      = 5 << 20 // 5 MB
	avatarCacheControl = "public, max-age=3600"
)

// (PUT /users/me/avatar).
func (h *Server) PutUsersMeAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	body, ok := helper.DecodeMultipartWithLimit[openapi.PutUsersMeAvatarMultipartBody](w, r, maxAvatarSize, maxAvatarSize, h.infra.Validator, h.OnError, "PutUsersMeAvatar")
	if !ok {
		return
	}

	reader, ok := helper.OpenMultipartFile(w, r, h.OnError, "PutUsersMeAvatar", &body.File)
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

	body, ok := helper.DecodeMultipartWithLimit[openapi.PutTeamsMeAvatarMultipartBody](w, r, maxAvatarSize, maxAvatarSize, h.infra.Validator, h.OnError, "PutTeamsMeAvatar")
	if !ok {
		return
	}

	reader, ok := helper.OpenMultipartFile(w, r, h.OnError, "PutTeamsMeAvatar", &body.File)
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
	body, ok := helper.DecodeMultipartWithLimit[openapi.PutAdminUsersIDAvatarMultipartBody](w, r, maxAvatarSize, maxAvatarSize, h.infra.Validator, h.OnError, "PutAdminUsersIDAvatar")
	if !ok {
		return
	}

	reader, ok := helper.OpenMultipartFile(w, r, h.OnError, "PutAdminUsersIDAvatar", &body.File)
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
	body, ok := helper.DecodeMultipartWithLimit[openapi.PutAdminTeamsIDAvatarMultipartBody](w, r, maxAvatarSize, maxAvatarSize, h.infra.Validator, h.OnError, "PutAdminTeamsIDAvatar")
	if !ok {
		return
	}

	reader, ok := helper.OpenMultipartFile(w, r, h.OnError, "PutAdminTeamsIDAvatar", &body.File)
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
	avatarPath, err := request.ParseAvatarPath(path)
	if h.OnError(w, r, err, "GetAvatarByPath", "PathValidate") {
		return
	}

	reader, err := h.challenge.FileUC.Download(r.Context(), path)
	if h.OnError(w, r, err, "GetAvatarByPath", "Download") {
		return
	}

	defer func() { _ = reader.Close() }()

	if err := helper.RenderCachedImage(w, r, reader, "image/webp", avatarCacheControl, avatarPath.ETag()); err != nil {
		h.infra.Logger.WithError(err).Warn("GetAvatarByPath - failed to stream avatar")
	}
}
