package v1

import (
	"net/http"

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
	list, err := h.fieldUC.GetByEntityType(r.Context(), entityType)
	if h.OnError(w, r, err, "GetFields", "GetByEntityType") {
		return
	}
	RenderOK(w, r, response.FromFieldList(list))
}

// Create field
// (POST /admin/fields)
func (h *Server) PostAdminFields(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestCreateFieldRequest](w, r, h.validator, h.logger, "PostAdminFields")
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
	field, err := h.fieldUC.Create(r.Context(), req.Name, fieldType, entityType, required, options, orderIndex)
	if h.OnError(w, r, err, "PostAdminFields", "Create") {
		return
	}
	RenderCreated(w, r, response.FromField(field))
}

// Update field
// (PUT /admin/fields/{ID})
func (h *Server) PutAdminFieldsID(w http.ResponseWriter, r *http.Request, id string) {
	fieldID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	req, ok := DecodeAndValidate[openapi.RequestUpdateFieldRequest](w, r, h.validator, h.logger, "PutAdminFieldsID")
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
	field, err := h.fieldUC.Update(r.Context(), fieldID, req.Name, fieldType, required, options, orderIndex)
	if h.OnError(w, r, err, "PutAdminFieldsID", "Update") {
		return
	}
	RenderOK(w, r, response.FromField(field))
}

// Delete field
// (DELETE /admin/fields/{ID})
func (h *Server) DeleteAdminFieldsID(w http.ResponseWriter, r *http.Request, id string) {
	fieldID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	if h.OnError(w, r, h.fieldUC.Delete(r.Context(), fieldID), "DeleteAdminFieldsID", "Delete") {
		return
	}
	RenderNoContent(w, r)
}
