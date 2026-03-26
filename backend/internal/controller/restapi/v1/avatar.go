package v1

import (
	"io"
	"net/http"
	"regexp"

	"github.com/oapi-codegen/runtime/types"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const maxAvatarSize = 5 << 20 // 5 MB

var validAvatarPath = regexp.MustCompile(`^(users|teams)/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/[a-f0-9]+_(full|thumb)\.webp$`)

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

	if !helper.RequireMultipartFile(w, r, h.OnError, "PutUsersMeAvatar", "FormFile", body.File.FileSize()) {
		return
	}

	reader, err := body.File.Reader()
	if h.OnError(w, r, err, "PutUsersMeAvatar", "OpenFile") {
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

	httputil.RenderOK(w, r, openapi.AvatarUploadResponse{
		FullURL:  fullURL,
		ThumbURL: thumbURL,
	})
}

func (h *Server) DeleteUsersMeAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	err := h.user.AvatarUC.DeleteUserAvatar(r.Context(), user.ID)
	if h.OnError(w, r, err, "DeleteUsersMeAvatar", "DeleteUserAvatar") {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Server) PutTeamsMeAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if user.TeamID == nil {
		h.OnError(w, r, httperr.ErrUserNotInTeam, "PutTeamsMeAvatar", "TeamIDNil")

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

	if !helper.RequireMultipartFile(w, r, h.OnError, "PutTeamsMeAvatar", "FormFile", body.File.FileSize()) {
		return
	}

	reader, err := body.File.Reader()
	if h.OnError(w, r, err, "PutTeamsMeAvatar", "OpenFile") {
		return
	}

	defer func() { _ = reader.Close() }()

	fullURL, thumbURL, err := h.user.AvatarUC.UploadTeamAvatar(
		r.Context(),
		*user.TeamID,
		user.ID,
		reader,
		body.File.Filename(),
		body.File.FileSize(),
	)
	if h.OnError(w, r, err, "PutTeamsMeAvatar", "UploadTeamAvatar") {
		return
	}

	httputil.RenderOK(w, r, openapi.AvatarUploadResponse{
		FullURL:  fullURL,
		ThumbURL: thumbURL,
	})
}

func (h *Server) DeleteTeamsMeAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if user.TeamID == nil {
		h.OnError(w, r, httperr.ErrUserNotInTeam, "DeleteTeamsMeAvatar", "TeamIDNil")

		return
	}

	err := h.user.AvatarUC.DeleteTeamAvatar(r.Context(), *user.TeamID, user.ID)
	if h.OnError(w, r, err, "DeleteTeamsMeAvatar", "DeleteTeamAvatar") {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Server) PutAdminUsersIDAvatar(w http.ResponseWriter, r *http.Request, ID types.UUID) {
	if !helper.ParseMultipartFormLimit(w, r, maxAvatarSize, maxAvatarSize) {
		return
	}

	var body openapi.PutAdminUsersIDAvatarMultipartBody
	if err := helper.DecodeMultipartForm(r, &body, h.infra.Validator); err != nil {
		h.OnError(w, r, err, "PutAdminUsersIDAvatar", "DecodeMultipartForm")

		return
	}

	if !helper.RequireMultipartFile(w, r, h.OnError, "PutAdminUsersIDAvatar", "FormFile", body.File.FileSize()) {
		return
	}

	reader, err := body.File.Reader()
	if h.OnError(w, r, err, "PutAdminUsersIDAvatar", "OpenFile") {
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

	httputil.RenderOK(w, r, openapi.AvatarUploadResponse{
		FullURL:  fullURL,
		ThumbURL: thumbURL,
	})
}

func (h *Server) DeleteAdminUsersIDAvatar(w http.ResponseWriter, r *http.Request, ID types.UUID) {
	err := h.user.AvatarUC.AdminDeleteUserAvatar(r.Context(), ID)
	if h.OnError(w, r, err, "DeleteAdminUsersIDAvatar", "AdminDeleteUserAvatar") {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Server) PutAdminTeamsIDAvatar(w http.ResponseWriter, r *http.Request, ID types.UUID) {
	if !helper.ParseMultipartFormLimit(w, r, maxAvatarSize, maxAvatarSize) {
		return
	}

	var body openapi.PutAdminTeamsIDAvatarMultipartBody
	if err := helper.DecodeMultipartForm(r, &body, h.infra.Validator); err != nil {
		h.OnError(w, r, err, "PutAdminTeamsIDAvatar", "DecodeMultipartForm")

		return
	}

	if !helper.RequireMultipartFile(w, r, h.OnError, "PutAdminTeamsIDAvatar", "FormFile", body.File.FileSize()) {
		return
	}

	reader, err := body.File.Reader()
	if h.OnError(w, r, err, "PutAdminTeamsIDAvatar", "OpenFile") {
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

	httputil.RenderOK(w, r, openapi.AvatarUploadResponse{
		FullURL:  fullURL,
		ThumbURL: thumbURL,
	})
}

func (h *Server) DeleteAdminTeamsIDAvatar(w http.ResponseWriter, r *http.Request, ID types.UUID) {
	err := h.user.AvatarUC.AdminDeleteTeamAvatar(r.Context(), ID)
	if h.OnError(w, r, err, "DeleteAdminTeamsIDAvatar", "AdminDeleteTeamAvatar") {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Server) GetAvatarByPath(w http.ResponseWriter, r *http.Request, path string) {
	if !validAvatarPath.MatchString(path) {
		h.OnError(w, r, httperr.NewValidationErrorf("invalid avatar path"), "GetAvatarByPath", "PathValidate")

		return
	}

	reader, err := h.infra.StorageProvider.Download(r.Context(), path)
	if h.OnError(w, r, err, "GetAvatarByPath", "Download") {
		return
	}

	defer func() { _ = reader.Close() }()

	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")

	if _, err := io.Copy(w, reader); err != nil {
		h.infra.Logger.WithError(err).Warn("GetAvatarByPath - failed to stream avatar")
	}
}
