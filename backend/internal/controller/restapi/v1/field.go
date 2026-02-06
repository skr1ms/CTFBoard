package v1

import (
	"net/http"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
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
	req, ok := helper.DecodeAndValidate[openapi.RequestCreateFieldRequest](w, r, h.infra.Validator, h.infra.Logger, "PostAdminFields")
	if !ok {
		return
	}
	fieldType := entity.FieldType(req.FieldType)
	entityType := entity.EntityType(req.EntityType)
	required := false
	if req.Required != nil {
		required = *req.Required
	}
	var options []string
	if req.Options != nil {
		options = *req.Options
	}
	orderIndex := 0
	if req.OrderIndex != nil {
		orderIndex = *req.OrderIndex
	}
	field, err := h.admin.FieldUC.Create(r.Context(), req.Name, fieldType, entityType, required, options, orderIndex)
	if h.OnError(w, r, err, "PostAdminFields", "Create") {
		return
	}
	helper.RenderCreated(w, r, response.FromField(field))
}

// Update field
// (PUT /admin/fields/{ID})
func (h *Server) PutAdminFieldsID(w http.ResponseWriter, r *http.Request, id string) {
	fieldID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.RequestUpdateFieldRequest](w, r, h.infra.Validator, h.infra.Logger, "PutAdminFieldsID")
	if !ok {
		return
	}
	fieldType := entity.FieldType(req.FieldType)
	required := false
	if req.Required != nil {
		required = *req.Required
	}
	var options []string
	if req.Options != nil {
		options = *req.Options
	}
	orderIndex := 0
	if req.OrderIndex != nil {
		orderIndex = *req.OrderIndex
	}
	field, err := h.admin.FieldUC.Update(r.Context(), fieldID, req.Name, fieldType, required, options, orderIndex)
	if h.OnError(w, r, err, "PutAdminFieldsID", "Update") {
		return
	}
	helper.RenderOK(w, r, response.FromField(field))
}

// Delete field
// (DELETE /admin/fields/{ID})
func (h *Server) DeleteAdminFieldsID(w http.ResponseWriter, r *http.Request, id string) {
	fieldID, ok := helper.ParseUUID(w, r, id)
	if !ok {
		return
	}
	if h.OnError(w, r, h.admin.FieldUC.Delete(r.Context(), fieldID), "DeleteAdminFieldsID", "Delete") {
		return
	}
	helper.RenderNoContent(w, r)
}
