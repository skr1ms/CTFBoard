package v1

import (
	"net/http"
	"strings"

	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /admin/storage).
func (h *Server) GetAdminStorage(w http.ResponseWriter, r *http.Request, params openapi.GetAdminStorageParams) {
	prefix := ""

	if params.Prefix != nil {
		prefix = *params.Prefix
	}

	if prefix != "" && !isValidStoragePath(prefix) {
		h.OnError(w, r, apperr.NewValidationErrorf("invalid prefix"), "GetAdminStorage", "Validate")

		return
	}

	paths, err := h.infra.StorageProvider.List(r.Context(), prefix)
	if h.OnError(w, r, err, "GetAdminStorage", "List") {
		return
	}

	objects := lo.Map(paths, func(p string, _ int) openapi.StorageObjectResponse {
		return openapi.StorageObjectResponse{Path: &p}
	})

	total := len(objects)
	httputil.RenderOK(w, r, openapi.StorageListResponse{Objects: &objects, Total: &total})
}

// (DELETE /admin/storage/{path}).
func (h *Server) DeleteAdminStoragePath(w http.ResponseWriter, r *http.Request, path string) {
	if !isValidStoragePath(path) {
		h.OnError(w, r, apperr.NewValidationErrorf("invalid path"), "DeleteAdminStoragePath", "Validate")

		return
	}

	err := h.infra.StorageProvider.Delete(r.Context(), path)
	if h.OnError(w, r, err, "DeleteAdminStoragePath", "Delete") {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// isValidStoragePath returns true when path is a safe, non-traversal storage key.
// Prevents path-traversal attacks by rejecting any path containing "..".
func isValidStoragePath(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}

	if strings.HasPrefix(path, "/") {
		return false
	}

	return true
}
