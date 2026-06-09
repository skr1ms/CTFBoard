package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /admin/storage).
func (h *Server) GetAdminStorage(w http.ResponseWriter, r *http.Request, params openapi.GetAdminStorageParams) {
	listParams, err := request.StorageAdminListParams(params)
	if h.OnError(w, r, err, "GetAdminStorage", "Validate") {
		return
	}

	result, err := h.admin.StorageAdminUC.List(r.Context(), listParams)
	if h.OnError(w, r, err, "GetAdminStorage", "List") {
		return
	}

	httputil.RenderOK(w, r, response.FromStorageList(result))
}

// (DELETE /admin/storage).
func (h *Server) DeleteAdminStorage(w http.ResponseWriter, r *http.Request, params openapi.DeleteAdminStorageParams) {
	user, ok := helper.RequireUser(w, r)
	if !ok {
		return
	}

	deleteParams, err := request.StorageAdminDeleteParams(params.Path, user.ID, helper.ClientIP(r))
	if h.OnError(w, r, err, "DeleteAdminStorage", "Validate") {
		return
	}

	err = h.admin.StorageAdminUC.Delete(r.Context(), deleteParams)
	if h.OnError(w, r, err, "DeleteAdminStorage", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}
