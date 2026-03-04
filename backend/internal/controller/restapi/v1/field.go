package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Get custom fields
// (GET /fields)
func (h *Server) GetFields(w http.ResponseWriter, r *http.Request, params openapi.GetFieldsParams) {
	entityType := entity.EntityTypeUser
	if params.EntityType == openapi.Team {
		entityType = entity.EntityTypeTeam
	}
	list, err := h.admin.FieldUC.GetByEntityType(r.Context(), entityType)
	if h.OnError(w, r, err, "GetFields", "GetByEntityType") {
		return
	}
	helper.RenderOK(w, r, response.FromFieldList(list))
}

// Create field
// (POST /admin/fields)
func (h *Server) PostAdminFields(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.CreateFieldRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminFields",
	)
	if !ok {
		return
	}
	name, fieldType, entityType, required, options, orderIndex, err := request.CreateFieldRequestToParams(&req)
	if err != nil {
		h.OnError(w, r, helper.NewValidationErrorf("%s", err.Error()), "PostAdminFields", "CreateFieldRequestToParams")
		return
	}
	field, err := h.admin.FieldUC.Create(r.Context(), name, fieldType, entityType, required, options, orderIndex)
	if h.OnError(w, r, err, "PostAdminFields", "Create") {
		return
	}
	helper.RenderCreated(w, r, response.FromField(field))
}

// Update field
// (PUT /admin/fields/{ID})
func (h *Server) PutAdminFieldsID(w http.ResponseWriter, r *http.Request, ID string) {
	fieldIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.UpdateFieldRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PutAdminFieldsID",
	)
	if !ok {
		return
	}
	name, fieldType, required, options, orderIndex, err := request.UpdateFieldRequestToParams(&req)
	if err != nil {
		h.OnError(w, r, helper.NewValidationErrorf("%s", err.Error()), "PutAdminFieldsID", "UpdateFieldRequestToParams")
		return
	}
	field, err := h.admin.FieldUC.Update(r.Context(), fieldIDParsed, name, fieldType, required, options, orderIndex)
	if h.OnError(w, r, err, "PutAdminFieldsID", "Update") {
		return
	}
	helper.RenderOK(w, r, response.FromField(field))
}

// Delete field
// (DELETE /admin/fields/{ID})
func (h *Server) DeleteAdminFieldsID(w http.ResponseWriter, r *http.Request, ID string) {
	fieldIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	if h.OnError(w, r, h.admin.FieldUC.Delete(r.Context(), fieldIDParsed), "DeleteAdminFieldsID", "Delete") {
		return
	}
	helper.RenderNoContent(w, r)
}
