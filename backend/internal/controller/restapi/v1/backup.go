package v1

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxBackupZIPSize = 500 << 20 // 500 MB
	maxBackupCSVSize = 50 << 20  // 50 MB
)

// Health check
// (GET /healthcheck)
func (h *Server) GetHealthcheck(w http.ResponseWriter, r *http.Request) {
	helper.RenderOK(w, r, response.FromHealthcheck("ok", "ok"))
}

// Get robots.txt
// (GET /robots.txt)
func (h *Server) GetRobotsTxt(w http.ResponseWriter, r *http.Request) {
	robotsTxt := `User-agent: *
Disallow: /api/
Allow: /
`
	helper.RenderText(w, r, http.StatusOK, "text/plain; charset=utf-8", robotsTxt)
}

// Get Terms of Service
// (GET /tos)
func (h *Server) GetTos(w http.ResponseWriter, r *http.Request) {
	tosContent := `<!DOCTYPE html>
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
	helper.RenderText(w, r, http.StatusOK, "text/html; charset=utf-8", tosContent)
}

// Get Privacy Policy
// (GET /privacy)
func (h *Server) GetPrivacy(w http.ResponseWriter, r *http.Request) {
	privacyContent := `<!DOCTYPE html>
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
	helper.RenderText(w, r, http.StatusOK, "text/html; charset=utf-8", privacyContent)
}

// Get debug information
// (GET /debug)
func (h *Server) GetDebug(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("DEBUG_ENABLED") != "true" {
		h.OnError(w, r, helper.ErrDebugNotEnabled, "GetDebug", "DebugCheck")
		return
	}

	debugInfo := map[string]any{
		"mode":      os.Getenv("CHI_MODE"),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	helper.RenderOK(w, r, debugInfo)
}

// Export competition backup as JSON
// (GET /admin/export)
func (h *Server) GetAdminExport(w http.ResponseWriter, r *http.Request, params openapi.GetAdminExportParams) {
	opts := entity.ExportOptions{
		IncludeUsers:  params.IncludeUsers != nil && *params.IncludeUsers,
		IncludeTeams:  params.IncludeTeams == nil || *params.IncludeTeams,
		IncludeSolves: params.IncludeSolves != nil && *params.IncludeSolves,
		IncludeAwards: params.IncludeAwards == nil || *params.IncludeAwards,
	}

	data, err := h.admin.BackupUC.Export(r.Context(), opts)
	if h.OnError(w, r, err, "GetAdminExport", "Export") {
		return
	}

	filename := fmt.Sprintf("ctf-backup-%s.json", time.Now().UTC().Format("20060102T150405Z"))
	if err := helper.RenderJSONAttachment(w, r, data, filename); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - GetAdminExport - write")
	}
}

// Export competition backup as ZIP archive
// (GET /admin/export/zip)
func (h *Server) GetAdminExportZip(w http.ResponseWriter, r *http.Request, params openapi.GetAdminExportZipParams) {
	includeFiles := params.IncludeFiles == nil || *params.IncludeFiles

	opts := entity.ExportOptions{
		IncludeUsers:  false,
		IncludeTeams:  true,
		IncludeSolves: false,
		IncludeAwards: true,
		IncludeFiles:  includeFiles,
	}

	rc, err := h.admin.BackupUC.ExportZIP(r.Context(), opts)
	if h.OnError(w, r, err, "GetAdminExportZip", "ExportZIP") {
		return
	}
	defer rc.Close()

	filename := fmt.Sprintf("backup-%s.zip", time.Now().UTC().Format("20060102T150405Z"))
	if err := helper.RenderStream(w, "application/zip", filename, rc); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - GetAdminExportZip - write")
	}
}

// Reset competition data
// (POST /admin/reset)
func (h *Server) PostAdminReset(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.AdminResetRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminReset",
	)
	if !ok {
		return
	}

	opts := request.AdminResetRequestToParams(&req)

	err := h.admin.BackupUC.Reset(r.Context(), opts)
	if h.OnError(w, r, err, "PostAdminReset", "Reset") {
		return
	}

	helper.RenderOK(w, r, response.Message("reset completed"))
}

