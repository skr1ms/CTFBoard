package v1

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
)

// validPathHexLen is the length of the hex directory prefix in uploaded file paths (e.g. "a3f1...0b/filename.txt").
const (
	validPathHexLen      = 16
	errMsgFileTypeMustBe = `type must be "challenge" or "writeup"`
)

var (
	validLegacyDownloadPathPattern = regexp.MustCompile(fmt.Sprintf(`^[a-f0-9]{%d}/.+$`, validPathHexLen))
	validTasksDownloadPathPattern  = regexp.MustCompile(fmt.Sprintf(`^tasks/[a-f0-9]{%d}/.+$`, validPathHexLen))
)

const maxFileUploadSize = 100 << 20 // 100 MB

// validateDownloadPath rejects path traversal attempts (any ".." component) and
// enforces two valid structural patterns via pre-compiled regex:
// legacy form: "<16-hex-chars>/<filename>" and tasks form: "tasks/<16-hex-chars>/<filename>".
func validateDownloadPath(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}

	return validLegacyDownloadPathPattern.MatchString(path) || validTasksDownloadPathPattern.MatchString(path)
}

func extractFilename(path string) string {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 2 {
		return filepath.Base(parts[1])
	}

	return "download"
}

const (
	detectContentTypePeekSize = 512
	magicBytePeekSize         = 8
)

func detectContentType(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "application/octet-stream"
	}

	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		return "application/octet-stream"
	}

	return contentType
}

// detectContentTypeFromReader peeks up to detectContentTypePeekSize bytes from rc,
// runs http.DetectContentType on the peeked bytes, then reassembles the original
// stream via io.MultiReader so the caller can still read the full content.
// Falls back to extension-based detection when the sniffed type is generic
// "application/octet-stream" or the peek read fails.
func detectContentTypeFromReader(filename string, rc io.Reader) (string, io.Reader) {
	peek := make([]byte, detectContentTypePeekSize)
	n, err := rc.Read(peek)
	peek = peek[:n]
	rest := rc

	if err != nil && err != io.EOF {
		return detectContentType(filename), io.MultiReader(bytes.NewReader(peek), rest)
	}

	if n > 0 {
		detected := http.DetectContentType(peek)
		if detected != "" && detected != "application/octet-stream" {
			return detected, io.MultiReader(bytes.NewReader(peek), rest)
		}
	}

	return detectContentType(filename), io.MultiReader(bytes.NewReader(peek), rest)
}

// prepareUploadReader peeks the first 8 bytes of rc for magic-byte validation,
// reconstructs the full stream via io.MultiReader, and derives the MIME type from filename.
// Returns (nil, "", false) after calling onError if the read fails or magic bytes are blocked.
func (h *Server) prepareUploadReader(w http.ResponseWriter, r *http.Request, rc io.ReadCloser, filename, op string) (io.Reader, string, bool) {
	peek := make([]byte, magicBytePeekSize)

	n, err := io.ReadFull(rc, peek)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		h.OnError(w, r, err, op, "ReadFull")

		return nil, "", false
	}

	if n > 0 && !challenge.ValidateFileMagic(peek[:n]) {
		h.OnError(w, r, apperr.NewValidationErrorf("file type not allowed"), op, "MagicBytes")

		return nil, "", false
	}

	return io.MultiReader(bytes.NewReader(peek[:n]), rc), detectContentType(filename), true
}

// (POST /admin/challenges/{challengeID}/files).
func (h *Server) PostAdminChallengesChallengeIDFiles(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := httputil.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	if !helper.ParseMultipartFormLimit(w, r, maxFileUploadSize, maxFileUploadSize) {
		return
	}

	var body openapi.PostAdminChallengesChallengeIDFilesMultipartBody
	if err := helper.DecodeMultipartForm(r, &body, h.infra.Validator); err != nil {
		h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "DecodeMultipartForm")

		return
	}

	if !helper.RequireMultipartFile(w, r, h.OnError, "PostAdminChallengesChallengeIDFiles", "FormFile", body.File.FileSize()) {
		return
	}

	if !challenge.ValidateUploadFilename(body.File.Filename()) {
		h.OnError(w, r, apperr.NewValidationErrorf("file type not allowed"), "PostAdminChallengesChallengeIDFiles", "Filename")

		return
	}

	fileType := domain.FileTypeChallenge

	if body.Type != nil && *body.Type != "" {
		if err := helper.ValidateMultipartEnum("type", string(*body.Type), []string{string(openapi.PostAdminChallengesChallengeIDFilesMultipartBodyTypeChallenge), string(openapi.PostAdminChallengesChallengeIDFilesMultipartBodyTypeWriteup)}); err != nil {
			h.OnError(w, r, apperr.NewValidationErrorf(errMsgFileTypeMustBe), "PostAdminChallengesChallengeIDFiles", "Type")

			return
		}

		if *body.Type == openapi.PostAdminChallengesChallengeIDFilesMultipartBodyTypeWriteup {
			fileType = domain.FileTypeWriteup
		}
	}

	reader, err := body.File.Reader()
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "OpenFile") {
		return
	}

	defer func() { _ = reader.Close() }()

	fileReader, contentType, ok := h.prepareUploadReader(w, r, reader, body.File.Filename(), "PostAdminChallengesChallengeIDFiles")
	if !ok {
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

	if status == domain.CompetitionStatusNotStarted {
		h.OnError(w, r, apperr.ErrCompetitionNotStarted, "GetFilesIDDownload", "CompetitionCheck")

		return
	}

	// File access control: banned teams cannot download
	if !helper.CheckOptionalTeamBan(w, r, h.team.TeamUC, user.TeamID, h.OnError, "GetFilesIDDownload") {
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

	var fileType domain.FileType

	if params.Type == nil || *params.Type == "challenge" {
		fileType = domain.FileTypeChallenge
	} else if *params.Type == "writeup" {
		fileType = domain.FileTypeWriteup
	} else {
		h.OnError(w, r, apperr.NewValidationErrorf(errMsgFileTypeMustBe), "GetChallengesChallengeIDFiles", "Type")

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
	if !validateDownloadPath(path) {
		h.OnError(w, r, apperr.NewValidationErrorf("invalid file path"), "Download", "PathValidate")

		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.CheckOptionalTeamBan(w, r, h.team.TeamUC, user.TeamID, h.OnError, "Download") {
		return
	}

	if token == "" {
		h.OnError(w, r, apperr.ErrTokenRequired, "Download", "TokenCheck")

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
		filename = extractFilename(path)
	}

	contentType, bodyReader := detectContentTypeFromReader(filename, rc)

	w.Header().Set("X-Content-Type-Options", "nosniff")

	if err := httputil.RenderStream(w, contentType, filename, bodyReader); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - Download - Copy")
	}
}

// (GET /files/download/*).
func (h *Server) Download(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "*")
	if path == "" {
		h.OnError(w, r, apperr.NewValidationErrorf("path is required"), "Download", "PathCheck")

		return
	}

	token := r.URL.Query().Get("token")
	h.downloadByPathAndToken(w, r, path, token)
}

// (GET /files/download/{path}).
func (h *Server) GetFilesDownloadPath(w http.ResponseWriter, r *http.Request, path string, params openapi.GetFilesDownloadPathParams) {
	if path == "" {
		h.OnError(w, r, apperr.NewValidationErrorf("path is required"), "GetFilesDownloadPath", "PathCheck")

		return
	}

	h.downloadByPathAndToken(w, r, path, params.Token)
}
