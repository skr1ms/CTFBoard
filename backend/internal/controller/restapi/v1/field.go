package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /fields).
func (h *Server) GetFields(w http.ResponseWriter, r *http.Request, params openapi.GetFieldsParams) {
	list, err := h.admin.FieldUC.GetByEntityType(r.Context(), request.FieldEntityTypeFromParams(params.EntityType))
	if h.OnError(w, r, err, "GetFields", "GetByEntityType") {
		return
	}

	setPublicCache(w, cacheStatic, false)
	httputil.RenderOK(w, r, response.FromFieldList(list))
}

// (POST /admin/fields).
func (h *Server) PostAdminFields(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.CreateFieldRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.CreateFieldRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminFields", "CreateFieldRequestToParams") {
		return
	}

	field, err := h.admin.FieldUC.Create(r.Context(), params)
	if h.OnError(w, r, err, "PostAdminFields", "Create") {
		return
	}

	httputil.RenderCreated(w, r, response.FromField(field))
}

// (PUT /admin/fields/{ID}).
func (h *Server) PutAdminFieldsID(w http.ResponseWriter, r *http.Request, ID string) {
	fieldIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateFieldRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	params, err := request.UpdateFieldRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminFieldsID", "UpdateFieldRequestToParams") {
		return
	}

	field, err := h.admin.FieldUC.Update(r.Context(), fieldIDParsed, params)
	if h.OnError(w, r, err, "PutAdminFieldsID", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.FromField(field))
}

// (DELETE /admin/fields/{ID}).
func (h *Server) DeleteAdminFieldsID(w http.ResponseWriter, r *http.Request, ID string) {
	fieldIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.admin.FieldUC.Delete(r.Context(), fieldIDParsed), "DeleteAdminFieldsID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}
