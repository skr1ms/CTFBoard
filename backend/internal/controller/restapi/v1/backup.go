package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

// Export competition backup as JSON
// (GET /admin/export)
func (h *Server) GetAdminExport(w http.ResponseWriter, r *http.Request, params openapi.GetAdminExportParams) {
	opts := entity.ExportOptions{
		IncludeUsers:  params.IncludeUsers != nil && *params.IncludeUsers,
		IncludeTeams:  params.IncludeTeams == nil || *params.IncludeTeams,
		IncludeSolves: params.IncludeSolves != nil && *params.IncludeSolves,
		IncludeAwards: params.IncludeAwards == nil || *params.IncludeAwards,
	}

	data, err := h.backupUC.Export(r.Context(), opts)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAdminExport")
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to export backup")
		return
	}

	filename := fmt.Sprintf("ctf-backup-%s.json", time.Now().UTC().Format("20060102T150405Z"))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAdminExport - encode")
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

	rc, err := h.backupUC.ExportZIP(r.Context(), opts)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAdminExportZip")
		httputil.RenderError(w, r, http.StatusInternalServerError, "failed to export backup")
		return
	}
	defer rc.Close()

	filename := fmt.Sprintf("backup-%s.zip", time.Now().UTC().Format("20060102T150405Z"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, rc); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetAdminExportZip - copy")
	}
}

// Import competition backup from ZIP file
// (POST /admin/import)
func (h *Server) PostAdminImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		httputil.RenderError(w, r, http.StatusBadRequest, "failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.RenderError(w, r, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		httputil.RenderError(w, r, http.StatusBadRequest, "failed to read file")
		return
	}

	opts := entity.ImportOptions{
		EraseExisting: r.FormValue("erase_existing") == "true",
		ValidateFiles: r.FormValue("validate_files") == "true",
		ConflictMode:  entity.ConflictMode(r.FormValue("conflict_mode")),
	}
	if opts.ConflictMode == "" {
		opts.ConflictMode = entity.ConflictModeOverwrite
	}

	reader := bytes.NewReader(data)
	result, err := h.backupUC.ImportZIP(r.Context(), reader, header.Size, opts)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAdminImport")
		httputil.RenderError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	httputil.RenderJSON(w, r, http.StatusOK, result)
}
