package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const maxFileUploadSize = 100 << 20 // 100 MB

// (POST /admin/challenges/{challengeID}/files).
func (h *Server) PostAdminChallengesChallengeIDFiles(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	body, ok := helper.DecodeMultipartWithLimit[openapi.PostAdminChallengesChallengeIDFilesMultipartBody](w, r, maxFileUploadSize, maxFileUploadSize, h.infra.Validator, h.OnError, "PostAdminChallengesChallengeIDFiles")
	if !ok {
		return
	}

	if err := request.ValidateChallengeUploadFilename(body.File.Filename()); h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "Filename") {
		return
	}

	fileType, err := request.MultipartFileType(body.Type)
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "Type") {
		return
	}

	reader, ok := helper.OpenMultipartFile(w, r, h.OnError, "PostAdminChallengesChallengeIDFiles", &body.File)
	if !ok {
		return
	}

	defer func() { _ = reader.Close() }()

	fileReader, contentType, err := helper.PrepareChallengeUploadReader(reader, body.File.Filename())
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "PrepareUploadReader") {
		return
	}

	uploadedFile, err := h.challenge.FileUC.Upload(r.Context(), challengeIDParsed, fileType, body.File.Filename(), fileReader, body.File.FileSize(), contentType)
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "Upload") {
		return
	}

	httputil.RenderCreated(w, r, response.FromUploadedFile(uploadedFile))
}

// (DELETE /admin/files/{ID}).
func (h *Server) DeleteAdminFilesID(w http.ResponseWriter, r *http.Request, ID string) {
	fileIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	err := h.challenge.FileUC.Delete(r.Context(), fileIDParsed)
	if h.OnError(w, r, err, "DeleteAdminFilesID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (GET /files/{ID}/download).
func (h *Server) GetFilesIDDownload(w http.ResponseWriter, r *http.Request, ID string) {
	fileIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	// File access control: CTF-time (allow only when competition has started)
	status, err := h.comp.CompetitionUC.GetStatus(r.Context())
	if h.OnError(w, r, err, "GetFilesIDDownload", "GetStatus") {
		return
	}

	if helper.IsCompetitionStatusNotStarted(status) {
		h.OnError(w, r, helper.ErrCompetitionNotStarted, "GetFilesIDDownload", "CompetitionCheck")

		return
	}

	// File access control: banned teams cannot download
	if !helper.CheckOptionalTeamBan(w, r, h.team.ReadUC, user.TeamID, h.OnError, "GetFilesIDDownload") {
		return
	}

	url, err := h.challenge.FileUC.GetDownloadURLWithAccess(r.Context(), fileIDParsed, user.TeamID, helper.IsAdmin(user))
	if h.OnError(w, r, err, "GetFilesIDDownload", "GetDownloadURLWithAccess") {
		return
	}

	httputil.RenderOK(w, r, response.FromFileDownloadURL(url))
}

// (GET /challenges/{challengeID}/files).
func (h *Server) GetChallengesChallengeIDFiles(w http.ResponseWriter, r *http.Request, challengeID string, params openapi.GetChallengesChallengeIDFilesParams) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	fileType, err := request.ChallengeFileTypeFromParams(params.Type)
	if h.OnError(w, r, err, "GetChallengesChallengeIDFiles", "Type") {
		return
	}

	files, err := h.challenge.FileUC.GetByChallengeIDWithAccess(r.Context(), challengeIDParsed, fileType, user.TeamID, helper.IsAdmin(user))
	if h.OnError(w, r, err, "GetChallengesChallengeIDFiles", "GetByChallengeIDWithAccess") {
		return
	}

	httputil.RenderOK(w, r, response.FromFileList(files))
}

// downloadByPathAndToken is the shared download orchestrator used by both the
// chi wildcard route (Download) and the OpenAPI handler (GetFilesDownloadPath).
// Steps: validate path structure and ban status -> verify JWT download token ->
// re-check file access rights -> stream from storage with X-Content-Type-Options:
// nosniff. Content type is sniffed from the first 512 bytes and the stream is
// recomposed so no bytes are lost.
func (h *Server) downloadByPathAndToken(w http.ResponseWriter, r *http.Request, path, token string) {
	if !helper.ValidateDownloadPath(path) {
		h.OnError(w, r, helper.ValidationErrorf("invalid file path"), "Download", "PathValidate")

		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.CheckOptionalTeamBan(w, r, h.team.ReadUC, user.TeamID, h.OnError, "Download") {
		return
	}

	if token == "" {
		h.OnError(w, r, helper.ErrTokenRequired, "Download", "TokenCheck")

		return
	}

	file, err := h.challenge.FileUC.VerifyDownloadTokenAndGetFile(r.Context(), path, token, user.TeamID)
	if h.OnError(w, r, err, "Download", "VerifyDownloadTokenAndGetFile") {
		return
	}

	_, err = h.challenge.FileUC.GetDownloadURLWithAccess(r.Context(), file.ID, user.TeamID, helper.IsAdmin(user))
	if h.OnError(w, r, err, "Download", "AccessCheck") {
		return
	}

	rc, err := h.challenge.FileUC.Download(r.Context(), path)
	if h.OnError(w, r, err, "Download", "Download") {
		return
	}

	defer func() { _ = rc.Close() }()

	filename := file.Filename
	if filename == "" {
		filename = helper.DownloadFilename(path)
	}

	contentType, bodyReader := helper.DetectContentTypeFromReader(filename, rc)

	if err := helper.RenderDownloadStream(w, contentType, filename, bodyReader); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - Download - Copy")
	}
}

// (GET /files/download/*).
func (h *Server) Download(w http.ResponseWriter, r *http.Request) {
	path := helper.DownloadPathFromWildcard(r)
	if path == "" {
		h.OnError(w, r, helper.ValidationErrorf("path is required"), "Download", "PathCheck")

		return
	}

	h.downloadByPathAndToken(w, r, path, helper.DownloadToken(r))
}

// (GET /files/download/{path}).
func (h *Server) GetFilesDownloadPath(w http.ResponseWriter, r *http.Request, path string, params openapi.GetFilesDownloadPathParams) {
	if path == "" {
		h.OnError(w, r, helper.ValidationErrorf("path is required"), "GetFilesDownloadPath", "PathCheck")

		return
	}

	h.downloadByPathAndToken(w, r, path, params.Token)
}
