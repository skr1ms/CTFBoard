package v1

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/go-chi/chi/v5"
)

// validPathHexLen is the length of the hex directory prefix in uploaded file paths (e.g. "a3f1...0b/filename.txt").
const validPathHexLen = 16

var validPathPattern = regexp.MustCompile(fmt.Sprintf(`^[a-f0-9]{%d}/.+$`, validPathHexLen))

const maxFileUploadSize = 100 << 20 // 100 MB

func validateDownloadPath(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}
	return validPathPattern.MatchString(path)
}

func extractFilename(path string) string {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 2 {
		return filepath.Base(parts[1])
	}
	return "download"
}

func sanitizeContentDispositionFilename(name string) string {
	name = filepath.Base(name)
	var b strings.Builder
	for _, r := range name {
		if r < 32 || r == 127 || r == '"' || r == '\\' {
			continue
		}
		b.WriteRune(r)
	}
	s := b.String()
	if s == "" {
		return "download"
	}
	return s
}

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

// Upload file to challenge
// (POST /admin/challenges/{challengeID}/files)
func (h *Server) PostAdminChallengesChallengeIDFiles(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	if !helper.ParseMultipartFormLimit(w, r, maxFileUploadSize) {
		return
	}

	var body openapi.PostAdminChallengesChallengeIDFilesMultipartBody
	helper.DecodeMultipartForm(r, &body)

	if body.File.FileSize() == 0 {
		h.OnError(w, r, helper.NewValidationErrorf("file is required"), "PostAdminChallengesChallengeIDFiles", "FormFile")
		return
	}

	fileType := entity.FileTypeChallenge
	if body.Type != nil {
		switch *body.Type {
		case openapi.Challenge, "":
		case openapi.Writeup:
			fileType = entity.FileTypeWriteup
		default:
			h.OnError(w, r, helper.NewValidationErrorf("type must be %q or %q", entity.FileTypeChallenge, entity.FileTypeWriteup), "PostAdminChallengesChallengeIDFiles", "Type")
			return
		}
	}

	reader, err := body.File.Reader()
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "OpenFile") {
		return
	}
	defer func() { _ = reader.Close() }()

	uploadedFile, err := h.challenge.FileUC.Upload(r.Context(), challengeIDParsed, fileType, body.File.Filename(), reader, body.File.FileSize(), "application/octet-stream")
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "Upload") {
		return
	}

	helper.RenderCreated(w, r, response.FromUploadedFile(uploadedFile))
}

// Delete file
// (DELETE /admin/files/{ID})
func (h *Server) DeleteAdminFilesID(w http.ResponseWriter, r *http.Request, ID string) {
	fileIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	err := h.challenge.FileUC.Delete(r.Context(), fileIDParsed)
	if h.OnError(w, r, err, "DeleteAdminFilesID", "Delete") {
		return
	}

	helper.RenderNoContent(w, r)
}

// Get download URL
// (GET /files/{ID}/download)
func (h *Server) GetFilesIDDownload(w http.ResponseWriter, r *http.Request, ID string) {
	fileIDParsed, ok := helper.ParseUUID(w, r, ID)
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
	if status == entity.CompetitionStatusNotStarted {
		h.OnError(w, r, helper.ErrCompetitionNotStarted, "GetFilesIDDownload", "CompetitionCheck")
		return
	}

	// File access control: banned teams cannot download
	if user.TeamID != nil {
		team, err := h.team.TeamUC.GetByID(r.Context(), *user.TeamID)
		if h.OnError(w, r, err, "GetFilesIDDownload", "GetByID") {
			return
		}
		if team.IsBanned {
			h.OnError(w, r, helper.ErrTeamBanned, "GetFilesIDDownload", "TeamBannedCheck")
			return
		}
	}

	url, err := h.challenge.FileUC.GetDownloadURLWithAccess(r.Context(), fileIDParsed, user.TeamID, user.Role == entity.RoleAdmin)
	if h.OnError(w, r, err, "GetFilesIDDownload", "GetDownloadURLWithAccess") {
		return
	}

	helper.RenderOK(w, r, response.FromFileDownloadURL(url))
}

// Get challenge files
// (GET /challenges/{challengeID}/files)
func (h *Server) GetChallengesChallengeIDFiles(w http.ResponseWriter, r *http.Request, challengeID string, params openapi.GetChallengesChallengeIDFilesParams) {
	challengeIDParsed, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	fileType := entity.FileTypeChallenge
	if params.Type != nil && *params.Type == "writeup" {
		fileType = entity.FileTypeWriteup
	}

	files, err := h.challenge.FileUC.GetByChallengeIDWithAccess(r.Context(), challengeIDParsed, fileType, user.TeamID, user.Role == entity.RoleAdmin)
	if h.OnError(w, r, err, "GetChallengesChallengeIDFiles", "GetByChallengeIDWithAccess") {
		return
	}

	helper.RenderOK(w, r, response.FromFileList(files))
}

// Download streams a file to the client after verifying the download token.
// This handler is manually routed at GET /files/download/* because OpenAPI 3.0
// has no native support for wildcard path segments. The corresponding presigned
// URL endpoint (GetFilesIDDownload) returns a short-lived token that is required
// here as the "token" query parameter.
func (h *Server) Download(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "*")
	if path == "" {
		h.OnError(w, r, helper.NewValidationErrorf("path is required"), "Download", "PathCheck")
		return
	}

	if !validateDownloadPath(path) {
		h.OnError(w, r, helper.NewValidationErrorf("invalid file path"), "Download", "PathValidate")
		return
	}

	// Banned teams cannot download files even with a valid token.
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}
	if user.TeamID != nil {
		team, err := h.team.TeamUC.GetByID(r.Context(), *user.TeamID)
		if h.OnError(w, r, err, "Download", "TeamCheck") {
			return
		}
		if team.IsBanned {
			h.OnError(w, r, helper.ErrTeamBanned, "Download", "BanCheck")
			return
		}
	}

	// Verify download token
	token := r.URL.Query().Get("token")
	if token == "" {
		h.OnError(w, r, helper.ErrTokenRequired, "Download", "TokenCheck")
		return
	}

	file, err := h.challenge.FileUC.VerifyDownloadTokenAndGetFile(r.Context(), path, token)
	if h.OnError(w, r, err, "Download", "VerifyDownloadTokenAndGetFile") {
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
	filename = sanitizeContentDispositionFilename(filename)

	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := helper.RenderStream(w, detectContentType(filename), filename, rc); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - Download - Copy")
	}
}
