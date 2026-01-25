package v1

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	httpMiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/skr1ms/CTFBoard/pkg/logger"
)

type fileRoutes struct {
	fileUC *usecase.FileUseCase
	logger logger.Logger
}

func NewFileRoutes(router chi.Router, fileUC *usecase.FileUseCase, logger logger.Logger, jwtService *jwt.JWTService) {
	routes := fileRoutes{
		fileUC: fileUC,
		logger: logger,
	}

	router.With(httpMiddleware.Auth(jwtService), httpMiddleware.Admin).Post("/admin/challenges/{challengeId}/files", routes.Upload)
	router.With(httpMiddleware.Auth(jwtService), httpMiddleware.Admin).Delete("/admin/files/{id}", routes.Delete)
	router.With(httpMiddleware.Auth(jwtService)).Get("/files/{id}/download", routes.GetDownloadURL)
	router.With(httpMiddleware.Auth(jwtService)).Get("/challenges/{challengeId}/files", routes.GetByChallengeID)

	router.Get("/files/download/*", routes.Download)
}

func (h *fileRoutes) Download(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "*")
	if path == "" {
		httputil.RenderError(w, r, http.StatusBadRequest, "path is required")
		return
	}

	rc, err := h.fileUC.Download(r.Context(), path)
	if err != nil {
		h.logger.WithError(err).Error("http - v1 - file - Download")
		handleError(w, r, err)
		return
	}
	defer func() { _ = rc.Close() }()

	if _, err := io.Copy(w, rc); err != nil {
		h.logger.WithError(err).Error("http - v1 - file - Download - Copy")
	}
}

// @Summary      Upload file to challenge
// @Description  Uploads file attachment to a challenge. Admin only
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        challengeId path     string true  "Challenge ID"
// @Param        file        formData file   true  "File to upload"
// @Param        type        formData string false "File type: challenge or writeup" default(challenge)
// @Success      201 {object} map[string]any
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Router       /admin/challenges/{challengeId}/files [post]
func (h *fileRoutes) Upload(w http.ResponseWriter, r *http.Request) {
	challengeUUID, ok := httputil.ParseUUIDParam(w, r, "challengeId")
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		h.logger.WithError(err).Error("http - v1 - file - Upload - ParseMultipartForm")
		httputil.RenderError(w, r, http.StatusBadRequest, "failed to parse form")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		h.logger.WithError(err).Error("http - v1 - file - Upload - FormFile")
		httputil.RenderError(w, r, http.StatusBadRequest, "file is required")
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

	uploadedFile, err := h.fileUC.Upload(r.Context(), challengeUUID, fileType, handler.Filename, file, handler.Size, contentType)
	if err != nil {
		h.logger.WithError(err).Error("http - v1 - file - Upload - Upload")
		handleError(w, r, err)
		return
	}

	httputil.RenderCreated(w, r, map[string]any{
		"id":       uploadedFile.Id.String(),
		"filename": uploadedFile.Filename,
		"size":     uploadedFile.Size,
		"sha256":   uploadedFile.SHA256,
	})
}

// @Summary      Delete file
// @Description  Deletes file from storage. Admin only
// @Tags         Admin
// @Security     BearerAuth
// @Param        id path string true "File ID"
// @Success      204 "No Content"
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /admin/files/{id} [delete]
func (h *fileRoutes) Delete(w http.ResponseWriter, r *http.Request) {
	fileUUID, ok := httputil.ParseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	err := h.fileUC.Delete(r.Context(), fileUUID)
	if err != nil {
		if errors.Is(err, entityError.ErrFileNotFound) {
			httputil.RenderError(w, r, http.StatusNotFound, "file not found")
			return
		}
		h.logger.WithError(err).Error("http - v1 - file - Delete - Delete")
		handleError(w, r, err)
		return
	}

	httputil.RenderNoContent(w, r)
}

// @Summary      Get download URL
// @Description  Returns presigned URL for file download
// @Tags         Challenges
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "File ID"
// @Success      200 {object} map[string]string
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /files/{id}/download [get]
func (h *fileRoutes) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	fileUUID, ok := httputil.ParseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	url, err := h.fileUC.GetDownloadURL(r.Context(), fileUUID)
	if err != nil {
		if errors.Is(err, entityError.ErrFileNotFound) {
			httputil.RenderError(w, r, http.StatusNotFound, "file not found")
			return
		}
		h.logger.WithError(err).Error("http - v1 - file - GetDownloadURL")
		handleError(w, r, err)
		return
	}

	httputil.RenderOK(w, r, map[string]string{"url": url})
}

// @Summary      Get challenge files
// @Description  Returns list of files attached to a challenge
// @Tags         Challenges
// @Produce      json
// @Security     BearerAuth
// @Param        challengeId path string true "Challenge ID"
// @Param        type query string false "File type: challenge or writeup" default(challenge)
// @Success      200 {array} map[string]any
// @Failure      401 {object} ErrorResponse
// @Router       /challenges/{challengeId}/files [get]
func (h *fileRoutes) GetByChallengeID(w http.ResponseWriter, r *http.Request) {
	challengeUUID, ok := httputil.ParseUUIDParam(w, r, "challengeId")
	if !ok {
		return
	}

	fileTypeStr := r.URL.Query().Get("type")
	fileType := entity.FileTypeChallenge
	if fileTypeStr == "writeup" {
		fileType = entity.FileTypeWriteup
	}

	files, err := h.fileUC.GetByChallengeID(r.Context(), challengeUUID, fileType)
	if err != nil {
		h.logger.WithError(err).Error("http - v1 - file - GetByChallengeID")
		handleError(w, r, err)
		return
	}

	result := make([]map[string]any, 0, len(files))
	for _, f := range files {
		result = append(result, map[string]any{
			"id":         f.Id.String(),
			"filename":   f.Filename,
			"size":       f.Size,
			"created_at": f.CreatedAt,
		})
	}

	httputil.RenderOK(w, r, result)
}