// Import competition backup from ZIP file
// (POST /admin/import)
func (h *Server) PostAdminImport(w http.ResponseWriter, r *http.Request) {
	if !helper.ParseMultipartFormLimit(w, r, maxBackupZIPSize) {
		return
	}

	var body openapi.PostAdminImportMultipartBody
	helper.DecodeMultipartForm(r, &body)

	if body.File.FileSize() == 0 {
		h.OnError(w, r, helper.NewValidationErrorf("file is required"), "PostAdminImport", "FileRequired")
		return
	}

	var cm entity.ConflictMode
	if body.ConflictMode != nil {
		cm = entity.ConflictMode(*body.ConflictMode)
	}
	switch cm {
	case entity.ConflictModeMerge, entity.ConflictModeOverwrite, entity.ConflictModeSkip:
	case "":
		cm = entity.ConflictModeOverwrite
	default:
		h.OnError(w, r, helper.NewValidationErrorf("invalid conflict_mode: allowed values are merge, overwrite, skip"), "PostAdminImport", "ConflictMode")
		return
	}

	data, err := body.File.Bytes()
	if h.OnError(w, r, err, "PostAdminImport", "ReadFile") {
		return
	}

	opts := entity.ImportOptions{
		EraseExisting:      body.EraseExisting != nil && *body.EraseExisting,
		ValidateFiles:      body.ValidateFiles != nil && *body.ValidateFiles,
		ConflictMode:       cm,
		PreserveAdminRoles: body.PreserveAdminRoles != nil && *body.PreserveAdminRoles,
	}

	reader := bytes.NewReader(data)
	result, err := h.admin.BackupUC.ImportZIP(r.Context(), reader, body.File.FileSize(), opts)
	if h.OnError(w, r, err, "PostAdminImport", "ImportZIP") {
		return
	}

	helper.RenderOK(w, r, response.FromImportResult(result))
}

// Export table as CSV
// (GET /admin/export/csv)
func (h *Server) GetAdminExportCsv(w http.ResponseWriter, r *http.Request, params openapi.GetAdminExportCsvParams) {
	if params.Table == "" {
		h.OnError(w, r, helper.NewValidationErrorf("table parameter is required"), "GetAdminExportCsv", "TableRequired")
		return
	}
	allowedTables := map[string]bool{"users": true, "teams": true, "challenges": true, "submissions": true, "solves": true, "awards": true}
	if !allowedTables[string(params.Table)] {
		h.OnError(w, r, helper.NewValidationErrorf("invalid table: allowed values are users, teams, challenges, submissions, solves, awards"), "GetAdminExportCsv", "TableValidate")
		return
	}
	csvData, err := h.admin.BackupUC.ExportCSV(r.Context(), string(params.Table))
	if h.OnError(w, r, err, "GetAdminExportCsv", "ExportCSV") {
		return
	}
	filename := filepath.Base(string(params.Table) + ".csv")
	if filename == "." || filename == "" {
		filename = "export.csv"
	}
	if err := helper.RenderBytes(w, "text/csv; charset=utf-8", filename, csvData); err != nil {
		h.infra.Logger.WithError(err).Error("restapi - v1 - GetAdminExportCsv - write")
	}
}

// Import CSV data
// (POST /admin/import/csv)
func (h *Server) PostAdminImportCsv(w http.ResponseWriter, r *http.Request) {
	if !helper.ParseMultipartFormLimit(w, r, maxBackupCSVSize) {
		return
	}

	var body openapi.PostAdminImportCsvMultipartBody
	helper.DecodeMultipartForm(r, &body)

	if body.File.FileSize() == 0 {
		h.OnError(w, r, helper.NewValidationErrorf("file is required"), "PostAdminImportCsv", "FileRequired")
		return
	}
	if body.Table == "" {
		h.OnError(w, r, helper.NewValidationErrorf("table parameter is required"), "PostAdminImportCsv", "TableRequired")
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
	helper.RenderOK(w, r, response.FromCSVImportResult(result))
}
