package v1

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/wahrwelt-kit/go-httpkit/httputil"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxBackupZIPSize = 500 << 20 // 500 MB
	maxBackupCSVSize = 50 << 20  // 50 MB

	tosHTML = `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Terms of Service</title>
</head>
<body>
	<h1>Terms of Service</h1>
	<p>Please update this content in your application settings.</p>
</body>
</html>
`

	privacyHTML = `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Privacy Policy</title>
</head>
<body>
	<h1>Privacy Policy</h1>
	<p>Please update this content in your application settings.</p>
</body>
</html>
`
)

var allowedExportTables = []string{"users", "teams", "challenges", "submissions", "solves", "awards"}

// (GET /healthcheck).
func (h *Server) GetHealthcheck(w http.ResponseWriter, r *http.Request) {
	httputil.RenderOK(w, r, response.FromHealthcheck("ok", "ok"))
}

// (GET /robots.txt).
func (h *Server) GetRobotsTxt(w http.ResponseWriter, r *http.Request) {
	robotsTxt := `User-agent: *
Disallow: /api/
Allow: /
`

	setPublicCache(w, cacheStatic, false)
	httputil.RenderText(w, r, http.StatusOK, "text/plain; charset=utf-8", robotsTxt)
}

// (GET /tos).
func (h *Server) GetTos(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w, cacheStatic, false)
	httputil.RenderText(w, r, http.StatusOK, "text/html; charset=utf-8", tosHTML)
}

// (GET /privacy).
func (h *Server) GetPrivacy(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w, cacheStatic, false)
	httputil.RenderText(w, r, http.StatusOK, "text/html; charset=utf-8", privacyHTML)
}

// (GET /debug).
func (h *Server) GetDebug(w http.ResponseWriter, r *http.Request) {
	if !h.infra.DebugEnabled {
		h.OnError(w, r, apperr.ErrDebugNotEnabled, "GetDebug", "DebugCheck")

		return
	}

	debugInfo := map[string]any{
		"debug":     true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	httputil.RenderOK(w, r, debugInfo)
}

// (GET /admin/export).
func (h *Server) GetAdminExport(w http.ResponseWriter, r *http.Request, params openapi.GetAdminExportParams) {
	opts := domain.ExportOptions{
		IncludeUsers:       params.IncludeUsers != nil && *params.IncludeUsers,
		IncludeTeams:       params.IncludeTeams == nil || *params.IncludeTeams,
		IncludeSolves:      params.IncludeSolves != nil && *params.IncludeSolves,
		IncludeHintUnlocks: params.IncludeHintUnlocks != nil && *params.IncludeHintUnlocks,
		IncludeAwards:      params.IncludeAwards == nil || *params.IncludeAwards,
	}

	data, err := h.admin.BackupUC.Export(r.Context(), opts)
	if h.OnError(w, r, err, "GetAdminExport", "Export") {
		return
	}

	filename := fmt.Sprintf("ctf-backup-%s.json", time.Now().UTC().Format("20060102T150405Z"))
	if err := httputil.RenderJSONAttachment(w, data, filename); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - GetAdminExport - write")
	}
}

// (GET /admin/export/zip).
func (h *Server) GetAdminExportZip(w http.ResponseWriter, r *http.Request, params openapi.GetAdminExportZipParams) {
	includeFiles := params.IncludeFiles == nil || *params.IncludeFiles

	opts := domain.ExportOptions{
		IncludeUsers:       false,
		IncludeTeams:       true,
		IncludeSolves:      false,
		IncludeHintUnlocks: false,
		IncludeAwards:      true,
		IncludeFiles:       includeFiles,
	}

	rc, err := h.admin.BackupUC.ExportZIP(r.Context(), opts)
	if h.OnError(w, r, err, "GetAdminExportZip", "ExportZIP") {
		return
	}
	defer rc.Close()

	filename := fmt.Sprintf("backup-%s.zip", time.Now().UTC().Format("20060102T150405Z"))
	if err := httputil.RenderStream(w, "application/zip", filename, rc); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - GetAdminExportZip - write")
	}
}

// (POST /admin/reset).
func (h *Server) PostAdminReset(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.AdminResetRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	opts := request.AdminResetRequestToParams(&req)

	err := h.admin.BackupUC.Reset(r.Context(), opts)
	if h.OnError(w, r, err, "PostAdminReset", "Reset") {
		return
	}

	httputil.RenderOK(w, r, response.Message("reset completed"))
}

