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

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// validPathHexLen is the length of the hex directory prefix in uploaded file paths (e.g. "a3f1...0b/filename.txt").
const validPathHexLen = 16

var validPathPattern = regexp.MustCompile(fmt.Sprintf(`^[a-f0-9]{%d}/.+$`, validPathHexLen))

const maxFileUploadSize = 100 << 20 // 100 MB

var forbiddenUploadExtensions = map[string]struct{}{
	".html": {}, ".htm": {}, ".xhtml": {}, ".svg": {}, ".xml": {},
	".exe": {}, ".bat": {}, ".cmd": {}, ".sh": {}, ".ps1": {},
	".php": {}, ".php3": {}, ".php4": {}, ".php5": {}, ".phtml": {},
	".js": {}, ".mjs": {}, ".cjs": {},
	".vbs": {}, ".wsf": {}, ".jse": {},
	".jar": {}, ".msi": {},
	".py": {}, ".rb": {}, ".pl": {},
}

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

const detectContentTypePeekSize = 512

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

func validateUploadFilename(filename string) bool {
	lower := strings.ToLower(filename)
	parts := strings.Split(lower, ".")
	for i := 1; i < len(parts); i++ {
		ext := "." + parts[i]
		if _, forbidden := forbiddenUploadExtensions[ext]; forbidden {
			return false
		}
	}
	return true
}

var dangerousMagicBytes = [][]byte{
	[]byte("MZ"),
	[]byte("\x7fELF"),
	[]byte("#!"),
	[]byte("%PDF"),
	[]byte("\xd0\xcf\x11\xe0"),
}

func validateFileMagic(header []byte) bool {
	if len(header) == 0 {
		return true
	}
	for _, magic := range dangerousMagicBytes {
		if len(header) >= len(magic) && bytes.Equal(header[:len(magic)], magic) {
			return false
		}
	}
	return true
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
	if err := helper.DecodeMultipartForm(r, &body, h.infra.Validator); err != nil {
		h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "DecodeMultipartForm")
		return
	}
	if !helper.RequireMultipartFile(w, r, h.OnError, "PostAdminChallengesChallengeIDFiles", "FormFile", body.File.FileSize()) {
		return
	}
	if !validateUploadFilename(body.File.Filename()) {
		h.OnError(w, r, helper.NewValidationErrorf("file type not allowed"), "PostAdminChallengesChallengeIDFiles", "Filename")
		return
	}

	fileType := entity.FileTypeChallenge
	if body.Type != nil && *body.Type != "" {
		if err := helper.ValidateMultipartEnum("type", string(*body.Type), []string{string(openapi.PostAdminChallengesChallengeIDFilesMultipartBodyTypeChallenge), string(openapi.PostAdminChallengesChallengeIDFilesMultipartBodyTypeWriteup)}); err != nil {
			h.OnError(w, r, helper.NewValidationErrorf("type must be %q or %q", entity.FileTypeChallenge, entity.FileTypeWriteup), "PostAdminChallengesChallengeIDFiles", "Type")
			return
		}
		if *body.Type == openapi.PostAdminChallengesChallengeIDFilesMultipartBodyTypeWriteup {
			fileType = entity.FileTypeWriteup
		}
	}

	reader, err := body.File.Reader()
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "OpenFile") {
		return
	}
	defer func() { _ = reader.Close() }()

	peek := make([]byte, 8)
	n, err := io.ReadFull(reader, peek)
	if err != nil && !errors.Is(err, io.EOF) {
		h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "ReadFull")
		return
	}
	if n > 0 && !validateFileMagic(peek[:n]) {
		h.OnError(w, r, helper.NewValidationErrorf("file type not allowed"), "PostAdminChallengesChallengeIDFiles", "MagicBytes")
		return
	}
	fileReader := io.MultiReader(bytes.NewReader(peek[:n]), reader)

	contentType := detectContentType(body.File.Filename())
	uploadedFile, err := h.challenge.FileUC.Upload(r.Context(), challengeIDParsed, fileType, body.File.Filename(), fileReader, body.File.FileSize(), contentType)
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

	var fileType entity.FileType
	if params.Type == nil || *params.Type == "challenge" {
		fileType = entity.FileTypeChallenge
	} else if *params.Type == "writeup" {
		fileType = entity.FileTypeWriteup
	} else {
		h.OnError(w, r, helper.NewValidationErrorf("type must be %q or %q", entity.FileTypeChallenge, entity.FileTypeWriteup), "GetChallengesChallengeIDFiles", "Type")
		return
	}

	files, err := h.challenge.FileUC.GetByChallengeIDWithAccess(r.Context(), challengeIDParsed, fileType, user.TeamID, user.Role == entity.RoleAdmin)
	if h.OnError(w, r, err, "GetChallengesChallengeIDFiles", "GetByChallengeIDWithAccess") {
		return
	}

	helper.RenderOK(w, r, response.FromFileList(files))
}

func (h *Server) downloadByPathAndToken(w http.ResponseWriter, r *http.Request, path, token string) {
	if !validateDownloadPath(path) {
		h.OnError(w, r, helper.NewValidationErrorf("invalid file path"), "Download", "PathValidate")
		return
	}
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
	if token == "" {
		h.OnError(w, r, helper.ErrTokenRequired, "Download", "TokenCheck")
		return
	}
	file, err := h.challenge.FileUC.VerifyDownloadTokenAndGetFile(r.Context(), path, token)
	if h.OnError(w, r, err, "Download", "VerifyDownloadTokenAndGetFile") {
		return
	}
	_, err = h.challenge.FileUC.GetDownloadURLWithAccess(r.Context(), file.ID, user.TeamID, user.Role == entity.RoleAdmin)
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
	if err := helper.RenderStream(w, contentType, filename, bodyReader); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - Download - Copy")
	}
}

func (h *Server) Download(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "*")
	if path == "" {
		h.OnError(w, r, helper.NewValidationErrorf("path is required"), "Download", "PathCheck")
		return
	}
	token := r.URL.Query().Get("token")
	h.downloadByPathAndToken(w, r, path, token)
}

func (h *Server) GetFilesDownloadPath(w http.ResponseWriter, r *http.Request, path string, params openapi.GetFilesDownloadPathParams) {
	if path == "" {
		h.OnError(w, r, helper.NewValidationErrorf("path is required"), "GetFilesDownloadPath", "PathCheck")
		return
	}
	h.downloadByPathAndToken(w, r, path, params.Token)
}
