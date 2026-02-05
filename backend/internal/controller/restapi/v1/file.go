package v1

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Upload file to challenge
// (POST /admin/challenges/{challengeID}/files)
func (h *Server) PostAdminChallengesChallengeIDFiles(w http.ResponseWriter, r *http.Request, challengeID string) {
	challengeuuid, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAdminChallengesChallengeIDFiles - ParseMultipartForm")
		helper.RenderError(w, r, http.StatusBadRequest, "failed to parse form")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostAdminChallengesChallengeIDFiles - FormFile")
		helper.RenderError(w, r, http.StatusBadRequest, "file is required")
		return
	}
	defer func() { _ = file.Close() }()

	fileTypeStr := r.FormValue("type")
	fileType := entity.FileTypeChallenge
	if fileTypeStr == "writeup" {
		fileType = entity.FileTypeWriteup
	}

	contentType := handler.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	uploadedFile, err := h.fileUC.Upload(r.Context(), challengeuuid, fileType, handler.Filename, file, handler.Size, contentType)
	if h.OnError(w, r, err, "PostAdminChallengesChallengeIDFiles", "Upload") {
		return
	}

	helper.RenderCreated(w, r, map[string]any{
		"id":       uploadedFile.ID.String(),
		"filename": uploadedFile.Filename,
		"size":     uploadedFile.Size,
		"sha256":   uploadedFile.SHA256,
	})
}

// Delete file
// (DELETE /admin/files/{ID})
func (h *Server) DeleteAdminFilesID(w http.ResponseWriter, r *http.Request, ID string) {
	fileuuid, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	err := h.fileUC.Delete(r.Context(), fileuuid)
	if err != nil {
		if errors.Is(err, entityError.ErrFileNotFound) {
			helper.RenderError(w, r, http.StatusNotFound, "file not found")
			return
		}
		if h.OnError(w, r, err, "DeleteAdminFilesID", "Delete") {
			return
		}
		return
	}

	helper.RenderNoContent(w, r)
}

// Get download URL
// (GET /files/{ID}/download)
func (h *Server) GetFilesIDDownload(w http.ResponseWriter, r *http.Request, ID string) {
	fileuuid, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	url, err := h.fileUC.GetDownloadURL(r.Context(), fileuuid)
	if err != nil {
		if errors.Is(err, entityError.ErrFileNotFound) {
			helper.RenderError(w, r, http.StatusNotFound, "file not found")
			return
		}
		h.logger.WithError(err).Error("restapi - v1 - GetFilesIDDownload")
		helper.HandleError(w, r, err)
		return
	}

	helper.RenderOK(w, r, map[string]string{"url": url})
}

// Get challenge files
// (GET /challenges/{challengeID}/files)
func (h *Server) GetChallengesChallengeIDFiles(w http.ResponseWriter, r *http.Request, challengeID string, params openapi.GetChallengesChallengeIDFilesParams) {
	challengeuuid, ok := helper.ParseUUID(w, r, challengeID)
	if !ok {
		return
	}

	fileType := entity.FileTypeChallenge
	if params.Type != nil && *params.Type == "writeup" {
		fileType = entity.FileTypeWriteup
	}

	files, err := h.fileUC.GetByChallengeID(r.Context(), challengeuuid, fileType)
	if h.OnError(w, r, err, "GetChallengesChallengeIDFiles", "GetByChallengeID") {
		return
	}

	result := make([]map[string]any, 0, len(files))
	for _, f := range files {
		result = append(result, map[string]any{
			"id":         f.ID.String(),
			"filename":   f.Filename,
			"size":       f.Size,
			"created_at": f.CreatedAt,
		})
	}

	helper.RenderOK(w, r, result)
}

// Download - Not part of OpenAPI interface, manually routed
func (h *Server) Download(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "*")
	if path == "" {
		helper.RenderError(w, r, http.StatusBadRequest, "path is required")
		return
	}

	rc, err := h.fileUC.Download(r.Context(), path)
	if h.OnError(w, r, err, "Download", "Download") {
		return
	}
	defer func() { _ = rc.Close() }()

	if _, err := io.Copy(w, rc); err != nil {
		h.OnError(w, r, err, "Download", "Copy")
	}
}