// PostAdminImport imports a competition backup from an uploaded ZIP file. The
// handler validates the ZIP magic bytes (PK: 0x50 0x4B) before passing to the
// use-case to prevent processing arbitrary binary uploads. It supports three
// conflict modes (merge, overwrite, skip), optional table erasure, file
// validation, and admin-role preservation - the requesting admin's ID and IP
// are recorded in ImportOptions for the audit trail.
// (POST /admin/import).
func (h *Server) PostAdminImport(w http.ResponseWriter, r *http.Request) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	if !helper.ParseMultipartFormLimit(w, r, maxBackupZIPSize, maxBackupZIPSize) {
		return
	}

	var body openapi.PostAdminImportMultipartBody
	if err := helper.DecodeMultipartForm(r, &body, h.infra.Validator); err != nil {
		h.OnError(w, r, err, "PostAdminImport", "DecodeMultipartForm")

		return
	}

	if !helper.RequireMultipartFile(w, r, h.OnError, "PostAdminImport", "FileRequired", body.File.FileSize()) {
		return
	}

	var cm domain.ConflictMode

	if body.ConflictMode != nil {
		cm = domain.ConflictMode(*body.ConflictMode)

		err := helper.ValidateMultipartEnum("conflict_mode", string(cm), []string{"merge", "overwrite", "skip"})
		if err != nil {
			h.OnError(w, r, err, "PostAdminImport", "ConflictMode")

			return
		}
	} else {
		cm = domain.ConflictModeOverwrite
	}

	data, err := body.File.Bytes()
	if h.OnError(w, r, err, "PostAdminImport", "ReadFile") {
		return
	}

	if len(data) < 2 || data[0] != 0x50 || data[1] != 0x4B {
		h.OnError(w, r, apperr.NewValidationErrorf("file must be a ZIP archive"), "PostAdminImport", "MIMECheck")

		return
	}

	opts := domain.ImportOptions{
		EraseExisting:      body.EraseExisting != nil && *body.EraseExisting,
		ValidateFiles:      body.ValidateFiles != nil && *body.ValidateFiles,
		ConflictMode:       cm,
		PreserveAdminRoles: body.PreserveAdminRoles != nil && *body.PreserveAdminRoles,
		AdminUserID:        &user.ID,
		AdminIP:            kitMiddleware.GetClientIPFromContext(r.Context()),
	}

	reader := bytes.NewReader(data)

	result, err := h.admin.BackupUC.ImportZIP(r.Context(), reader, body.File.FileSize(), opts)
	if h.OnError(w, r, err, "PostAdminImport", "ImportZIP") {
		return
	}

	httputil.RenderOK(w, r, response.FromImportResult(result))
}

// (GET /admin/export/csv).
func (h *Server) GetAdminExportCsv(w http.ResponseWriter, r *http.Request, params openapi.GetAdminExportCsvParams) {
	table, ok := httputil.ParseEnumQuery(r, "table", allowedExportTables)
	if !ok {
		h.OnError(w, r, apperr.NewValidationErrorf("invalid table: allowed values are %s", strings.Join(allowedExportTables, ", ")), "GetAdminExportCsv", "TableValidate")

		return
	}

	csvData, err := h.admin.BackupUC.ExportCSV(r.Context(), string(table))
	if h.OnError(w, r, err, "GetAdminExportCsv", "ExportCSV") {
		return
	}

	filename := filepath.Base(string(table) + ".csv")
	if filename == "." || filename == "" {
		filename = "export.csv"
	}

	if err := httputil.RenderBytes(w, "text/csv; charset=utf-8", filename, csvData); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - GetAdminExportCsv - write")
	}
}

// (POST /admin/import/csv).
func (h *Server) PostAdminImportCsv(w http.ResponseWriter, r *http.Request) {
	if !helper.ParseMultipartFormLimit(w, r, maxBackupCSVSize, maxBackupCSVSize) {
		return
	}

	var body openapi.PostAdminImportCsvMultipartBody
	if err := helper.DecodeMultipartForm(r, &body, h.infra.Validator); err != nil {
		h.OnError(w, r, err, "PostAdminImportCsv", "DecodeMultipartForm")

		return
	}

	if !helper.RequireMultipartFile(w, r, h.OnError, "PostAdminImportCsv", "FileRequired", body.File.FileSize()) {
		return
	}

	if body.Table == "" {
		h.OnError(w, r, apperr.NewValidationErrorf("table parameter is required"), "PostAdminImportCsv", "TableRequired")

		return
	}

	if err := helper.ValidateMultipartEnum("table", string(body.Table), allowedExportTables); err != nil {
		h.OnError(w, r, err, "PostAdminImportCsv", "TableValidate")

		return
	}

	data, err := body.File.Bytes()
	if h.OnError(w, r, err, "PostAdminImportCsv", "ReadFile") {
		return
	}

	result, err := h.admin.BackupUC.ImportCSV(r.Context(), string(body.Table), data)
	if h.OnError(w, r, err, "PostAdminImportCsv", "ImportCSV") {
		return
	}

	httputil.RenderOK(w, r, response.FromCSVImportResult(result))
}
