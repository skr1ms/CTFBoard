package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /admin/storage).
func (h *Server) GetAdminStorage(w http.ResponseWriter, r *http.Request, params openapi.GetAdminStorageParams) {
	prefix := ""

	if params.Prefix != nil {
		prefix = *params.Prefix
	}

	if err := request.ValidateStoragePrefix(prefix); h.OnError(w, r, err, "GetAdminStorage", "Validate") {
		return
	}

	paths, err := h.admin.StorageAdminUC.List(r.Context(), prefix)
	if h.OnError(w, r, err, "GetAdminStorage", "List") {
		return
	}

	httputil.RenderOK(w, r, response.FromStorageList(paths))
}

// (DELETE /admin/storage/{path}).
func (h *Server) DeleteAdminStoragePath(w http.ResponseWriter, r *http.Request, path string) {
	if err := request.ValidateStoragePath(path); h.OnError(w, r, err, "DeleteAdminStoragePath", "Validate") {
		return
	}

	err := h.admin.StorageAdminUC.Delete(r.Context(), path)
	if h.OnError(w, r, err, "DeleteAdminStoragePath", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}
